package configpolicy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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

var structExperiment = ExperimentConfigPolicies{
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
	Constraints: &model.Constraints{
		ResourceConstraints: &model.ResourceConstraints{
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
		output *ExperimentConfigPolicies
	}{
		{"valid yaml", yamlExperiment + yamlConstraints, true, &structExperiment},
		{"just config", yamlExperiment, true, &justConfig},
		{"just constraints", yamlConstraints, true, &justConstraints},
		{"extra fields", yamlExperiment + `  extra_field: "string"` + yamlConstraints, false, nil},
		{"invalid fields", yamlExperiment + "  debug true\n", false, nil},
		{"empty input", "", true, &ExperimentConfigPolicies{}},
		{"null/empty fields", yamlExperiment + "  debug:\n", true, &justConfig},
		{"wrong field type", "invariant_config:\n  debug: 3\n", false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalConfigPolicy[ExperimentConfigPolicies](tt.input, InvalidExperimentConfigPolicyErr)
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

var structNTSC = NTSCConfigPolicies{
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
	Constraints: &model.Constraints{
		ResourceConstraints: &model.ResourceConstraints{
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
		output *NTSCConfigPolicies
	}{
		{"valid yaml", yamlNTSC + yamlConstraints, true, &structNTSC},
		{"just config", yamlNTSC, true, &justConfig},
		{"just constraints", yamlConstraints, true, &justConstraints},
		{"extra fields", yamlNTSC + `  extra_field: "string"` + yamlConstraints, false, nil},
		{"invalid fields", yamlNTSC + "  debug true\n", false, nil},
		{"empty input", "", true, &NTSCConfigPolicies{}},
		{"null/empty fields", yamlNTSC + "  debug:\n", true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", "invariant_config:\n  debug: 3\n", false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalConfigPolicy[NTSCConfigPolicies](tt.input, InvalidNTSCConfigPolicyErr)
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
		output *ExperimentConfigPolicies
	}{
		{"valid json", `{` + jsonExperiment + `,` + jsonConstraints + `}`, true, &structExperiment},
		{"just config", `{` + jsonExperiment + `}`, true, &justConfig},
		{"just constraints", `{` + jsonConstraints + `}`, true, &justConstraints},
		{"extra fields", jsonExtraField, false, nil},
		{"invalid fields", jsonInvalidField, false, nil},
		{"empty input", "", true, &ExperimentConfigPolicies{}},
		{"null/empty fields", jsonEmptyField, true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", jsonWrongFieldType, false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalConfigPolicy[ExperimentConfigPolicies](tt.input, InvalidExperimentConfigPolicyErr)
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
		output *NTSCConfigPolicies
	}{
		{"valid json", `{` + jsonNTSC + `,` + jsonConstraints + `}`, true, &structNTSC},
		{"just config", `{` + jsonNTSC + `}`, true, &justConfig},
		{"just constraints", `{` + jsonConstraints + `}`, true, &justConstraints},
		{"extra fields", jsonExtraField, false, nil},
		{"invalid fields", jsonInvalidField, false, nil},
		{"empty input", "", true, &NTSCConfigPolicies{}},
		{"null/empty fields", jsonEmptyField, true, &justConfig}, // empty fields unmarshal to default value
		{"wrong field type", jsonWrongFieldType, false, nil},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := UnmarshalConfigPolicy[NTSCConfigPolicies](tt.input, InvalidNTSCConfigPolicyErr)
			require.Equal(t, tt.noErr, err == nil)
			if tt.noErr {
				assert.DeepEqual(t, tt.output, config)
			}
		})
	}
}

func TestCheckConstraintsConflicts(t *testing.T) {
	constraints := &model.Constraints{
		ResourceConstraints: &model.ResourceConstraints{
			MaxSlots: ptrs.Ptr(10),
		},
		PriorityLimit: ptrs.Ptr(50),
	}
	t.Run("max_slots differs to high", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, ptrs.Ptr(11), ptrs.Ptr(5), nil)
		require.Error(t, err)
	})
	t.Run("max_slots differs to low", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, ptrs.Ptr(9), ptrs.Ptr(5), nil)
		require.Error(t, err)
	})

	t.Run("slots_per_trial too high", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, ptrs.Ptr(5), ptrs.Ptr(11), nil)
		require.Error(t, err)
	})

	t.Run("slots_per_trial within range", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, ptrs.Ptr(10), ptrs.Ptr(8), nil)
		require.NoError(t, err)
	})

	t.Run("priority differs too high", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, nil, nil, ptrs.Ptr(100))
		require.Error(t, err)
	})

	t.Run("priority differs too low", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, nil, nil, ptrs.Ptr(10))
		require.Error(t, err)
	})

	t.Run("all comply", func(t *testing.T) {
		err := checkConstraintConflicts(constraints, ptrs.Ptr(10), ptrs.Ptr(10), ptrs.Ptr(50))
		require.NoError(t, err)
	})
}

func TestValidateConfigs(t *testing.T) {
	t.Run("exp global config conflicts with global constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(nil,
			`
invariant_config:
  resources:
    max_slots: 3

constraints:
  resources:
    max_slots: 10
`, nil,
		)
		require.Error(t, err)
	})

	t.Run("ntsc global config conflicts with global constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(nil,
			`
invariant_config:
  resources:
    max_slots: 3

constraints:
  resources:
    max_slots: 10
`, nil,
		)
		require.Error(t, err)
	})

	t.Run("exp global config complies with constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(nil,
			`
invariant_config:
  resources:
    max_slots: 10

constraints:
  resources:
    max_slots: 10
`, nil,
		)
		require.NoError(t, err)
	})

	t.Run("ntsc global config complies with constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(nil,
			`
invariant_config:
  resources:
    max_slots: 10

constraints:
  resources:
    max_slots: 10
`, nil,
		)
		require.Error(t, err) // stub CM-493
	})

	t.Run("exp workspace config complies with global constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(&model.TaskConfigPolicies{
			WorkloadType: model.ExperimentType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 15
`, nil,
		)
		require.NoError(t, err)
	})

	t.Run("ntsc workspace config complies with global constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(&model.TaskConfigPolicies{
			WorkloadType: model.NTSCType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
max_slots: 15
`, nil,
		)
		require.Error(t, err) // stub CM-493
	})

	t.Run("exp workspace config complies with workspace and global constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(&model.TaskConfigPolicies{
			WorkloadType: model.ExperimentType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 15

constraints:
  resources:
    max_slots: 15
`, nil,
		)
		require.NoError(t, err)
	})

	t.Run("ntsc workspace config complies with workspace and global constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(&model.TaskConfigPolicies{
			WorkloadType: model.NTSCType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 15

constraints:
  resources:
    max_slots: 15
`, nil,
		)
		require.Error(t, err) // stub CM-493
	})

	// Workspace invariant config complies with workspace constraints, violates global constraints
	t.Run("exp workspace config violates global constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(&model.TaskConfigPolicies{
			WorkloadType: model.ExperimentType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 8

constraints:
  resources:
    max_slots: 8
`, nil,
		)
		require.Error(t, err)
	})

	t.Run("ntsc workspace config violates global constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(&model.TaskConfigPolicies{
			WorkloadType: model.NTSCType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 8

constraints:
  resources:
    max_slots: 8
`, nil,
		)
		require.Error(t, err)
	})

	// Workspace constraints conflicts with global constraints
	t.Run("exp workspace constraints conflict with global constraints", func(t *testing.T) {
		err := ValidateExperimentConfig(&model.TaskConfigPolicies{
			WorkloadType: model.ExperimentType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 15

constraints:
  resources:
    max_slots: 8
`, nil,
		)
		require.Error(t, err)
	})

	t.Run("ntsc workspace constraints conflict with global constraints", func(t *testing.T) {
		err := ValidateNTSCConfig(&model.TaskConfigPolicies{
			WorkloadType: model.NTSCType,
			Constraints: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 15

constraints:
  resources:
    max_slots: 8
`, nil,
		)
		require.Error(t, err)
	})

	t.Run("exp workspace invariant config different from global", func(t *testing.T) {
		err := ValidateExperimentConfig(&model.TaskConfigPolicies{
			WorkloadType: model.ExperimentType,
			InvariantConfig: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 12
`, nil,
		)
		require.NoError(t, err)
	})

	t.Run("ntsc workspace invariant config different from global", func(t *testing.T) {
		err := ValidateNTSCConfig(&model.TaskConfigPolicies{
			WorkloadType: model.NTSCType,
			InvariantConfig: ptrs.Ptr(`
resources:
  max_slots: 15
`),
		},
			`
invariant_config:
  resources:
    max_slots: 12
`, nil,
		)
		require.Error(t, err) // stub CM-493
	})
}

func TestCanSetMaxSlots(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)
	ctx := context.Background()
	w := createWorkspaceWithUser(ctx, t, user.ID)
	t.Run("nil slots request", func(t *testing.T) {
		canSetReqSlots, slots, err := CanSetMaxSlots(nil, w.ID)
		require.NoError(t, err)
		require.Nil(t, slots)
		require.True(t, canSetReqSlots)
	})

	err := SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
		WorkspaceID:   &w.ID,
		WorkloadType:  model.ExperimentType,
		LastUpdatedBy: user.ID,
		InvariantConfig: ptrs.Ptr(`
{
	"resources": {
		"max_slots": 13
	}
}
`),
		Constraints: ptrs.Ptr(`
{
	"resources": {
		"max_slots": 13
	}
}
`),
	})
	require.NoError(t, err)

	t.Run("slots different than config higher", func(t *testing.T) {
		canSetReqSlots, slots, err := CanSetMaxSlots(ptrs.Ptr(15), w.ID)
		require.NoError(t, err)
		require.True(t, canSetReqSlots)
		require.NotNil(t, slots)
		require.Equal(t, 13, *slots)
	})

	t.Run("slots different than config lower", func(t *testing.T) {
		canSetReqSlots, slots, err := CanSetMaxSlots(ptrs.Ptr(10), w.ID)
		require.NoError(t, err)
		require.True(t, canSetReqSlots)
		require.NotNil(t, slots)
		require.Equal(t, 13, *slots)
	})

	t.Run("just constarints slots higher", func(t *testing.T) {
		err := SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
			WorkspaceID:   &w.ID,
			WorkloadType:  model.ExperimentType,
			LastUpdatedBy: user.ID,
			Constraints: ptrs.Ptr(`
	{
		"resources": {
			"max_slots": 23
		}
	}
	`),
		})

		canSetReqSlots, slots, err := CanSetMaxSlots(ptrs.Ptr(25), w.ID)
		require.ErrorContains(t, err, SlotsReqTooHighErr)
		require.False(t, canSetReqSlots)
		require.Nil(t, slots)
	})

	t.Run("just constarints slots lower", func(t *testing.T) {
		err := SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
			WorkspaceID:   &w.ID,
			WorkloadType:  model.ExperimentType,
			LastUpdatedBy: user.ID,
			Constraints: ptrs.Ptr(`
	{
		"resources": {
			"max_slots": 23
		}
	}
	`),
		})

		canSetReqSlots, slots, err := CanSetMaxSlots(ptrs.Ptr(20), w.ID)
		require.NoError(t, err)
		require.True(t, canSetReqSlots)
		require.NotNil(t, slots)
		require.Equal(t, 20, *slots)
	})
}
