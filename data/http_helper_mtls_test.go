package data

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMTLSRoundTrip exercises the real client path (HTTPHelper.Call via the
// new client-cert / root-CA options) against a server whose tls.Config mirrors
// what LandauServer.buildServerTLSConfig produces for mTLS. It proves:
//   - a client presenting a CA-issued cert connects and the server sees the
//     verified CN (transport-layer identity),
//   - a client with NO cert is rejected at the TLS handshake.
func TestMTLSRoundTrip(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey := genCA(t)
	// server cert for name "redis-server" + 127.0.0.1
	srvCertPEM, srvKeyPEM := genLeaf(t, caCert, caKey, "redis-server", []string{"redis-server"}, true)
	// client (agent) cert whose CN is the authoritative instance-id
	const instanceID = "inst-dev-0001"
	cliCertFile := filepath.Join(dir, "agent.crt")
	cliKeyFile := filepath.Join(dir, "agent.key")
	cliCertPEM, cliKeyPEM := genLeaf(t, caCert, caKey, instanceID, nil, false)
	writeFile(t, cliCertFile, cliCertPEM)
	writeFile(t, cliKeyFile, cliKeyPEM)
	caFile := filepath.Join(dir, "ca.crt")
	writeFile(t, caFile, caCert.pemBytes)

	// Server side: same tls.Config shape as buildServerTLSConfig's mTLS branch.
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert.pemBytes)
	srvPair, err := tls.X509KeyPair(srvCertPEM, srvKeyPEM)
	if err != nil {
		t.Fatalf("server keypair: %v", err)
	}
	gotCN := make(chan string, 1)
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cn := ""
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			cn = r.TLS.PeerCertificates[0].Subject.CommonName
		}
		select {
		case gotCN <- cn:
		default:
		}
		w.Write([]byte(`{"Code":0,"Message":"ok"}`))
	}))
	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{srvPair},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	ts.StartTLS()
	defer ts.Close()
	url := ts.URL // https://127.0.0.1:PORT

	t.Run("with client cert -> server sees verified CN", func(t *testing.T) {
		h, err := NewHTTPHelper(
			SetHTTPUrl(url),
			SetHTTPClientCertificate(cliCertFile, cliKeyFile),
			SetHTTPRootCA(caFile),
			SetHTTPTLSServerName("redis-server"),
		)
		if err != nil {
			t.Fatalf("new helper: %v", err)
		}
		if _, err := h.Call(); err != nil {
			t.Fatalf("mTLS call failed: %v", err)
		}
		if cn := <-gotCN; cn != instanceID {
			t.Fatalf("server saw CN %q, want %q", cn, instanceID)
		}
	})

	t.Run("without client cert -> handshake rejected", func(t *testing.T) {
		h, err := NewHTTPHelper(
			SetHTTPUrl(url),
			SetHTTPRootCA(caFile),
			SetHTTPTLSServerName("redis-server"),
		)
		if err != nil {
			t.Fatalf("new helper: %v", err)
		}
		if _, err := h.Call(); err == nil {
			t.Fatalf("expected handshake failure without client cert, got success")
		}
	})
	_ = cliCertPEM
	_ = cliKeyPEM
}

// --- tiny self-contained cert helpers (no openssl / no fixtures) ---

type caMaterial struct {
	cert     *x509.Certificate
	pemBytes []byte
}

func genCA(t *testing.T) (*caMaterial, *ecdsa.PrivateKey) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA: %v", err)
	}
	cert, _ := x509.ParseCertificate(der)
	return &caMaterial{cert: cert, pemBytes: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}, key
}

func genLeaf(t *testing.T, ca *caMaterial, caKey *ecdsa.PrivateKey, cn string, dnsNames []string, server bool) (certPEM, keyPEM []byte) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     dnsNames,
	}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.IPAddresses = append(tmpl.IPAddresses, net.ParseIP("127.0.0.1"))
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf %s: %v", cn, err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
