package monitor

import (
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCerts_GeneratesNewCerts(t *testing.T) {
	dir := t.TempDir()

	err := EnsureCerts(dir)
	if err != nil {
		t.Fatalf("EnsureCerts: %v", err)
	}

	for _, name := range []string{"cert.pem", "key.pem", "ca.pem"} {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", name)
		}
	}

	certPEM, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	keyPEM, _ := os.ReadFile(filepath.Join(dir, "key.pem"))
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("invalid TLS key pair: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Error("cert has no certificate chain")
	}
}

func TestEnsureCerts_LoadsExistingCerts(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureCerts(dir); err != nil {
		t.Fatalf("first EnsureCerts: %v", err)
	}

	cert1, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	key1, _ := os.ReadFile(filepath.Join(dir, "key.pem"))
	ca1, _ := os.ReadFile(filepath.Join(dir, "ca.pem"))

	if err := EnsureCerts(dir); err != nil {
		t.Fatalf("second EnsureCerts: %v", err)
	}

	cert2, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	key2, _ := os.ReadFile(filepath.Join(dir, "key.pem"))
	ca2, _ := os.ReadFile(filepath.Join(dir, "ca.pem"))

	if string(cert1) != string(cert2) {
		t.Error("cert changed on second load")
	}
	if string(key1) != string(key2) {
		t.Error("key changed on second load")
	}
	if string(ca1) != string(ca2) {
		t.Error("ca changed on second load")
	}
}

func TestEnsureCerts_CertSignedByCA(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureCerts(dir); err != nil {
		t.Fatalf("EnsureCerts: %v", err)
	}

	certPEM, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	caPEM, _ := os.ReadFile(filepath.Join(dir, "ca.pem"))

	caBlock, _ := pem.Decode(caPEM)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("parse CA: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Errorf("cert not signed by CA: %v", err)
	}
}

func TestEnsureCerts_CertHasSAN(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureCerts(dir); err != nil {
		t.Fatalf("EnsureCerts: %v", err)
	}

	certPEM, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "kiviq"
	}

	if len(cert.DNSNames) == 0 {
		t.Error("cert has no SAN DNS names")
	}
	found := false
	for _, name := range cert.DNSNames {
		if name == hostname {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("cert DNSNames %v does not contain hostname %q", cert.DNSNames, hostname)
	}
}

func TestEnsureCerts_CleansUpOnPartialFailure(t *testing.T) {
	dir := t.TempDir()

	// Write a cert.pem so EnsureCerts thinks certs exist partially
	os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("partial"), 0644)

	// EnsureCerts should detect partial state and regenerate
	err := EnsureCerts(dir)
	if err != nil {
		t.Fatalf("EnsureCerts should handle partial state: %v", err)
	}

	// All three files should now exist and be valid
	for _, name := range []string{"cert.pem", "key.pem", "ca.pem"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("missing %s: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestConfigDir(t *testing.T) {
	cfg := &Config{path: "/some/path/config.json"}
	if got := cfg.ConfigDir(); got != "/some/path" {
		t.Errorf("ConfigDir() = %q, want %q", got, "/some/path")
	}

	cfg2 := &Config{path: "config.json"}
	if got := cfg2.ConfigDir(); got != "." {
		t.Errorf("ConfigDir() = %q, want %q", got, ".")
	}
}

func TestEnsureCerts_InvalidDir(t *testing.T) {
	err := EnsureCerts("/nonexistent/dir/that/does/not/exist")
	if err == nil {
		t.Error("expected error for invalid dir")
	}
}

func TestGenerateCA(t *testing.T) {
	caCertPEM, caKeyPEM, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	if len(caCertPEM) == 0 {
		t.Fatal("caCertPEM is empty")
	}
	if len(caKeyPEM) == 0 {
		t.Fatal("caKeyPEM is empty")
	}

	block, _ := pem.Decode(caCertPEM)
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	if !parsed.IsCA {
		t.Error("cert is not a CA")
	}
	if parsed.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA cert does not have CertSign key usage")
	}
}

func TestGenerateCert(t *testing.T) {
	caCertPEM, caKeyPEM, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	certPEM, keyPEM, err := GenerateCert(caCertPEM, caKeyPEM, "test-host")
	if err != nil {
		t.Fatalf("GenerateCert: %v", err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatal("empty cert or key PEM")
	}

	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	if cert.Subject.CommonName != "test-host" {
		t.Errorf("CommonName = %q, want %q", cert.Subject.CommonName, "test-host")
	}
	if cert.IsCA {
		t.Error("agent cert should not be CA")
	}
	if len(cert.DNSNames) == 0 {
		t.Error("cert has no SAN DNS names")
	}

	// Verify signed by CA
	caBlock, _ := pem.Decode(caCertPEM)
	caCert, _ := x509.ParseCertificate(caBlock.Bytes)
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Errorf("cert not signed by CA: %v", err)
	}
}

func TestGenerateCert_DifferentHosts(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCA()

	cert1, _, _ := GenerateCert(caCertPEM, caKeyPEM, "host-a")
	cert2, _, _ := GenerateCert(caCertPEM, caKeyPEM, "host-b")

	block1, _ := pem.Decode(cert1)
	parsed1, _ := x509.ParseCertificate(block1.Bytes)
	block2, _ := pem.Decode(cert2)
	parsed2, _ := x509.ParseCertificate(block2.Bytes)

	if parsed1.Subject.CommonName == parsed2.Subject.CommonName {
		t.Error("different hosts should produce different CommonNames")
	}
}

func TestEnsureCerts_UniqueSerialNumbers(t *testing.T) {
	dirs := make([]string, 3)
	for i := range dirs {
		dirs[i] = t.TempDir()
		if err := EnsureCerts(dirs[i]); err != nil {
			t.Fatalf("iteration %d: EnsureCerts: %v", i, err)
		}
	}

	serials := make(map[string]bool)
	for _, dir := range dirs {
		certPEM, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
		block, _ := pem.Decode(certPEM)
		cert, _ := x509.ParseCertificate(block.Bytes)
		s := cert.SerialNumber.String()
		if serials[s] {
			t.Errorf("duplicate serial number: %s", s)
		}
		serials[s] = true
	}
}

func TestGenerateCA_ReturnsValidKey(t *testing.T) {
	caCertPEM, caKeyPEM, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	// Verify key is valid ECDSA P-256
	keyBlock, _ := pem.Decode(caKeyPEM)
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	if key.Curve != elliptic.P256() {
		t.Errorf("expected P-256 curve, got %v", key.Curve)
	}

	// Verify cert and key match
	certBlock, _ := pem.Decode(caCertPEM)
	cert, _ := x509.ParseCertificate(certBlock.Bytes)
	if !key.PublicKey.Equal(cert.PublicKey) {
		t.Error("CA key does not match CA cert public key")
	}
}

func TestEnsureCerts_ConsistentAcrossRestarts(t *testing.T) {
	dir := t.TempDir()

	EnsureCerts(dir)
	data1, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))

	EnsureCerts(dir)
	data2, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))

	if string(data1) != string(data2) {
		t.Error("cert differs across restarts")
	}
}

func BenchmarkEnsureCerts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		EnsureCerts(dir)
	}
}
