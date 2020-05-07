package saml

import (
	"os"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

// Config describes config for SAML.
type Config struct {
	Enabled             bool   `json:"enabled"`
	Provider            string `json:"provider"`
	IDPRecipientURL     string `json:"idp_recipient_url"`
	IDPSSOURL           string `json:"idp_sso_url"`
	IDPSSODescriptorURL string `json:"idp_sso_descriptor_url"`
	IDPCertPath         string `json:"idp_cert_path"`
}

// Validate implements the check.Validatable interface.
func (c Config) Validate() []error {
	if !c.Enabled {
		return nil
	}

	certErr := check.NotEmpty(c.IDPCertPath, "saml_idp_cert_path must be specified")
	if certErr == nil {
		_, certErr = os.Stat(c.IDPCertPath)
		if os.IsNotExist(certErr) {
			certErr = errors.Wrap(certErr, "saml_idp_cert_path supplied but file does not exist")
		} else if certErr != nil {
			certErr = errors.Wrap(certErr, "found file at saml_idp_cert_path but could not open it")
		}
	}

	return []error{
		check.NotEmpty(c.Provider, "saml_provider must be specified"),
		check.NotEmpty(c.IDPRecipientURL, "saml_idp_recipient_url must be specified"),
		check.NotEmpty(c.IDPSSOURL, "saml_idp_sso_url must be specified"),
		check.NotEmpty(c.IDPSSODescriptorURL, "saml_idp_sso_descriptor_url must be specified"),
		certErr,
	}
}
