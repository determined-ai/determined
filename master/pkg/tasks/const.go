package tasks

const (
	trialEntrypointFile = "/run/determined/train/entrypoint.sh"
	trialEntrypointMode = 0o744

	// SingularityEntrypointWrapperScript is just the name of the singularity entrypoint wrapper.
	SingularityEntrypointWrapperScript = "singularity-entrypoint-wrapper.sh"
	singularityEntrypointWrapperMode   = 0o744

	taskSetupScript = "task-setup.sh"
	taskSetupMode   = 0o744

	taskShipLogsShell     = "ship-logs.sh"
	taskShipLogsShellMode = 0o755

	taskShipLogsPython     = "ship_logs.py"
	taskShipLogsPythonMode = 0o755

	// Put as many ssh-related files in /run/determined as possible. In particular, it is very
	// important that we don't overwrite the user's host $HOME/.ssh/id_rsa, if the user happens to
	// mount their host $HOME into the container's $HOME. Since we control the invocation of sshd,
	// we can keep our sshd_config in a location not likely to be mounted by users.
	trialAuthorizedKeysFile = "/run/determined/ssh/authorized_keys"
	trialAuthorizedKeysMode = 0o600

	// horovodrun controls how ssh is invoked, and we are force to overwrite a default ssh
	// configuration file.
	trialSSHConfigFile = "/etc/ssh/ssh_config"
	trialSSHConfigMode = 0o644

	// Shared SSHD configuration.
	sshDir         = "/run/determined/ssh"
	sshDirMode     = 0o700
	sshdConfigFile = "/run/determined/ssh/sshd_config"
	sshdConfigMode = 0o600
	privKeyFile    = "/run/determined/ssh/id_rsa"
	privKeyMode    = 0o600
	pubKeyFile     = "/run/determined/ssh/id_rsa.pub"
	pubKeyMode     = 0o600

	shellAuthorizedKeysFile = "/run/determined/ssh/authorized_keys_unmodified"
)
