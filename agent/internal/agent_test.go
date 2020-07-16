package internal

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestNoAddProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	ans := append([]string{}, inputEnv.Env...)

	testAgent.addProxy(&inputEnv)

	// InputEnv should not change because we didn't set any environment variables.
	if v, ok := compareLists(inputEnv.Env, ans); !ok {
		if len(v.extraA) != 0 {
			t.Errorf("Extra variables found in Environment: %v", v.extraA)
		} else {
			t.Errorf("Missing variables in Environment: %v", v.extraB)
		}
	}
}

func TestAddProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	testAgent.Options.HTTPProxy = "192.168.1.1"
	testAgent.Options.HTTPSProxy = "192.168.1.2"
	testAgent.Options.FTPProxy = "192.168.1.3"
	testAgent.Options.NoProxy = "*.test.com"

	ans := append(inputEnv.Env, []string{
		"HTTP_PROXY=192.168.1.1",
		"HTTPS_PROXY=192.168.1.2",
		"FTP_PROXY=192.168.1.3",
		"NO_PROXY=*.test.com",
	}...)

	testAgent.addProxy(&inputEnv)

	if v, ok := compareLists(inputEnv.Env, ans); !ok {
		if len(v.extraA) != 0 {
			t.Errorf("Extra variables found in Environment: %v", v.extraA)
		} else {
			t.Errorf("Missing variables in Environment: %v", v.extraB)
		}
	}
}

func TestAlreadyAddedProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
		"HTTP_PROXY=10.0.0.1",
	}

	testAgent.Options.HTTPProxy = "10.0.0.2"

	// We should be overriding earlier proxy variables
	ans := append(inputEnv.Env, "HTTP_PROXY="+testAgent.Options.HTTPProxy)

	testAgent.addProxy(&inputEnv)

	for i, v := range inputEnv.Env {
		if ans[i] != v {
			t.Errorf("Expected: %v But got: %v", ans, inputEnv.Env)
			return
		}
	}
}

type ListComp struct {
	extraA []string
	extraB []string
}

func compareLists(a []string, b []string) (ListComp, bool) {
	checklistMap := make(map[string]bool)
	output := ListComp{[]string{}, []string{}}
	isEqual := true
	for _, v := range a {
		checklistMap[v] = false
	}

	for _, v := range b {
		if _, ok := checklistMap[v]; ok {
			checklistMap[v] = true
		} else { //b has, but a does not
			output.extraB = append(output.extraB, v)
			isEqual = false
		}
	}

	for k, v := range checklistMap {
		if !v { //a has, but b does not
			output.extraA = append(output.extraA, k)
			isEqual = false
		}
	}

	return output, isEqual
}
