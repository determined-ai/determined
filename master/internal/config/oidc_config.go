package config

import (
	"net/url"
)

// OIDCConfig holds the parameters for the OIDC provider.
type OIDCConfig struct {
	Enabled                     bool   `json:"enabled"`
	Provider                    string `json:"provider"`
	ClientID                    string `json:"client_id"`
	ClientSecret                string `json:"client_secret"`
	IDPSSOURL                   string `json:"idp_sso_url"`
	IDPRecipientURL             string `json:"idp_recipient_url"`
	AuthenticationClaim         string `json:"authentication_claim"`
	SCIMAuthenticationAttribute string `json:"scim_authentication_attribute"`
	AutoProvisionUsers          bool   `json:"auto_provision_users"`
	GroupsAttributeName         string `json:"groups_attribute_name"`
	DisplayNameAttributeName    string `json:"display_name_attribute_name"`
	AgentUIDAttributeName       int    `json:"agent_uid_attribute_name"`
	AgentGIDAttributeName       int    `json:"agent_gid_attribute_name"`
	AgentUserNameAttributeName  string `json:"agent_user_name_attribute_name"`
	AgentGroupNameAttributeName string `json:"agent_group_name_attribute_name"`
	AlwaysRedirect              bool   `json:"always_redirect"`
}

// Validate implements the check.Validatable interface.
func (c OIDCConfig) Validate() []error {
	if !c.Enabled {
		return nil
	}

	_, err := url.Parse(c.IDPRecipientURL)
	return []error{err}
}
