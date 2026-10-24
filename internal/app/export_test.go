package app

import "time"

// MakeTestCertificate writes a self-signed certificate and key into dir for tests in package
// app_test, which can't reach the unexported helper.
func MakeTestCertificate(dir string) (certPath, keyPath string, err error) {
	return ensureSelfSigned(dir, []string{"localhost"}, nil, time.Now())
}
