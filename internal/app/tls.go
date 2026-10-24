package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Kept as variables so tests can control the clock.
var timeNow = time.Now

const (
	// Apple refuses server certificates valid for longer than 825 days.
	selfSignedLifetime = 825 * 24 * time.Hour
	// A certificate this close to expiry is replaced the next time the server starts.
	renewBefore = 30 * 24 * time.Hour
)

// localNames lists the names and addresses a browser on this network could use to reach the
// server, so a self-signed certificate covers them.
func localNames() ([]string, []net.IP) {
	names := []string{"localhost"}
	if host, err := os.Hostname(); err == nil && host != "" && host != "localhost" {
		names = append(names, host)
	}
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return names, ips
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalUnicast() || ipnet.IP.IsLinkLocalMulticast() {
			continue
		}
		ips = append(ips, ipnet.IP)
	}
	return names, ips
}

// ensureSelfSigned returns the paths of a self-signed certificate and its key in dir. It
// reuses what is already there while it is still valid for every name and address, and
// makes a new pair otherwise (a new address on the network, or a certificate near expiry).
func ensureSelfSigned(dir string, names []string, ips []net.IP, now time.Time) (certPath, keyPath string, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	if reusable(certPath, keyPath, names, ips, now) {
		return certPath, keyPath, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return "", "", err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "LibreDental local server", Organization: []string{"LibreDental"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(selfSignedLifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     names,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", err
	}
	if err := writeFileAtomic(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return "", "", err
	}
	if err := writeFileAtomic(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

func reusable(certPath, keyPath string, names []string, ips []net.IP, now time.Time) bool {
	pair, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil || len(pair.Certificate) == 0 {
		return false
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return false
	}
	if now.Before(cert.NotBefore) || now.Add(renewBefore).After(cert.NotAfter) {
		return false
	}
	for _, n := range names {
		if cert.VerifyHostname(n) != nil {
			return false
		}
	}
	for _, ip := range ips {
		if cert.VerifyHostname(ip.String()) != nil {
			return false
		}
	}
	return true
}

// writeFileAtomic writes to a temporary file in the same folder and renames it into place, so
// a crash never leaves half a key behind. mode is set explicitly, whatever the umask says.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return os.Rename(tmp.Name(), path)
}
