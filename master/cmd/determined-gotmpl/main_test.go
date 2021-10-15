package main

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestRunTmpl(t *testing.T) {
	expected := `
resource_manager:
  type: agent
  default_aux_resource_pool: aux-pool
  default_compute_resource_pool: compute-pool
  scheduler:
    type: fair_share

resource_pools:
  - pool_name: compute-pool-solo
    max_aux_containers_per_agent: 0
    provider:
      instance_type:
        machine_type: n1-standard-4
        gpu_type: nvidia-tesla-k80
        gpu_num: 1
        preemptible: false
      cpu_slots_allowed: true
      type: gcp
`
	expected = strings.Trim(expected, " \n\t")

	template := "./main_test.tmpl"

	data := map[string]interface{}{} // sprig dict functions require map[string]interface{}
	data["resource_manager"] = map[string]interface{}{
		"scheduler": map[string]interface{}{
			"type": "fair_share",
		},
	}
	data["resource_pools"] = map[string]interface{}{
		"pools": map[string]interface{}{
			"compute_pool": map[string]interface{}{
				"instance_type": map[string]interface{}{
					"machine_type": "n1-standard-4",
					"gpu_type":     "nvidia-tesla-k80",
					"gpu_num":      0,
					"preemptible":  false,
				},
			},
		},
		"gcp": map[string]interface{}{
			"type": "gcp",
		},
	}
	files := []string{template}

	b := RunTmpl(data, files)
	result := strings.Trim(b.String(), " \n\t")
	assert.Equal(t, result, expected)
}
