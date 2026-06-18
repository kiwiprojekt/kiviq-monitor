package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/michal/kiviq/internal/monitor"
)

func main() {
	configPath := os.Getenv("KIVIQ_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}

	cfg := monitor.LoadConfig(configPath)

	if err := cfg.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	cfg.SeedAgent()

	historyPoints := 0
	if v := os.Getenv("KIVIQ_HISTORY_POINTS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatalf("Invalid KIVIQ_HISTORY_POINTS %q: must be a positive integer", v)
		}
		historyPoints = n
	}

	store := monitor.NewStore(historyPoints)
	hub := monitor.NewHub()
	go hub.Run()

	mux := monitor.NewServer(store, hub, cfg)

	configDir := filepath.Dir(configPath)
	if err := monitor.EnsureCerts(configDir); err != nil {
		log.Fatalf("Failed to ensure TLS certs: %v", err)
	}

	certFile := filepath.Join(configDir, "cert.pem")
	keyFile := filepath.Join(configDir, "key.pem")

	port := cfg.GetPort()

	// KIVIQ_HTTP serves plain HTTP instead of HTTPS. This exists for fronting
	// proxies that terminate TLS themselves (notably Home Assistant ingress,
	// which connects to the add-on over HTTP on an internal network). Do not
	// enable it on a port reachable by untrusted clients — agent tokens and
	// the dashboard password would travel in clear text.
	if os.Getenv("KIVIQ_HTTP") != "" {
		log.Printf("Monitor starting on :%s (HTTP)", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Printf("Monitor starting on :%s (HTTPS)", port)
	if err := http.ListenAndServeTLS(":"+port, certFile, keyFile, mux); err != nil {
		log.Fatal(err)
	}
}
