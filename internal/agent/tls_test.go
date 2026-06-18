package agent

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateTestCA(t *testing.T) []byte {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func TestFetchAndSaveCA(t *testing.T) {
	caPEM := generateTestCA(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ca" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write(caPEM)
	}))
	defer ts.Close()

	dir := t.TempDir()
	pool, err := FetchAndSaveCA(ts.URL, dir)
	if err != nil {
		t.Fatalf("FetchAndSaveCA: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}

	saved, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	if err != nil {
		t.Fatalf("ca.pem not saved: %v", err)
	}
	if string(saved) != string(caPEM) {
		t.Error("saved CA does not match")
	}
}

func TestFetchAndSaveCA_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	dir := t.TempDir()
	_, err := FetchAndSaveCA(ts.URL, dir)
	if err == nil {
		t.Error("expected error for agent error")
	}
}

func TestFetchAndSaveCA_Unreachable(t *testing.T) {
	dir := t.TempDir()
	_, err := FetchAndSaveCA("http://192.0.2.1:1", dir)
	if err == nil {
		t.Error("expected error for unreachable agent")
	}
}

func TestEnsureCA_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	caPEM := generateTestCA(t)
	os.WriteFile(filepath.Join(dir, "ca.pem"), caPEM, 0644)

	pool, err := EnsureCA("http://unused:8080", dir)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}
}

func TestEnsureCA_FetchFromMonitor(t *testing.T) {
	caPEM := generateTestCA(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(caPEM)
	}))
	defer ts.Close()

	dir := t.TempDir()
	pool, err := EnsureCA(ts.URL, dir)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}

	if _, err := os.Stat(filepath.Join(dir, "ca.pem")); os.IsNotExist(err) {
		t.Error("ca.pem not saved after fetch")
	}
}

func TestEnsureCA_PersistsAcrossCalls(t *testing.T) {
	caPEM := generateTestCA(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(caPEM)
	}))
	defer ts.Close()

	dir := t.TempDir()

	pool1, err := EnsureCA(ts.URL, dir)
	if err != nil {
		t.Fatalf("first EnsureCA: %v", err)
	}

	ts.Close()

	pool2, err := EnsureCA("http://192.0.2.1:1", dir)
	if err != nil {
		t.Fatalf("second EnsureCA: %v", err)
	}

	if pool1 == nil || pool2 == nil {
		t.Fatal("pools are nil")
	}
}
