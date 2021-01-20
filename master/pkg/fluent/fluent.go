package fluent

import (
	"fmt"
	"strings"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	localhost    = "localhost"
	ipv4Loopback = "127.0.0.1"
)

// ConfigItem describes one line in a Fluent Bit config file.
type ConfigItem struct {
	Name, Value string
}

// ConfigSection describes the contents of one section (input, output, parser, etc.) in a Fluent Bit
// config file.
type ConfigSection []ConfigItem

func (c ConfigSection) String() string {
	var lines []string
	for _, item := range c {
		lines = append(lines, fmt.Sprintf("  %s %s", item.Name, item.Value))
	}
	return strings.Join(lines, "\n")
}

func makeOutputConfig(
	config *strings.Builder,
	files map[string][]byte,
	masterHost string,
	masterPort int,
	loggingConfig model.LoggingConfig,
	tlsConfig model.TLSClientConfig,
) {
	switch {
	case loggingConfig.DefaultLoggingConfig != nil:
		// HACK: If a host resolves to both IPv4 and IPv6 addresses, Fluent Bit seems to only try IPv6 and
		// fail if that connection doesn't work. IPv6 doesn't play well with Docker and many Linux
		// distributions ship with an `/etc/hosts` that maps "localhost" to both 127.0.0.1 (IPv4) and
		// [::1] (IPv6), so Fluent Bit will break when run in host mode. To avoid that, translate
		// "localhost" diretcly into an IP address before passing it to Fluent Bit.
		if masterHost == localhost {
			masterHost = ipv4Loopback
			if tlsConfig.CertificateName == "" {
				tlsConfig.CertificateName = localhost
			}
		}

		fmt.Fprintf(config, `
[OUTPUT]
  Name http
  Match *
  Host %s
  Port %d
  URI /trial_logs
  Header_tag X-Fluent-Tag
  Format json
  Json_date_key timestamp
  Json_date_format iso8601
  storage.total_limit_size 1G
`, masterHost, masterPort)

	case loggingConfig.ElasticLoggingConfig != nil:
		elasticOpts := loggingConfig.ElasticLoggingConfig

		fluentElasticHost := elasticOpts.Host
		// HACK: Also a hack, described above in detail.
		if fluentElasticHost == localhost {
			fluentElasticHost = ipv4Loopback
		}

		fmt.Fprintf(config, `
[OUTPUT]
  Name  es
  Match *
  Host  %s
  Port  %d
  Logstash_Format True
  Logstash_Prefix determined-triallogs
  Time_Key timestamp
  Time_Key_Nanos On
  storage.total_limit_size 1G
`, fluentElasticHost, elasticOpts.Port)

		elasticSecOpts := elasticOpts.Security
		if elasticSecOpts.Username != nil && elasticSecOpts.Password != nil {
			fmt.Fprintf(config, `
  HTTP_User   %s
  HTTP_Passwd %s
`, *elasticOpts.Security.Username, *elasticOpts.Security.Password)
		}

	default:
		panic("no log driver set for agent")
	}

	const (
		tlsOn         = "  tls On\n"
		tlsVerifyOff  = "  tls.verify Off\n"
		tlsCaCertFile = "  tls.ca_file %s\n"
		tlsVhost      = "  tls.vhost %s\n"
		certPath      = "host.crt"
	)

	if tlsConfig.Enabled {
		fmt.Fprint(config, tlsOn)

		if tlsConfig.SkipVerify {
			fmt.Fprint(config, tlsVerifyOff)
		}
		if tlsConfig.CertBytes != nil {
			fmt.Fprintf(config, tlsCaCertFile, certPath)
			files[certPath] = tlsConfig.CertBytes
		}
		if a := tlsConfig.CertificateName; a != "" {
			fmt.Fprintf(config, tlsVhost, a)
		}
	}
}

// ContainerConfig computes the command-line arguments and extra files needed to start Fluent Bit
// with an appropriate configuration. The files are returned as a map from name to content; the
// contents should be placed into files in the same directory and Fluent Bit should be started in
// that directory.
func ContainerConfig(
	masterHost string,
	masterPort int,
	inputs []ConfigSection,
	filters []ConfigSection,
	loggingConfig model.LoggingConfig,
	tlsConfig model.TLSClientConfig,
) ([]string, map[string][]byte) {
	const luaPath = "tonumber.lua"
	const configPath = "fluent.conf"
	const parserConfigPath = "parsers.conf"

	files := make(map[string][]byte)

	files[luaPath] = []byte(`
-- Do some tweaking of values that can't be expressed with the normal filters.
function run(tag, timestamp, record)
    record.rank_id = tonumber(record.rank_id)
    record.trial_id = tonumber(record.trial_id)

    -- TODO: Only do this if it's not a partial record.
    record.log = (record.log or '') .. '\n'

    return 2, timestamp, record
end
`)

	files[parserConfigPath] = []byte(`
[PARSER]
  Name rank_id
  Format regex
  # Look for a rank ID from the beginning of the line (e.g., "[rank=0] xxx").
  Regex ^\[rank=(?<rank_id>([0-9]+))\] (?<log>.*)

[PARSER]
  Name log_level
  Format regex
  # Look for a log level at the start of the line (e.g., "INFO: xxx").
  Regex ^(?<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)): (?<log>.*)
`)

	var config strings.Builder

	fmt.Fprintf(&config, `
[SERVICE]
  # Flush every .05 seconds to reduce latency for users.
  Flush .05
  Parsers_File %s
  storage.path /var/log/flb-buffers/
`, parserConfigPath)

	for _, input := range inputs {
		fmt.Fprintf(&config, `[INPUT]
%s
`, input.String())
	}

	fmt.Fprintf(&config, `
# Attempt to parse the rank ID and log level out of output lines.
[FILTER]
  Name parser
  Match *
  Key_Name log
  Parser rank_id
  Reserve_Data true

[FILTER]
  Name parser
  Match *
  Key_Name log
  Parser log_level
  Reserve_Data true
`)

	for _, filter := range filters {
		fmt.Fprintf(&config, `[FILTER]
%s
`, filter.String())
	}

	fmt.Fprintf(&config, `
# Apply the Lua code for miscellaneous field tweaking.
[FILTER]
  Name lua
  Match *
  Script %s
  Call run
`, luaPath)

	makeOutputConfig(&config, files, masterHost, masterPort, loggingConfig, tlsConfig)

	files[configPath] = []byte(config.String())

	args := []string{"/fluent-bit/bin/fluent-bit", "-c", configPath}

	return args, files
}
