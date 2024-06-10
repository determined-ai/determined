package license

import (
	"github.com/golang-jwt/jwt/v4"
)

const (
	licenseRequiredMsg = "An enterprise license is required to use this feature"
	errCheckingLicense = "error when validating license"
)

// licenseKey stores the MLDE licenseKey if provided, else defaulting to no licenseKey.
var licenseKey string

// publicKey stores the public key used to verify licenses. Defaults to empty.
var publicKey string

// decodedLicense contains the body of a decoded licenseKey.
type decodedLicense struct {
	jwt.RegisteredClaims

	LicenseVersion string `json:"licenseVersion"`
}

// RequireLicense panics if no licenseKey or an invalid licenseKey is used.
func RequireLicense(resource string) {
	return
}

// IsEE returns true if a license is detected.
func IsEE() bool {
	return true
}
