//nolint:exhaustruct

package options

import (
	"testing"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/logger"
)

func TestUnmarshalOptions(t *testing.T) {
	type OptionsUnmarshledTestCase struct {
		name     string
		raw      string
		expected Options
	}

	optionsTests := []OptionsUnmarshledTestCase{
		{
			name: "agent_config_no_log",
			raw: `
master_host: master_host_IP
master_port: 5000
container_master_host: docker_localhost
`,
			expected: Options{
				MasterHost:          "master_host_IP",
				MasterPort:          5000,
				ContainerMasterHost: "docker_localhost",
			},
		},
		{
			name: "agent_config_with_log",
			raw: `
master_host: master_host_IP
master_port: 5000
container_master_host: docker_localhost
log:
    level: debug
    color: false
`,
			expected: Options{
				MasterHost:          "master_host_IP",
				MasterPort:          5000,
				ContainerMasterHost: "docker_localhost",
				Log: logger.Config{
					Level: "debug",
					Color: false,
				},
			},
		},
		{
			name: "default_options_config",
			raw: `
log:
    level: trace
    color: true
slot_type: auto
security:
    tls:
        enabled: false
        skip_verify: false
debug: false
tls: false
api_enabled: false
bind_ip: 0.0.0.0
bind_port: 9090
agent_reconnect_attempts: 5
agent_reconnect_backoff: 5
container_runtime: docker
`,
			expected: *DefaultOptions(),
		},
		{
			name: "full_options_config",
			raw: `
config_file: agent_config
log:
    level: debug
    color: false
master_host: master_host_IP
master_port: 5000
agent_id: agent_device_name
resource_pool: agent_rp
container_master_host: docker_localhost
container_master_port: 2000
slot_type: gpu_slot_type
visible_gpus: 3
security:
    tls:
        enabled: true
        skip_verify: true
        master_cert: master_certificate_file
        master_cert_name: master_certificate
debug: true
artificial_slots: 12
tls: true
tls_cert: tls_certificate_file
tls_key: tls_key_file
api_enabled: true
bind_ip:  0.0.0.0
bind_port: 9090
http_proxy: determined_http_proxy
https_proxy: determined_https_proxy
ftp_proxy: determined_ftp_proxy
no_proxy: determined_no_proxy
agent_reconnect_attempts: 3
agent_reconnect_backoff: 4
container_runtime: docker_runtime_env
`,
			expected: Options{
				ConfigFile: "agent_config",
				Log: logger.Config{
					Level: "debug",
					Color: false,
				},
				MasterHost:          "master_host_IP",
				MasterPort:          5000,
				AgentID:             "agent_device_name",
				ResourcePool:        "agent_rp",
				ContainerMasterHost: "docker_localhost",
				ContainerMasterPort: 2000,
				SlotType:            "gpu_slot_type",
				VisibleGPUs:         "3",
				Security: SecurityOptions{
					TLS: TLSOptions{
						Enabled:        true,
						SkipVerify:     true,
						MasterCert:     "master_certificate_file",
						MasterCertName: "master_certificate",
					},
				},
				Debug:                  true,
				ArtificialSlots:        12,
				TLS:                    true,
				TLSCertFile:            "tls_certificate_file",
				TLSKeyFile:             "tls_key_file",
				APIEnabled:             true,
				BindIP:                 "0.0.0.0",
				BindPort:               9090,
				HTTPProxy:              "determined_http_proxy",
				HTTPSProxy:             "determined_https_proxy",
				FTPProxy:               "determined_ftp_proxy",
				NoProxy:                "determined_no_proxy",
				AgentReconnectAttempts: 3,
				AgentReconnectBackoff:  4,
				ContainerRuntime:       "docker_runtime_env",
			},
		},
	}

	for _, test := range optionsTests {
		t.Run(test.name, func(t *testing.T) {
			unmarshaled := Options{}
			err := yaml.Unmarshal([]byte(test.raw), &unmarshaled, yaml.DisallowUnknownFields)
			assert.NilError(t, err)
			assert.DeepEqual(t, test.expected, unmarshaled)
		})
	}
}
