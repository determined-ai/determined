package sso

import (
	"net/url"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/oidc"
	"github.com/determined-ai/determined/master/internal/saml"
	"github.com/determined-ai/determined/master/pkg/aproto"
)

// AddSSOProviderInfo uses the master config to add SSOProvider
// to the provided masterInfo. In OSS this just returns masterInfo.
func AddSSOProviderInfo(config *config.Config, masterInfo aproto.MasterInfo) aproto.MasterInfo {
	return masterInfo
}
