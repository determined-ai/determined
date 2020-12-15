package kubernetes

import (
	"archive/tar"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	localhost    = "localhost"
	ipv4Loopback = "127.0.0.1"

	fluentBaseDir = "/run/determined/fluent/"
)

// fluentConfig computes the command-line arguments and extra files needed to start Fluent Bit with
// an appropriate configuration.
func fluentConfig(
	masterHost string,
	masterPort int,
	fields map[string]string,
	// TODO Figure out how to configure this.
	tlsCert *tls.Certificate,
	loggingConfig model.LoggingConfig,
) ([]string, archive.Archive, error) {
	const luaPath = "tonumber.lua"
	const configPath = "fluent.conf"
	const parserConfigPath = "parsers.conf"
	// const masterCertPath = "master.crt"
	const elasticCertPath = "elastic.crt"

	var files archive.Archive

	luaCode := `
-- Do some tweaking of values that can't be expressed with the normal filters.
function run(tag, timestamp, record)
    record.rank_id = tonumber(record.rank_id)
    record.trial_id = tonumber(record.trial_id)

    -- TODO: Only do this if it's not a partial record.
    if (record.log == nil) then
        record.log = '\n'
    else
        record.log = record.log .. '\n'
    end

    return 2, timestamp, record
end
`
	files = append(files,
		archive.Item{
			Path:     luaPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(luaCode),
		},
	)

	parserConfig := `
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
`

	files = append(files,
		archive.Item{
			Path:     parserConfigPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(parserConfig),
		},
	)

	baseConfig := fmt.Sprintf(`
[SERVICE]
  # Flush every .05 seconds to reduce latency for users.
  Flush .05
  Parsers_File %s

[INPUT]
  Name tail
  Path /run/determined/train/logs/stdout.log-rotate/current
  Refresh_Interval 3
  Read_From_Head true
  Buffer_Max_Size 1M
  Skip_Long_Lines On
  Tag stdout

[INPUT]
  Name tail
  Path /run/determined/train/logs/stderr.log-rotate/current
  Refresh_Interval 3
  Read_From_Head true
  Buffer_Max_Size 1M
  Skip_Long_Lines On
  Tag stderr
`, fluentBaseDir+parserConfigPath)

	var fieldsStrs []string
	for k, v := range fields {
		fieldsStrs = append(fieldsStrs, fmt.Sprintf("  Add %s %s", k, v))
	}

	filterConfig := fmt.Sprintf(`
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

# Move around fields to create the desired shape of object.
[FILTER]
  Name modify
  Match *
%s

[FILTER]
  Name modify
  Match stdout
  Add stdtype stdout

[FILTER]
  Name modify
  Match stderr
  Add stdtype stderr

# Apply the Lua code for miscellaneous field tweaking.
[FILTER]
  Name lua
  Match *
  Script %s
  Call run
`, strings.Join(fieldsStrs, "\n"), luaPath)

	var outputConfig string
	const (
		tlsOn         = "  tls On\n"
		tlsVerifyOff  = "  tls.verify Off\n"
		tlsCaCertFile = "  tls.ca_file %s\n"
	)
	switch {
	case loggingConfig.DefaultLoggingConfig != nil:
		// HACK: If a host resolves to both IPv4 and IPv6 addresses, Fluent Bit seems to only try IPv6 and
		// fail if that connection doesn't work. IPv6 doesn't play well with Docker and many Linux
		// distributions ship with an `/etc/hosts` that maps "localhost" to both 127.0.0.1 (IPv4) and
		// [::1] (IPv6), so Fluent Bit will break when run in host mode. To avoid that, translate
		// "localhost" diretcly into an IP address before passing it to Fluent Bit.
		if masterHost == localhost {
			masterHost = ipv4Loopback
			// TODO: Handle cert name.
		}

		outputConfig = fmt.Sprintf(`
# Temporary for the PR.
[OUTPUT]
  Name stdout
  Match *

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
`, masterHost, masterPort)

		// const
		// if tlsCert != nil {
		// 	outputConfig += tlsOn
		// 	outputConfig += fmt.Sprintf(tlsCaCertFile, masterCertPath)
		// 	// TODO

		// 	files = append(files,
		// 		archive.Item{
		// 			Path:     masterCertPath,
		// 			Type:     tar.TypeReg,
		// 			FileMode: 0444,
		// 			Content:  certBytes,
		// 		},
		// 	)
		// }
	case loggingConfig.ElasticLoggingConfig != nil:
		elasticOpts := loggingConfig.ElasticLoggingConfig

		fluentElasticHost := elasticOpts.Host
		// HACK: Also a hack, described above in detail.
		if fluentElasticHost == localhost {
			fluentElasticHost = ipv4Loopback
		}

		outputConfig = fmt.Sprintf(`
[OUTPUT]
  Name  es
  Match *
  Host  %s
  Port  %d
  Logstash_Format True
  Logstash_Prefix triallogs
  Time_Key timestamp
  Time_Key_Nanos On
`, fluentElasticHost, elasticOpts.Port)

		elasticSecOpts := elasticOpts.Security
		if elasticSecOpts.Username != nil && elasticSecOpts.Password != nil {
			outputConfig += fmt.Sprintf(`
  HTTPUser   %s
  HTTPPasswd %s
`, *elasticOpts.Security.Username, *elasticOpts.Security.Password)
		}

		if elasticSecOpts.TLS.Enabled {
			outputConfig += tlsOn

			if elasticSecOpts.TLS.SkipVerify {
				outputConfig += tlsVerifyOff
			}

			if elasticSecOpts.TLS.CertBytes != nil {
				outputConfig += fmt.Sprintf(tlsCaCertFile, fluentBaseDir+elasticCertPath)
				files = append(files,
					archive.Item{
						Path:     elasticCertPath,
						Type:     tar.TypeReg,
						FileMode: 0444,
						Content:  elasticSecOpts.TLS.CertBytes,
					},
				)
			}
		}

	default:
		panic("no log driver set for agent")
	}

	files = append(files,
		archive.Item{
			Path:     configPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(baseConfig + filterConfig + outputConfig),
		})

	args := []string{"/fluent-bit/bin/fluent-bit", "-c", fluentBaseDir + configPath}

	return args, files, nil
}
