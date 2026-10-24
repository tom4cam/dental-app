package app_test

import (
	"crypto/tls"
	"testing"

	"github.com/LibreDental/libredental/internal/app"
)

func TestLoadServerConfig_Defaults(t *testing.T) {
	t.Setenv("LIBREDENTAL_HOST", "")
	t.Setenv("LIBREDENTAL_PORT", "")

	cfg := app.LoadServerConfig()
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %q", cfg.Host)
	}
	if cfg.Port != 4242 {
		t.Errorf("Expected default port 4242, got %d", cfg.Port)
	}
}

func TestLoadServerConfig_Custom(t *testing.T) {
	t.Setenv("LIBREDENTAL_HOST", "127.0.0.1")
	t.Setenv("LIBREDENTAL_PORT", "8080")

	cfg := app.LoadServerConfig()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %q", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}
}

func TestLoadServerConfig_InvalidPortFallback(t *testing.T) {
	t.Setenv("LIBREDENTAL_PORT", "invalid")

	cfg := app.LoadServerConfig()
	if cfg.Port != 4242 {
		t.Errorf("Expected fallback port 4242 for invalid input, got %d", cfg.Port)
	}
}

func clearTLSEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"LIBREDENTAL_TLS", "LIBREDENTAL_TLS_CERT", "LIBREDENTAL_TLS_KEY"} {
		t.Setenv(k, "")
	}
}

func TestServerConfig_TLS_OffByDefault(t *testing.T) {
	clearTLSEnv(t)
	files, err := app.LoadServerConfig().TLS(t.TempDir())
	if err != nil || files != nil {
		t.Fatalf("plain HTTP by default, got %+v, %v", files, err)
	}
}

func TestServerConfig_TLS_AutoMakesASelfSignedCertificate(t *testing.T) {
	clearTLSEnv(t)
	for _, v := range []string{"auto", "AUTO", "1", "true", "on"} {
		t.Setenv("LIBREDENTAL_TLS", v)
		dir := t.TempDir()
		files, err := app.LoadServerConfig().TLS(dir)
		if err != nil || files == nil || !files.SelfSigned {
			t.Fatalf("LIBREDENTAL_TLS=%s: got %+v, %v", v, files, err)
		}
		if _, err := tls.LoadX509KeyPair(files.CertFile, files.KeyFile); err != nil {
			t.Errorf("LIBREDENTAL_TLS=%s: the pair doesn't load: %v", v, err)
		}
	}
	t.Setenv("LIBREDENTAL_TLS", "off")
	if files, err := app.LoadServerConfig().TLS(t.TempDir()); err != nil || files != nil {
		t.Errorf("LIBREDENTAL_TLS=off should mean plain HTTP, got %+v, %v", files, err)
	}
}

func TestServerConfig_TLS_UsesYourOwnCertificate(t *testing.T) {
	clearTLSEnv(t)
	dir := t.TempDir()
	cert, key, err := app.MakeTestCertificate(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIBREDENTAL_TLS_CERT", cert)
	t.Setenv("LIBREDENTAL_TLS_KEY", key)
	t.Setenv("LIBREDENTAL_TLS", "auto") // an explicit certificate wins over auto
	files, err := app.LoadServerConfig().TLS(t.TempDir())
	if err != nil || files == nil || files.SelfSigned || files.CertFile != cert || files.KeyFile != key {
		t.Fatalf("got %+v, %v", files, err)
	}
}

func TestServerConfig_TLS_RefusesToStartOnABadSetup(t *testing.T) {
	clearTLSEnv(t)
	t.Setenv("LIBREDENTAL_TLS_CERT", "/nonexistent/cert.pem")
	if _, err := app.LoadServerConfig().TLS(t.TempDir()); err == nil {
		t.Error("a certificate without a key must be an error")
	}
	t.Setenv("LIBREDENTAL_TLS_KEY", "/nonexistent/key.pem")
	if _, err := app.LoadServerConfig().TLS(t.TempDir()); err == nil {
		t.Error("an unreadable certificate must be an error, not a quiet fall back to HTTP")
	}
}

func TestServerConfig_ExposedToNetwork(t *testing.T) {
	cases := map[string]bool{
		"0.0.0.0": true, "192.168.1.100": true, "": true, "front-desk": true,
		"127.0.0.1": false, "::1": false, "localhost": false,
	}
	for host, want := range cases {
		cfg := app.ServerConfig{Host: host}
		if got := cfg.ExposedToNetwork(); got != want {
			t.Errorf("host %q: ExposedToNetwork() = %v, want %v", host, got, want)
		}
	}
}
