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

func TestBootstrapCertificatesAndDoctor(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")

	if err := bootstrapCertificates(certBootstrapOptions{
		CommonName:       "edge.local",
		ClientCommonName: "Device-001",
		CACommonName:     "GONK Test Offline CA",
		Output:           certDir,
		Days:             365,
		CADays:           3650,
	}); err != nil {
		t.Fatalf("bootstrapCertificates failed: %v", err)
	}

	configPath := filepath.Join(dir, "gonk.yaml")
	writeFile(t, configPath, `server:
  listen: ":8443"
  tls:
    enabled: true
    cert_file: "`+filepath.ToSlash(filepath.Join(certDir, "server.crt"))+`"
    key_file: "`+filepath.ToSlash(filepath.Join(certDir, "server.key"))+`"
    client_ca: "`+filepath.ToSlash(filepath.Join(certDir, "ca.crt"))+`"
    client_auth: "require"
logging:
  level: info
  format: text
  output: stdout
routes:
  - name: device-api
    path: /device/*
    methods: [GET]
    upstreams:
      - url: http://backend:3000
    auth:
      type: mtls
      required: true
`)

	if err := runCertsDoctor(certDoctorOptions{
		ConfigPath:     configPath,
		ClientCertFile: filepath.Join(certDir, "client.crt"),
		ServerName:     "edge.local",
		WarnDays:       30,
	}); err != nil {
		t.Fatalf("runCertsDoctor failed: %v", err)
	}
}

func TestCertsDoctorFailsWhenClientCAMissing(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")

	if err := bootstrapCertificates(certBootstrapOptions{
		CommonName:       "edge.local",
		ClientCommonName: "Device-001",
		Output:           certDir,
	}); err != nil {
		t.Fatalf("bootstrapCertificates failed: %v", err)
	}

	configPath := filepath.Join(dir, "gonk.yaml")
	writeFile(t, configPath, `server:
  listen: ":8443"
  tls:
    enabled: true
    cert_file: "`+filepath.ToSlash(filepath.Join(certDir, "server.crt"))+`"
    key_file: "`+filepath.ToSlash(filepath.Join(certDir, "server.key"))+`"
    client_auth: "require"
logging:
  level: info
  format: text
  output: stdout
routes:
  - name: device-api
    path: /device/*
    methods: [GET]
    upstreams:
      - url: http://backend:3000
    auth:
      type: mtls
      required: true
`)

	if err := runCertsDoctor(certDoctorOptions{
		ConfigPath: configPath,
		ServerName: "edge.local",
		WarnDays:   30,
	}); err == nil {
		t.Fatal("runCertsDoctor should fail when client_ca is missing")
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

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
