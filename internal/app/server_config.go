package app

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ServerConfig holds environment and network configuration for server mode.
type ServerConfig struct {
	Host string
	Port int
	// TLSCertFile and TLSKeyFile name a certificate to serve HTTPS with.
	TLSCertFile string
	TLSKeyFile  string
	// TLSAuto asks for a self-signed certificate, made and kept in the app data folder.
	TLSAuto bool
}

// TLSFiles are the certificate and key server mode serves HTTPS with.
type TLSFiles struct {
	CertFile   string
	KeyFile    string
	SelfSigned bool
}

// LoadServerConfig parses server configuration from environment variables (LIBREDENTAL_HOST,
// LIBREDENTAL_PORT, LIBREDENTAL_TLS, LIBREDENTAL_TLS_CERT, LIBREDENTAL_TLS_KEY).
func LoadServerConfig() ServerConfig {
	serveHost := os.Getenv("LIBREDENTAL_HOST")
	if serveHost == "" {
		serveHost = "0.0.0.0"
	}
	servePort := 4242
	if portStr := os.Getenv("LIBREDENTAL_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			servePort = p
		} else {
			log.Printf("Warning: invalid LIBREDENTAL_PORT %q, using default %d", portStr, servePort)
		}
	}
	return ServerConfig{
		Host:        serveHost,
		Port:        servePort,
		TLSCertFile: strings.TrimSpace(os.Getenv("LIBREDENTAL_TLS_CERT")),
		TLSKeyFile:  strings.TrimSpace(os.Getenv("LIBREDENTAL_TLS_KEY")),
		TLSAuto:     isTruthy(os.Getenv("LIBREDENTAL_TLS")),
	}
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "auto", "1", "true", "yes", "on":
		return true
	}
	return false
}

// ExposedToNetwork reports whether the server listens beyond this computer.
func (c ServerConfig) ExposedToNetwork() bool {
	if c.Host == "localhost" {
		return false
	}
	ip := net.ParseIP(c.Host)
	return ip == nil || !ip.IsLoopback()
}

// TLS returns the certificate to serve HTTPS with, or nil to serve plain HTTP. A certificate
// that was asked for but can't be used is an error, never a quiet fall back to HTTP.
// appDir is where a self-signed certificate is kept.
func (c ServerConfig) TLS(appDir string) (*TLSFiles, error) {
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return nil, errors.New("LIBREDENTAL_TLS_CERT and LIBREDENTAL_TLS_KEY must be set together")
	}
	if c.TLSCertFile != "" {
		if _, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile); err != nil {
			return nil, fmt.Errorf("can't use the certificate in LIBREDENTAL_TLS_CERT and LIBREDENTAL_TLS_KEY: %w", err)
		}
		return &TLSFiles{CertFile: c.TLSCertFile, KeyFile: c.TLSKeyFile}, nil
	}
	if !c.TLSAuto {
		return nil, nil
	}
	names, ips := localNames()
	cert, key, err := ensureSelfSigned(filepath.Join(appDir, "tls"), names, ips, timeNow())
	if err != nil {
		return nil, fmt.Errorf("can't make a self-signed certificate: %w", err)
	}
	return &TLSFiles{CertFile: cert, KeyFile: key, SelfSigned: true}, nil
}
