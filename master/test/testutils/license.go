package testutils

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/determined-ai/determined/master/internal/license"
)

// LoadLicenseAndKeyFromFilesystem attempts to find a license key and public key in
// the local filesystem. If found, it sets them. Returns an error if the files are
// not found or there is an error reading from them.
func LoadLicenseAndKeyFromFilesystem(keyPath string) (err error) {
	if license.IsEE() {
		return nil
	}

	const (
		licenseKeyFilename = "license.txt"
		publicKeyFilename  = "public.txt"
	)
	var (
		licenseKeyPath = path.Join(keyPath, licenseKeyFilename)
		publicKeyPath  = path.Join(keyPath, publicKeyFilename)
	)

	// Check for the existence of license key file
	licenseKeyFile, licenseKeyErr := os.Open(licenseKeyPath) // #nosec G304
	if licenseKeyErr != nil {
		return fmt.Errorf("error opening %s: %w", licenseKeyPath, licenseKeyErr)
	}
	defer closeOrOverwriteError(licenseKeyFile, &err)

	// Check for the existence of public key file
	publicKeyFile, publicKeyErr := os.Open(publicKeyPath) // #nosec G304
	if publicKeyErr != nil {
		return fmt.Errorf("error opening %s: %w", publicKeyPath, publicKeyErr)
	}
	defer closeOrOverwriteError(publicKeyFile, &err)

	// Since they exist, read their contents
	licenseKeyContent, licenseKeyErr := io.ReadAll(licenseKeyFile)
	if licenseKeyErr != nil {
		return fmt.Errorf("error reading %s: %w", licenseKeyPath, licenseKeyErr)
	}
	publicKeyContent, publicKeyErr := io.ReadAll(publicKeyFile)
	if publicKeyErr != nil {
		return fmt.Errorf("error opening %s: %w", publicKeyPath, publicKeyErr)
	}

	// No errors have been encountered, so set the keys.
	license.SetLicenseAndKey(string(licenseKeyContent), string(publicKeyContent))
	return nil
}

// MustLoadLicenseAndKeyFromFilesystem attempts to find a license key and public key in
// the local filesystem. If found, it sets them. Panics if the files are
// not found or there is an error reading from them.
func MustLoadLicenseAndKeyFromFilesystem(path string) {
	err := LoadLicenseAndKeyFromFilesystem(path)
	if err != nil {
		panic(err)
	}
}

func closeOrOverwriteError(c io.Closer, err *error) {
	closeErr := c.Close()
	if err != nil && *err == nil && closeErr != nil {
		*err = closeErr
	}
}
