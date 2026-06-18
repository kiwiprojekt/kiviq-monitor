package agent

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

func Run() {
	monitorURL := os.Getenv("MONITOR_URL")
	token := os.Getenv("AGENT_TOKEN")
	intervalStr := os.Getenv("REPORT_INTERVAL")
	caDir := os.Getenv("AGENT_CA_DIR")
	if caDir == "" {
		caDir = "/data"
	}

	interval := 1 * time.Second
	if intervalStr != "" {
		d, err := time.ParseDuration(intervalStr + "s")
		if err != nil || d <= 0 {
			log.Fatalf("Invalid REPORT_INTERVAL %q: must be a positive number of seconds", intervalStr)
		}
		interval = d
	}

	if monitorURL == "" {
		log.Fatal("MONITOR_URL is required")
	}
	if token == "" {
		log.Fatal("AGENT_TOKEN is required")
	}

	caPool, err := EnsureCA(monitorURL, caDir)
	if err != nil {
		log.Fatalf("Failed to get CA: %v", err)
	}

	log.Printf("Agent starting — monitor=%s interval=%s", monitorURL, interval)

	dockerClient, err := shared.NewDockerClient()
	if err != nil {
		log.Printf("Docker not available: %v", err)
	}

	httpClient := newHTTPClient(caPool)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		report, err := CollectStats(dockerClient)
		if err != nil {
			log.Printf("Collect error: %v", err)
			continue
		}
		sendReport(httpClient, monitorURL, token, report)
	}
}

// newHTTPClient builds a client that pins the monitor's CA but ignores the
// certificate's hostname/SAN. The monitor mints a self-signed cert for whatever
// hostname it happens to have, yet agents reach it by many addresses (localhost,
// a LAN IP, a VPN name). Verifying the chain against the pinned CA still blocks
// MITM after the CA has been fetched; only the address check is dropped.
func newHTTPClient(caPool *x509.CertPool) *http.Client {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // default verification off; replaced by VerifyConnection below
		VerifyConnection: func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{Roots: caPool, Intermediates: x509.NewCertPool()}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		},
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &http.Client{Timeout: 10 * time.Second, Transport: transport}
}

func sendReport(httpClient *http.Client, monitorURL, token string, report *shared.ReportRequest) {
	data, err := json.Marshal(report)
	if err != nil {
		log.Printf("Marshal error: %v", err)
		return
	}

	req, err := http.NewRequest("POST", monitorURL+"/api/v1/report", bytes.NewReader(data))
	if err != nil {
		log.Printf("Request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Monitor returned %d", resp.StatusCode)
	}
}
