package internal

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestNoAddProxy(t *testing.T) {
	inputEnv := container.Config{}
	testActor := dockerActor{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	extraVars, _, found := currEnv()
	if found {
		inputEnv.Env = append(inputEnv.Env, extraVars...)
	}

	ans := append([]string{}, inputEnv.Env...)

	testActor.addProxy(&inputEnv)

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
	testActor := dockerActor{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	extraVars, extraVarsMap, found := currEnv()
	if found {
		inputEnv.Env = append(inputEnv.Env, extraVars...)
	}

	newVarsMap := map[string]string{
		"HTTP_PROXY": "192.168.1.1", "HTTPS_PROXY": "192.168.1.2",
		"FTP_PROXY": "192.168.1.3", "NO_PROXY": "*.test.com",
	}

	ans := append([]string{}, inputEnv.Env...)

	for k, v1 := range extraVarsMap {
		v2, ok := newVarsMap[k]
		if v1 != "" && ok {
			delete(newVarsMap, k)
		} else if ok {
			ans = append(ans, k+"="+v2)
		}
	}

	for k, v := range newVarsMap {
		err := os.Setenv(k, v)
		if err != nil {
			t.Errorf("Error setting environment variable: %s", k)
		}
	}

	testActor.addProxy(&inputEnv)

	// InputEnv should change since we set some environment variables.
	if v, ok := compareLists(inputEnv.Env, ans); !ok {
		if len(v.extraA) != 0 {
			t.Errorf("Extra variables found in Environment: %v", v.extraA)
		} else {
			t.Errorf("Missing variables in Environment: %v", v.extraB)
		}
	}

	for k := range newVarsMap {
		err := os.Unsetenv(k)
		if err != nil {
			t.Errorf("Error unsetting environment variable: %s", k)
		}
	}
}

func TestAlreadyAddedProxy(t *testing.T) {
	inputEnv := container.Config{}
	testActor := dockerActor{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	extraVars, extraVarsMap, found := currEnv()
	if found {
		inputEnv.Env = append(inputEnv.Env, extraVars...)
	}

	newVarsMap := map[string]string{
		"HTTP_PROXY": "192.168.1.1", "HTTPS_PROXY": "192.168.1.2",
		"FTP_PROXY": "192.168.1.3", "NO_PROXY": "*.test.com",
	}

	alternateVarsMap := map[string]string{
		"HTTP_PROXY": "10.0.0.1", "HTTPS_PROXY": "10.0.0.2",
		"FTP_PROXY": "10.0.0.3", "NO_PROXY": "10.0.0.4",
	}

	ans := append([]string{}, inputEnv.Env...)

	for k, v1 := range extraVarsMap {
		v2, ok := newVarsMap[k]
		if v1 != "" && ok {
			delete(newVarsMap, k)
			delete(alternateVarsMap, k)
		} else if ok {
			ans = append(ans, k+"="+v2)
			inputEnv.Env = append(inputEnv.Env, k+"="+v2)
		}
	}

	for k, v := range alternateVarsMap {
		err := os.Setenv(k, v)
		if err != nil {
			t.Errorf("Error setting environment variable: %s", k)
		}
	}

	testActor.addProxy(&inputEnv)
	fmt.Println(ans)

	// InputEnv should not change because environment already set (with config).
	if v, ok := compareLists(inputEnv.Env, ans); !ok {
		if len(v.extraA) != 0 {
			t.Errorf("Extra variables found in Environment: %v", v.extraA)
		} else {
			t.Errorf("Missing variables in Environment: %v", v.extraB)
		}
	}

	for k := range alternateVarsMap {
		err := os.Unsetenv(k)
		if err != nil {
			t.Errorf("Error unsetting environment variable: %s", k)
		}
	}
}

type ListComp struct {
	extraA []string
	extraB []string
}

// currEnv looks for existing proxy variables in the environment.
// If such variables exist, they get returned so the test case is aware.
func currEnv() ([]string, map[string]string, bool) {
	envMap := map[string]string{
		"HTTP_PROXY": "", "HTTPS_PROXY": "",
		"FTP_PROXY": "", "NO_PROXY": "",
	}
	var output []string
	found := false

	for _, x := range os.Environ() {
		k := strings.SplitN(x, "=", 2)[0]
		if _, ok := envMap[k]; ok {
			output = append(output, x)
			envMap[k] = x
			found = true
		}
	}

	return output, envMap, found
}

// compareLists compares two slices without considering order.
// It returns a ListStatus structure containing extraneous objects.
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
