package entry

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildServerTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, caFile := filepath.Join(dir, "s.crt"), filepath.Join(dir, "s.key"), filepath.Join(dir, "ca.crt")
	certPEM, keyPEM := selfSigned(t)
	mustWrite(t, certFile, certPEM)
	mustWrite(t, keyFile, keyPEM)
	mustWrite(t, caFile, certPEM) // reuse the self-signed cert as a CA bundle for the pool

	t.Run("no TLS fields -> nil (plain HTTP, backward compatible)", func(t *testing.T) {
		cfg, err := (&LandauServer{}).buildServerTLSConfig()
		if err != nil || cfg != nil {
			t.Fatalf("want (nil,nil), got (%v,%v)", cfg, err)
		}
	})

	t.Run("cert+key, no client CA -> TLS without client auth", func(t *testing.T) {
		cfg, err := (&LandauServer{TLSCertFile: certFile, TLSKeyFile: keyFile}).buildServerTLSConfig()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("want 1 server cert")
		}
		if cfg.ClientAuth != tls.NoClientCert {
			t.Fatalf("want NoClientCert, got %v", cfg.ClientAuth)
		}
	})

	t.Run("cert+key+client CA -> mTLS (RequireAndVerifyClientCert)", func(t *testing.T) {
		cfg, err := (&LandauServer{TLSCertFile: certFile, TLSKeyFile: keyFile, TLSClientCAFile: caFile}).buildServerTLSConfig()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Fatalf("want RequireAndVerifyClientCert, got %v", cfg.ClientAuth)
		}
		if cfg.ClientCAs == nil {
			t.Fatalf("want ClientCAs pool set")
		}
	})

	t.Run("explicit TLSConfig wins", func(t *testing.T) {
		custom := &tls.Config{MinVersion: tls.VersionTLS13}
		cfg, err := (&LandauServer{TLSConfig: custom, TLSCertFile: certFile, TLSKeyFile: keyFile}).buildServerTLSConfig()
		if err != nil || cfg != custom {
			t.Fatalf("want the custom config returned verbatim")
		}
	})

	t.Run("bad cert file -> error (fail closed, not silent plaintext)", func(t *testing.T) {
		_, err := (&LandauServer{TLSCertFile: filepath.Join(dir, "nope.crt"), TLSKeyFile: keyFile}).buildServerTLSConfig()
		if err == nil {
			t.Fatalf("want error for missing cert file")
		}
	})

	t.Run("cert without key -> error (asymmetric misconfig, not silent plaintext)", func(t *testing.T) {
		if _, err := (&LandauServer{TLSCertFile: certFile}).buildServerTLSConfig(); err == nil {
			t.Fatalf("want error when only TLSCertFile is set")
		}
	})

	t.Run("key without cert -> error", func(t *testing.T) {
		if _, err := (&LandauServer{TLSKeyFile: keyFile}).buildServerTLSConfig(); err == nil {
			t.Fatalf("want error when only TLSKeyFile is set")
		}
	})

	t.Run("client CA without server cert/key -> error", func(t *testing.T) {
		if _, err := (&LandauServer{TLSClientCAFile: caFile}).buildServerTLSConfig(); err == nil {
			t.Fatalf("want error when only TLSClientCAFile is set (no server cert to do TLS)")
		}
	})
}

func selfSigned(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func mustWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
