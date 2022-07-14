package sso

import (
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/aproto"
)

// AddSSOProviderInfo uses the master config to add SSOProvider
// to the provided masterInfo. In OSS this just returns masterInfo.
func AddSSOProviderInfo(config *config.Config, masterInfo aproto.MasterInfo) aproto.MasterInfo {
	return masterInfo
}
