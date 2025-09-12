package azure

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/retry"
)

// ensureContainer checks access using a minimal list (SAS sr=c cannot create containers).
func (p *AzureProvider) ensureContainer(ctx context.Context) error {
	start := time.Now()
	attempt := 0
	ensureOnce := func(ctx context.Context) error {
		attempt++
		log.Debug().Str("action", "azure_container_check").Str("container", p.container).
			Int("attempt", attempt).Msg("starting attempt")

		pager := p.client.NewListBlobsFlatPager(p.container, &azblob.ListBlobsFlatOptions{
			MaxResults: to.Ptr(int32(1)),
		})
		if pager.More() {
			_, err := pager.NextPage(ctx)
			if err == nil {
				log.Debug().Str("action", "azure_container_check").Str("container", p.container).
					Int("attempt", attempt).Msg("attempt succeeded")
				return nil
			}
			var re *azcore.ResponseError
			if errors.As(err, &re) {
				switch re.ErrorCode {
				case string(bloberror.ContainerNotFound):
					return fmt.Errorf("container %q not found: create it first (container SAS cannot create containers)", p.container)
				case string(bloberror.AuthorizationFailure),
					string(bloberror.AuthorizationPermissionMismatch),
					string(bloberror.AuthenticationFailed):
					return fmt.Errorf("not authorized for container %q; ensure a container SAS with at least rwl", p.container)
				}
			}
			log.Debug().Err(err).Str("action", "azure_container_check").Str("container", p.container).
				Int("attempt", attempt).Msg("attempt failed")
			return err
		}
		return nil
	}
	if err := retry.Do(ctx, p.ro, p.isAzRetryable, ensureOnce); err != nil {
		return err
	}
	log.Debug().Str("action", "azure_container_check").Str("container", p.container).
		Int("attempts", attempt).Dur("elapsed_ms", time.Since(start)).Msg("container access OK")
	return nil
}

// validateSizeByList finds the exact blob and returns (found, size).
func (p *AzureProvider) validateSizeByList(ctx context.Context, exactKey string) (bool, int64, error) {
	pager := p.client.NewListBlobsFlatPager(p.container, &azblob.ListBlobsFlatOptions{
		Prefix:     to.Ptr(exactKey),
		MaxResults: to.Ptr(int32(1)),
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, 0, err
		}
		for _, it := range page.Segment.BlobItems {
			if it.Name != nil && *it.Name == exactKey {
				if it.Properties != nil && it.Properties.ContentLength != nil {
					return true, *it.Properties.ContentLength, nil
				}
				return true, 0, nil
			}
		}
	}
	return false, 0, nil
}

// isAzRetryable: retry rules for Azure (timeout, 5xx, 429, 408, ServerBusy).
func (p *AzureProvider) isAzRetryable(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var re *azcore.ResponseError
	if errors.As(err, &re) {
		if re.StatusCode == http.StatusTooManyRequests || re.StatusCode == http.StatusRequestTimeout {
			return true
		}
		if re.StatusCode >= 500 && re.StatusCode <= 599 {
			return true
		}
		if re.ErrorCode == string(bloberror.ServerBusy) {
			return true
		}
	}
	return false
}
