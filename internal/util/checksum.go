package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256File computes the SHA-256 checksum of a file and returns:
//   - the hex-encoded digest
//   - the file size in bytes
func SHA256File(path string) (sum string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
