package configpolicy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestValidWorkloadType(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid experiment type", "EXPERIMENT", true},
		{"valid ntsc type", "NTSC", true},
		{"invalid type", "EXPERIMENTS", false},
		{"lowercase", "experiment", false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			valid := ValidWorkloadType(tt.input)
			require.Equal(t, tt.valid, valid)
		})
	}
}

const yamlConstraints = `
constraints:
  resources:
    max_slots: 4
  priority_limit: 10
`

const yamlExperiment = `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - "cap1"
      - "cap2"
  resources:
    slots: 1
`

var (
	description   = "test\nspecial\tchar"
	slots         = 1
	forcePull     = false
	maxSlots      = 4
	priorityLimit = 10
)

var structExperiment = ExperimentConfigPolicy{
	InvariantConfig: &expconf.ExperimentConfig{
		RawDescription: &description,
		RawResources: &expconf.ResourcesConfig{
			RawSlots: &slots,
		},
		RawEnvironment: &expconf.EnvironmentConfigV0{
			RawForcePullImage:  &forcePull,
			RawAddCapabilities: []string{"cap1", "cap2"},
		},
	},
	Constraints: &Constraints{
		ResourceConstraints: &ResourceConstraints{
			MaxSlots: &maxSlots,
		},
		PriorityLimit: &priorityLimit,
	},
}

func TestUnmarshalYamlExperiment(t *testing.T) {
	justConfig := structExperiment
	justConfig.Constraints = nil
	justConstraints := structExperiment
	justConstraints.InvariantConfig = nil

	testCases := []struct {
		name   string
		input  string
		noErr  bool
		output *ExperimentConfigPolicy
	}{
		{"valid yaml", yamlExperiment + yamlConstraints, true, &structExperiment},
		{"just config", yamlExperiment, true, &justConfig},
		{"just constraints", yamlConstraints, true, &justConstraints},
		{"extra fields", yamlExperiment + `  extra_field: "string"` + yamlConstraints, false, nil},
		{"invalid fields", yamlExperiment + "  debug true\n", false, nil},
		{"empty input", "", true, &ExperimentConfigPolicy{}},
		{"null/empty fields", yamlExperiment + "  debug:\n", true, &justConfig},
		{"wrong field type", "invariant_config:\n  debug: 3\n", false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalExperimentConfigPolicy(tt.input)
			require.Equal(t, tt.noErr, err == nil)
			if tt.noErr {
				assert.DeepEqual(t, tt.output, config)
			}
		})
	}
}

const yamlNTSC = `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - "cap1"
      - "cap2"
  resources:
    slots: 1
`

var structNTSC = NTSCConfigPolicy{
	InvariantConfig: &model.CommandConfig{
		Description: "test\nspecial\tchar",
		Resources: model.ResourcesConfig{
			Slots: 1,
		},
		Environment: model.Environment{
			ForcePullImage:  false,
			AddCapabilities: []string{"cap1", "cap2"},
		},
	},
	Constraints: &Constraints{
		ResourceConstraints: &ResourceConstraints{
			MaxSlots: &maxSlots,
		},
		PriorityLimit: &priorityLimit,
	},
}

func TestUnmarshalYamlNTSC(t *testing.T) {
	justConfig := structNTSC
	justConfig.Constraints = nil
	justConstraints := structNTSC
	justConstraints.InvariantConfig = nil

	testCases := []struct {
		name   string
		input  string
		noErr  bool
		output *NTSCConfigPolicy
	}{
		{"valid yaml", yamlNTSC + yamlConstraints, true, &structNTSC},
		{"just config", yamlNTSC, true, &justConfig},
		{"just constraints", yamlConstraints, true, &justConstraints},
		{"extra fields", yamlNTSC + `  extra_field: "string"` + yamlConstraints, false, nil},
		{"invalid fields", yamlNTSC + "  debug true\n", false, nil},
		{"empty input", "", true, &NTSCConfigPolicy{}},
		{"null/empty fields", yamlNTSC + "  debug:\n", true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", "invariant_config:\n  debug: 3\n", false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalNTSCConfigPolicy(tt.input)
			require.Equal(t, tt.noErr, err == nil)
			if tt.noErr {
				assert.DeepEqual(t, tt.output, config)
			}
		})
	}
}

const jsonConstraints = `
    "constraints": {
        "resources": {
            "max_slots": 4
        },
        "priority_limit": 10
    }
`

const jsonExperiment = `
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        }
    }
`

const jsonExtraField = `{
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "extra_field": "test"
    }
}
`

const jsonInvalidField = `{
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "debug"=true
    }
}`

const jsonEmptyField = `{
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "debug":,
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        }
    }
}`

const jsonWrongFieldType = `{
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "debug": 4
    }
}`

func TestUnmarshalJSONExperiment(t *testing.T) {
	justConfig := structExperiment
	justConfig.Constraints = nil
	justConstraints := structExperiment
	justConstraints.InvariantConfig = nil

	testCases := []struct {
		name   string
		input  string
		noErr  bool
		output *ExperimentConfigPolicy
	}{
		{"valid json", `{` + jsonExperiment + `,` + jsonConstraints + `}`, true, &structExperiment},
		{"just config", `{` + jsonExperiment + `}`, true, &justConfig},
		{"just constraints", `{` + jsonConstraints + `}`, true, &justConstraints},
		{"extra fields", jsonExtraField, false, nil},
		{"invalid fields", jsonInvalidField, false, nil},
		{"empty input", "", true, &ExperimentConfigPolicy{}},
		{"null/empty fields", jsonEmptyField, true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", jsonWrongFieldType, false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalExperimentConfigPolicy(tt.input)
			require.Equal(t, tt.noErr, err == nil)
			if tt.noErr {
				assert.DeepEqual(t, tt.output, config)
			}
		})
	}
}

const jsonNTSC = `
    "invariant_config": {
        "description": "test\nspecial\tchar",
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        }
    }
`

func TestUnmarshalJSONNTSC(t *testing.T) {
	justConfig := structNTSC
	justConfig.Constraints = nil
	justConstraints := structNTSC
	justConstraints.InvariantConfig = nil

	testCases := []struct {
		name   string
		input  string
		noErr  bool
		output *NTSCConfigPolicy
	}{
		{"valid json", `{` + jsonNTSC + `,` + jsonConstraints + `}`, true, &structNTSC},
		{"just config", `{` + jsonNTSC + `}`, true, &justConfig},
		{"just constraints", `{` + jsonConstraints + `}`, true, &justConstraints},
		{"extra fields", jsonExtraField, false, nil},
		{"invalid fields", jsonInvalidField, false, nil},
		{"empty input", "", true, &NTSCConfigPolicy{}},
		{"null/empty fields", jsonEmptyField, true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", jsonWrongFieldType, false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalNTSCConfigPolicy(tt.input)
			require.Equal(t, tt.noErr, err == nil)
			if tt.noErr {
				assert.DeepEqual(t, tt.output, config)
			}
		})
	}
}
