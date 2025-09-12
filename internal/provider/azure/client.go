package azure

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/provider"
)

// Build client from config and capture endpoint/SAS for HEAD validation.
// Priority: 1) SAS  2) Service Principal  3) DefaultAzureCredential.
func newClientFromConfig(c config.Config) (*azblob.Client, string, string, bool, error) {
	endpoint := os.Getenv("AZURE_BLOB_ENDPOINT")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.blob.core.windows.net/", c.Azure.Account)
	}

	// 1) SAS
	if sasRaw := strings.TrimSpace(c.Azure.SASToken); sasRaw != "" {
		sas := strings.TrimPrefix(sasRaw, "?")
		url := endpoint + "?" + sas
		cl, err := azblob.NewClientWithNoCredential(url, nil)
		return cl, endpoint, sas, true, err
	}

	// 2) Service Principal
	if c.Azure.ClientID != "" && c.Azure.ClientSecret != "" && c.Azure.TenantID != "" {
		cred, err := azidentity.NewClientSecretCredential(
			c.Azure.TenantID, c.Azure.ClientID, c.Azure.ClientSecret, nil,
		)
		if err != nil {
			return nil, "", "", false, err
		}
		cl, err := azblob.NewClient(endpoint, cred, nil)
		return cl, endpoint, "", false, err
	}

	// 3) Managed Identity / DefaultAzureCredential
	defCred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, "", "", false, err
	}
	cl, err := azblob.NewClient(endpoint, defCred, nil)
	return cl, endpoint, "", false, err
}

func init() {
	provider.Register("azure", func(cfg any) (provider.Provider, error) {
		c, ok := cfg.(config.Config)
		if !ok {
			return nil, fmt.Errorf("azure: invalid config type")
		}
		client, endpoint, sas, viaSAS, err := newClientFromConfig(c)
		if err != nil {
			return nil, err
		}
		return &AzureProvider{
			client:     client,
			account:    c.Azure.Account,
			container:  c.Azure.Container,
			endpoint:   endpoint,
			sas:        sas,
			authViaSAS: viaSAS,
			ro:         c.RetryOptions(),
		}, nil
	})
}
