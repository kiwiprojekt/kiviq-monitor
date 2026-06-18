package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func EnsureCA(monitorURL, caDir string) (*x509.CertPool, error) {
	caPath := filepath.Join(caDir, "ca.pem")

	if data, err := os.ReadFile(caPath); err == nil {
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM(data) {
			return pool, nil
		}
	}

	return FetchAndSaveCA(monitorURL, caDir)
}

func FetchAndSaveCA(monitorURL, caDir string) (*x509.CertPool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(monitorURL + "/api/v1/ca")
	if err != nil {
		return nil, fmt.Errorf("fetch CA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CA endpoint returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read CA response: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("failed to parse CA from monitor")
	}

	if err := os.MkdirAll(caDir, 0755); err != nil {
		return nil, fmt.Errorf("create CA dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), data, 0644); err != nil {
		return nil, fmt.Errorf("save CA: %w", err)
	}

	return pool, nil
}
