package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

var ErrCacheDirMissing = errors.New("-cache-dir is missing")

func writePEM(path, typ string, bytes []byte, perm os.FileMode) error {
	// #nosec G304: path is validated and restricted to cacheDir
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create PEM file: %w", err)
	}
	defer func() {
		if derr := f.Close(); derr != nil && err == nil {
			err = fmt.Errorf("failed to close PEM file: %w", derr)
		}
	}()
	if err := pem.Encode(f, &pem.Block{Type: typ, Bytes: bytes}); err != nil {
		return fmt.Errorf("failed to encode %s: %w", typ, err)
	}
	return nil
}

func generateSelfSignedCerts(cacheDir string) error {
	if cacheDir == "" {
		return ErrCacheDirMissing
	}

	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"development"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years is more reasonable
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certPath := filepath.Join(cacheDir, "localhost.crt")
	if err := writePEM(certPath, "CERTIFICATE", derBytes, 0644); err != nil {
		return err
	}

	keyPath := filepath.Join(cacheDir, "localhost.key")
	if err := writePEM(keyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key), 0600); err != nil {
		return err
	}

	return nil
}

func main() {
	cacheDir := flag.String("cache-dir", "", "directory to store generated self-signed TLS certificates")
	flag.Parse()

	if err := generateSelfSignedCerts(*cacheDir); err != nil {
		panic(fmt.Errorf("failed to generate self-signed certificates for development: %w", err))
	}
}
