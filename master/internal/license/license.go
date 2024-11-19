package license

const (
	licenseRequiredMsg = "An enterprise license is required to use this feature"
	errCheckingLicense = "error when validating license"
)

// licenseKey stores the MLDE licenseKey if provided, else defaulting to no licenseKey.
var licenseKey string

// publicKey stores the public key used to verify licenses. Defaults to empty.
var publicKey string

// RequireLicense is a no-op.
func RequireLicense(resource string) {}

// IsEE returns true if a license is detected.
func IsEE() bool {
	if publicKey != "" && licenseKey != "" {
		return true
	}
	return false
}

// SetLicenseAndKey populates the license key and public key used for EE
// license checks. This is primarily useful for testing.
func SetLicenseAndKey(newLicenseKey, newPublicKey string) {
	if licenseKey != "" || publicKey != "" {
		return
	}

	licenseKey = newLicenseKey
	publicKey = newPublicKey
}
