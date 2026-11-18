//go:build integration

package integration

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Starts the built executable, not only its HTTP handler. The trusted certificate
// and TLS key are temporary local test data; verification is never disabled.
func testAccountProcess(t *testing.T) {
	t.Helper()
	certFile, keyFile, pool := localCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	binary := os.Getenv("CONNECT_API_TEST_BINARY")
	if binary == "" {
		t.Fatal("Use make integration in backend/ to build and select the test executable")
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary)
	cmd.Env = []string{"APP_ENV=test", "DATABASE_URL=" + os.Getenv("TEST_DATABASE_URL"), "API_LISTEN_ADDR=" + addr,
		"AUTH_RP_ID=connect.test", "AUTH_RP_ORIGINS=https://connect.test", "API_TLS_CERT_FILE=" + certFile, "API_TLS_KEY_FILE=" + keyFile}
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if cmd.Start() != nil {
		t.Fatal("start built API; run make api-integration to build its prerequisite")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	exited := false
	t.Cleanup(func() {
		if !exited {
			_ = cmd.Process.Kill()
			<-done
		}
	})
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		response, err := client.Get("https://" + addr + "/readyz")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				ready = true
				break
			}
		}
		select {
		case <-done:
			exited = true
			t.Fatal("API exited before readiness")
		default:
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		t.Fatal("TLS API did not become ready")
	}
	response, err := client.Post("https://"+addr+"/v1/accounts/registration/options", "application/json", strings.NewReader(`{"username":"process_test"}`))
	if err != nil {
		t.Fatal("verified TLS registration options failed")
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || response.StatusCode != 200 || !bytes.Contains(body, []byte(`"publicKey"`)) {
		t.Fatal("account routes were not enabled in built API")
	}
	if cmd.Process.Signal(syscall.SIGTERM) != nil {
		t.Fatal("signal API")
	}
	select {
	case err := <-done:
		exited = true
		if err != nil {
			t.Fatal("API shutdown failed")
		}
	case <-time.After(7 * time.Second):
		t.Fatal("API shutdown exceeded deadline")
	}
	text := logs.String()
	if !strings.Contains(text, "api stopped") || strings.Contains(text, os.Getenv("TEST_DATABASE_URL")) || strings.Contains(text, "process_test") || strings.Contains(text, "127.0.0.1:") && strings.Contains(text, "TLS handshake") {
		t.Fatal("unexpected or sensitive lifecycle log")
	}
	// The process's pending ceremony is test data, not an enrolled account.
}
func localCertificate(t *testing.T) (string, string, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Connect local integration test"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	certFile, keyFile := filepath.Join(directory, "local-cert.pem"), filepath.Join(directory, "local-key.pem")
	if os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600) != nil || os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600) != nil {
		t.Fatal("write temporary TLS test material")
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	return certFile, keyFile, pool
}
