// Package etc provides configuration files for setting up common
// system programs like ssh, sshd, bash, notebooks, and tensorboard.
package etc

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

const (
	// CommandEntrypointResource is the pre-flight script for commands.
	CommandEntrypointResource = "command-entrypoint.sh"
	// SSHConfigResource is the template SSH config file.
	SSHConfigResource = "ssh_config"
	// SSHDConfigResource is the template SSHD config file.
	SSHDConfigResource = "sshd_config"
	// ShellEntrypointResource is the script to set up sshd.
	ShellEntrypointResource = "shell-entrypoint.sh"
	// GCCheckpointsEntrypointResource is the script to run checkpoint GC.
	GCCheckpointsEntrypointResource = "gc-checkpoints-entrypoint.sh"
	// NotebookTemplateResource is the template notebook config file.
	NotebookTemplateResource = "notebook-template.ipynb"
	// NotebookEntrypointResource is the script to set up a notebook.
	NotebookEntrypointResource = "notebook-entrypoint.sh"
	// NotebookIdleCheckResource is the script to check if a notebook is idle.
	NotebookIdleCheckResource = "check_idle.py"
	// TaskCheckReadyLogsResource is the script to parse logs to check if a task is ready.
	TaskCheckReadyLogsResource = "check_ready_logs.py"
	// TensorboardEntryScriptResource is the script to set up TensorBoard.
	TensorboardEntryScriptResource = "tensorboard-entrypoint.sh"
	// TrialEntrypointScriptResource is the script to set up a trial.
	TrialEntrypointScriptResource = "entrypoint.sh"
	// AgentSetupScriptTemplateResource is the template for the script to run a dynamic agent.
	AgentSetupScriptTemplateResource = "agent_setup_script.sh.template"
	// K8InitContainerEntryScriptResource is the script to run the init container on k8s.
	K8InitContainerEntryScriptResource = "k8_init_container_entrypoint.sh"
	// TaskLoggingSetupScriptResource is the script to setup prerequistites for logging.
	TaskLoggingSetupScriptResource = "task-logging-setup.sh"
	// TaskLoggingTeardownScriptResource is the script to teardown stuff for logging.
	TaskLoggingTeardownScriptResource = "task-logging-teardown.sh"
	// TaskSignalHandlingScriptResource is the script to teardown stuff for logging.
	TaskSignalHandlingScriptResource = "task-signal-handling.sh"
)

var staticRoot string

// SetRootPath sets the path relative to which the paths for resources are resolved.
func SetRootPath(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	staticRoot = root
	return nil
}

// MustStaticFile returns the content of the file with the provided name as a byte array.
func MustStaticFile(name string) []byte {
	if staticRoot == "" {
		panic("static file root has not been set")
	}
	path := filepath.Join(staticRoot, name)

	// Check that the final path is inside the root directory.
	insideDir, err := filepath.Match(filepath.Join(staticRoot, "*"), path)
	check.Panic(errors.Wrapf(err, "unable to find static file: %s", name))
	check.Panic(
		check.TrueSilent(
			insideDir,
			"attempted to read path outside the static root: %s",
			path,
		),
	)

	bytes, err := ioutil.ReadFile(path) // #nosec G304
	check.Panic(errors.Wrapf(err, "unable to find static file: %s", name))
	return bytes
}
