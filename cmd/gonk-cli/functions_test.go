package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCertificateSignsServerAndClientWithCA(t *testing.T) {
	dir := t.TempDir()

	generateCertificate("GONK Test CA", "ca", dir, "", "")
	generateCertificate("localhost", "server", dir, filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key"))
	generateCertificate("Device-001", "client", dir, filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key"))

	roots := x509.NewCertPool()
	roots.AddCert(readCertificate(t, filepath.Join(dir, "ca.crt")))

	serverCert := readCertificate(t, filepath.Join(dir, "server.crt"))
	if _, err := serverCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		DNSName:   "localhost",
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Fatalf("server certificate did not verify against CA: %v", err)
	}

	clientCert := readCertificate(t, filepath.Join(dir, "client.crt"))
	if _, err := clientCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		t.Fatalf("client certificate did not verify against CA: %v", err)
	}
}

func readCertificate(t *testing.T, path string) *x509.Certificate {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read certificate %s: %v", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("failed to decode PEM certificate %s", path)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate %s: %v", path, err)
	}
	return cert
}
