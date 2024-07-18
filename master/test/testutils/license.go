package testutils

import (
	"fmt"
	"io"
	"os"

	"github.com/determined-ai/determined/master/internal/license"
)

// LoadLicenseAndKeyFromFilesystem attempts to find a license key and public key in
// the local filesystem. If found, it sets them. Returns an error if the files are
// not found or there is an error reading from them.
func LoadLicenseAndKeyFromFilesystem() error {
	// TODO: read these from environment but default to this when env vars unset
	const (
		licenseKeyPath = "license.txt" // TODO: maybe "../license.txt"?
		publicKeyPath  = "public.txt"  // TODO: maybe "../public.txt"?
	)

	// Check for the existence of license key file
	licenseKeyFile, licenseKeyErr := os.Open(licenseKeyPath)
	if licenseKeyErr != nil {
		return fmt.Errorf("error opening %s: %w", licenseKeyPath, licenseKeyErr)
	} else {
		_, statErr := licenseKeyFile.Stat()
		if statErr != nil {
			return fmt.Errorf("error opening %s: %w", licenseKeyPath, statErr)
		}
	}

	// Check for the existence of public key file
	publicKeyFile, publicKeyErr := os.Open(publicKeyPath)
	if publicKeyErr != nil {
		return fmt.Errorf("error opening %s: %w", publicKeyPath, publicKeyErr)
	} else {
		_, statErr := publicKeyFile.Stat()
		if statErr != nil {
			return fmt.Errorf("error opening %s: %w", publicKeyPath, statErr)
		}
	}

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
}

// MustLoadLicenseAndKeyFromFilesystem attempts to find a license key and public key in
// the local filesystem. If found, it sets them. Panics if the files are
// not found or there is an error reading from them.
func MustLoadLicenseAndKeyFromFilesystem() {
	err := LoadLicenseAndKeyFromFilesystem()
	if err != nil {
		panic(err)
	}
}
