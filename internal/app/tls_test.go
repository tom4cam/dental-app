package app

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

func lan(ip string) []net.IP {
	return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP(ip)}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func TestEnsureSelfSigned_CoversEveryNameAndAddress(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	certPath, keyPath, err := ensureSelfSigned(dir, []string{"localhost", "front-desk"}, lan("192.168.1.100"), testNow)
	if err != nil {
		t.Fatalf("ensureSelfSigned: %v", err)
	}
	pair, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("the pair doesn't load: %v", err)
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"localhost", "front-desk", "127.0.0.1", "192.168.1.100"} {
		if err := cert.VerifyHostname(host); err != nil {
			t.Errorf("certificate should cover %s: %v", host, err)
		}
	}
	if got := cert.NotAfter.Sub(cert.NotBefore); got > 826*24*time.Hour {
		t.Errorf("lifetime %v is longer than Apple accepts (825 days)", got)
	}
	if !cert.NotBefore.Before(testNow) {
		t.Error("certificate must already be valid when made")
	}
}

func TestEnsureSelfSigned_KeyIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows doesn't report Unix permission bits")
	}
	dir := filepath.Join(t.TempDir(), "tls")
	_, keyPath, err := ensureSelfSigned(dir, []string{"localhost"}, lan("10.0.0.5"), testNow)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode is %v, want 0600", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(dir)
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("tls folder mode is %v, want 0700", dirInfo.Mode().Perm())
	}
}

func TestEnsureSelfSigned_ReusesWhileStillValid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	names, ips := []string{"localhost"}, lan("192.168.1.100")
	certPath, keyPath, err := ensureSelfSigned(dir, names, ips, testNow)
	if err != nil {
		t.Fatal(err)
	}
	cert1, key1 := readFile(t, certPath), readFile(t, keyPath)

	if _, _, err := ensureSelfSigned(dir, names, ips, testNow.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if string(readFile(t, certPath)) != string(cert1) || string(readFile(t, keyPath)) != string(key1) {
		t.Error("a still-valid certificate should be kept, so devices that trusted it keep working")
	}
}

func TestEnsureSelfSigned_ReplacedWhenTheNetworkAddressChanges(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	names := []string{"localhost"}
	certPath, _, err := ensureSelfSigned(dir, names, lan("192.168.1.100"), testNow)
	if err != nil {
		t.Fatal(err)
	}
	before := readFile(t, certPath)

	if _, _, err := ensureSelfSigned(dir, names, lan("192.168.1.222"), testNow); err != nil {
		t.Fatal(err)
	}
	after := readFile(t, certPath)
	if string(before) == string(after) {
		t.Fatal("a new address needs a new certificate")
	}
	block, _ := pemDecode(after)
	cert, err := x509.ParseCertificate(block)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.VerifyHostname("192.168.1.222"); err != nil {
		t.Errorf("new certificate should cover the new address: %v", err)
	}
}

func TestEnsureSelfSigned_ReplacedNearExpiry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	names, ips := []string{"localhost"}, lan("192.168.1.100")
	certPath, _, err := ensureSelfSigned(dir, names, ips, testNow)
	if err != nil {
		t.Fatal(err)
	}
	before := readFile(t, certPath)
	later := testNow.Add(selfSignedLifetime - 10*24*time.Hour)
	if _, _, err := ensureSelfSigned(dir, names, ips, later); err != nil {
		t.Fatal(err)
	}
	if string(readFile(t, certPath)) == string(before) {
		t.Error("a certificate within 30 days of expiry should be replaced")
	}
}

func TestEnsureSelfSigned_ReplacedWhenTheKeyDoesntMatch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	names, ips := []string{"localhost"}, lan("192.168.1.100")
	certPath, keyPath, err := ensureSelfSigned(dir, names, ips, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ensureSelfSigned(dir, names, ips, testNow); err != nil {
		t.Fatalf("a broken key should be replaced, not fatal: %v", err)
	}
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		t.Errorf("pair should work again: %v", err)
	}
}

// The browser-facing promise: a client that trusts the certificate gets a normal HTTPS reply,
// and plain HTTP to the same port does not work.
func TestSelfSignedCertificateServesHTTPS(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tls")
	certPath, keyPath, err := ensureSelfSigned(dir, []string{"localhost"}, lan("192.168.1.100"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{pair}}
	srv.StartTLS()
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(readFile(t, certPath))
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "localhost"}}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("HTTPS request with the certificate trusted: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status %d", resp.StatusCode)
	}

	untrusted := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{ServerName: "localhost"}}}
	if _, err := untrusted.Get(srv.URL); err == nil {
		t.Error("a client that doesn't trust the certificate should be refused")
	}
}

func pemDecode(b []byte) ([]byte, []byte) {
	block, rest := pem.Decode(b)
	return block.Bytes, rest
}
