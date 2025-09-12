package azure

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// headSizeAndSHA does a direct HEAD (SAS) to read Content-Length and x-ms-meta-sha256.
func (p *AzureProvider) headSizeAndSHA(ctx context.Context, key string) (int64, string, error) {
	base := p.endpoint
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	path := p.container + "/" + normalizeKey(key)
	url := base + path + "?" + p.sas

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, http.NoBody)
	if err != nil {
		return 0, "", err
	}
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("HEAD %s: %s", url, resp.Status)
	}

	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		return 0, "", fmt.Errorf("missing Content-Length")
	}
	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("parse Content-Length: %w", err)
	}
	sha := resp.Header.Get("x-ms-meta-sha256")
	return n, sha, nil
}
