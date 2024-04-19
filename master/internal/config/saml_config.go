package config

import (
	"net/url"

	"github.com/determined-ai/determined/master/pkg/check"
)

// SAMLConfig describes config for SAML.
type SAMLConfig struct {
	Enabled                  bool   `json:"enabled"`
	Provider                 string `json:"provider"`
	IDPRecipientURL          string `json:"idp_recipient_url"`
	IDPSSOURL                string `json:"idp_sso_url"`
	IDPSSODescriptorURL      string `json:"idp_sso_descriptor_url"`
	IDPMetadataURL           string `json:"idp_metadata_url"`
	AutoProvisionUsers       bool   `json:"auto_provision_users"`
	GroupsAttributeName      string `json:"groups_attribute_name"`
	DisplayNameAttributeName string `json:"display_name_attribute_name"`
}

// Validate implements the check.Validatable interface.
func (c SAMLConfig) Validate() []error {
	if !c.Enabled {
		return nil
	}

	_, urlErr := url.Parse(c.IDPRecipientURL)

	return []error{
		urlErr,
		check.NotEmpty(c.Provider, "saml_provider must be specified"),
		check.NotEmpty(c.IDPRecipientURL, "saml_idp_recipient_url must be specified"),
		check.NotEmpty(c.IDPSSOURL, "saml_idp_sso_url must be specified"),
		check.NotEmpty(c.IDPSSODescriptorURL, "saml_idp_sso_descriptor_url must be specified"),
		check.NotEmpty(c.IDPMetadataURL, "saml_metadata_url must be specified"),
	}
}
