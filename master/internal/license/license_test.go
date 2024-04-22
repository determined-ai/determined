package license

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testResource = "test resource"
)

func TestRequireLicenseWithEmptyLicenseKey(t *testing.T) {
	licenseKey = ""

	defer func() {
		if r := recover(); r != nil {
			require.Contains(t, r, licenseRequiredMsg)
			require.Contains(t, r, testResource)
			return
		}
		require.Fail(t, "panic expected")
	}()
	RequireLicense(testResource)
}

func TestRequireLicenseWithEmptyPublicKey(t *testing.T) {
	publicKey = ""

	defer func() {
		if r := recover(); r != nil {
			require.Contains(t, r, licenseRequiredMsg)
			require.Contains(t, r, testResource)
			return
		}
		require.Fail(t, "panic expected")
	}()
	RequireLicense(testResource)
}

func TestRequireLicenseWithInvalidPublicKey(t *testing.T) {
	publicKey = "badKey"
	licenseKey = "fakeLicense"
	defer func() {
		if r := recover(); r != nil {
			require.Contains(t, r, errCheckingLicense)
		}
	}()
	RequireLicense(testResource)
}

func TestRequireLicenseWithInvalidLicenseKey(t *testing.T) {
	publicKey = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0" +
		"RRZ0FFTkRVTlV3cG82RkpoOWl5Ni8ySC9wTUFyRGxXYQppUUxjNko3QVVYWmFuUzdxbC8xcjZjRDBrbjN2bHV2" +
		"MjBxSk1abEVPb1NscEU0WWpKbjQxNzZUWEJnPT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="
	licenseKey = "badLicense"
	defer func() {
		if r := recover(); r != nil {
			require.Contains(t, r, errCheckingLicense)
		}
	}()
	RequireLicense(testResource)
}
