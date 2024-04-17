package license

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

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
	if publicKey == "" || licenseKey == "" {
		// TODO: get better messaging for this
		panic(fmt.Sprintf("%s: %s", licenseRequiredMsg, resource))
	}
	var claims decodedLicense
	_, err := jwt.ParseWithClaims(licenseKey, &claims, func(token *jwt.Token) (interface{}, error) {
		pemData, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			return nil, err
		}
		blk, _ := pem.Decode(pemData)
		if blk == nil {
			return nil, fmt.Errorf("error decoding pem")
		}
		key, err := x509.ParsePKIXPublicKey(blk.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing public key: %w", err)
		}
		return key, nil
	})
	if err != nil {
		panic(fmt.Sprintf("%s: %s", errCheckingLicense, err.Error()))
	}
	if claims.LicenseVersion != "1" {
		panic("Specified licenseKey version is incompatible")
	}
}
