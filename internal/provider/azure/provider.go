package azure

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/retry"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/util"
)

type AzureProvider struct {
	client     *azblob.Client
	account    string
	container  string
	endpoint   string // e.g. https://<account>.blob.core.windows.net/
	sas        string // raw SAS without leading "?"
	authViaSAS bool
	ro         retry.Options
}

func (p *AzureProvider) Name() string { return "azure" }

// Backup uploads file and validates it (HEAD with SAS, list otherwise).
func (p *AzureProvider) Backup(ctx context.Context, source, target string) error {
	if err := p.ensureContainer(ctx); err != nil {
		return fmt.Errorf("ensure container: %w", err)
	}
	key := normalizeKey(target)

	sum, size, err := util.SHA256File(source)
	if err != nil {
		return fmt.Errorf("checksum: %w", err)
	}

	upStart := time.Now()
	upAttempt := 0
	uploadOnce := func(ctx context.Context) error {
		upAttempt++
		log.Debug().
			Str("action", "azure_upload").
			Str("container", p.container).
			Str("key", key).
			Int("attempt", upAttempt).
			Msg("starting attempt")

		f, err := os.Open(source)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				log.Warn().
					Err(cerr).
					Str("file", source).
					Msg("failed to close source file after upload")
			}
		}()
		_, err = p.client.UploadFile(ctx, p.container, key, f, &azblob.UploadFileOptions{
			Metadata: map[string]*string{"sha256": to.Ptr(sum)},
		})
		if err != nil {
			log.Debug().Err(err).Str("action", "azure_upload").Str("container", p.container).Str("key", key).
				Int("attempt", upAttempt).Msg("attempt failed")
			return err
		}

		log.Debug().Str("action", "azure_upload").Str("container", p.container).Str("key", key).
			Int("attempt", upAttempt).Msg("attempt succeeded")
		return nil
	}
	if err := retry.Do(ctx, p.ro, p.isAzRetryable, uploadOnce); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	log.Info().Str("action", "azure_upload").Str("container", p.container).Str("key", key).
		Int("attempts", upAttempt).Dur("elapsed_ms", time.Since(upStart)).Msg("upload OK")

	// Post-upload validation.
	if p.authViaSAS {
		headStart := time.Now()
		headAttempt := 0
		headOnce := func(ctx context.Context) error {
			headAttempt++
			log.Debug().Str("action", "azure_head").Str("container", p.container).Str("key", key).
				Int("attempt", headAttempt).Msg("starting attempt")

			remoteSize, remoteSHA, err := p.headSizeAndSHA(ctx, key)
			if err != nil {
				log.Debug().Err(err).Str("action", "azure_head").Str("container", p.container).Str("key", key).
					Int("attempt", headAttempt).Msg("attempt failed")
				return err
			}
			if remoteSize != size {
				return fmt.Errorf("size mismatch: local=%d, remote=%d", size, remoteSize)
			}
			if remoteSHA == "" {
				return fmt.Errorf("missing metadata: sha256")
			}
			if remoteSHA != sum {
				return fmt.Errorf("sha256 mismatch: local=%s, remote=%s", sum, remoteSHA)
			}

			log.Debug().Str("action", "azure_head").Str("container", p.container).Str("key", key).
				Int("attempt", headAttempt).Int64("remote_size", remoteSize).Msg("attempt succeeded")
			return nil
		}
		if err := retry.Do(ctx, p.ro, p.isAzRetryable, headOnce); err != nil {
			return fmt.Errorf("validate (head): %w", err)
		}
		log.Info().Str("action", "azure_head").Str("container", p.container).Str("key", key).
			Int("attempts", headAttempt).Dur("elapsed_ms", time.Since(headStart)).
			Msg("validation OK (sha256 & size)")
	} else {
		listStart := time.Now()
		listAttempt := 0
		validateOnce := func(ctx context.Context) error {
			listAttempt++
			log.Debug().Str("action", "azure_list_validate").Str("container", p.container).Str("key", key).
				Int("attempt", listAttempt).Msg("starting attempt")

			found, remoteSize, err := p.validateSizeByList(ctx, key)
			if err != nil {
				log.Debug().Err(err).Str("action", "azure_list_validate").Str("container", p.container).Str("key", key).
					Int("attempt", listAttempt).Msg("attempt failed")
				return err
			}
			if !found {
				return fmt.Errorf("uploaded blob not found at %q", key)
			}
			if remoteSize != size {
				return fmt.Errorf("size mismatch: local=%d, remote=%d", size, remoteSize)
			}

			log.Debug().Str("action", "azure_list_validate").Str("container", p.container).Str("key", key).
				Int("attempt", listAttempt).Int64("remote_size", remoteSize).Msg("attempt succeeded")
			return nil
		}
		if err := retry.Do(ctx, p.ro, p.isAzRetryable, validateOnce); err != nil {
			return fmt.Errorf("validate (list): %w", err)
		}
		log.Info().Str("action", "azure_list_validate").Str("container", p.container).Str("key", key).
			Int("attempts", listAttempt).Dur("elapsed_ms", time.Since(listStart)).Msg("validation OK (size)")
	}

	return nil
}

// Restore downloads a blob to a local path with retries.
func (p *AzureProvider) Restore(ctx context.Context, source, target string) error {
	key := normalizeKey(source)

	dlStart := time.Now()
	dlAttempt := 0
	downloadOnce := func(ctx context.Context) error {
		dlAttempt++
		log.Debug().Str("action", "azure_download").Str("container", p.container).Str("key", key).
			Str("local", target).Int("attempt", dlAttempt).Msg("starting attempt")

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := out.Close(); cerr != nil {
				log.Warn().
					Err(cerr).
					Str("file", target).
					Msg("failed to close local file after download")
			}
		}()
		_, err = p.client.DownloadFile(ctx, p.container, key, out, nil)
		if err != nil {
			log.Debug().Err(err).Str("action", "azure_download").Str("container", p.container).Str("key", key).
				Int("attempt", dlAttempt).Msg("attempt failed")
			return err
		}

		log.Debug().Str("action", "azure_download").Str("container", p.container).Str("key", key).
			Int("attempt", dlAttempt).Msg("attempt succeeded")
		return nil
	}
	if err := retry.Do(ctx, p.ro, p.isAzRetryable, downloadOnce); err != nil {
		return err
	}
	log.Info().Str("action", "azure_download").Str("container", p.container).Str("key", key).
		Str("local", target).Int("attempts", dlAttempt).Dur("elapsed_ms", time.Since(dlStart)).Msg("download OK")
	return nil
}

func normalizeKey(k string) string {
	return strings.TrimPrefix(k, "/")
}
