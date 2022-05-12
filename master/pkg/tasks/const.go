package tasks

const (
	trialEntrypointFile = "/run/determined/train/entrypoint.sh"
	trialEntrypointMode = 0744

	taskLoggingSetupScript = "task-logging-setup.sh"
	taskLoggingSetupMode   = 0744

	taskLoggingTeardownScript = "task-logging-teardown.sh"
	taskLoggingTeardownMode   = 0744

	taskSignalHandlingScript = "task-signal-handling.sh"
	taskSignalHandlingMode   = 0744

	// Put as many ssh-related files in /run/determined as possible. In particular, it is very
	// important that we don't overwrite the user's host $HOME/.ssh/id_rsa, if the user happens to
	// mount their host $HOME into the container's $HOME. Since we control the invocation of sshd,
	// we can keep our sshd_config in a location not likely to be mounted by users.
	trialAuthorizedKeysFile = "/run/determined/ssh/authorized_keys"
	trialAuthorizedKeysMode = 0600

	// horovodrun controls how ssh is invoked, and we are force to overwrite a default ssh
	// configuration file.
	trialSSHConfigFile = "/etc/ssh/ssh_config"
	trialSSHConfigMode = 0644

	// Shared SSHD configuration.
	sshDir         = "/run/determined/ssh"
	sshDirMode     = 0700
	sshdConfigFile = "/run/determined/ssh/sshd_config"
	sshdConfigMode = 0600
	privKeyFile    = "/run/determined/ssh/id_rsa"
	privKeyMode    = 0600
	pubKeyFile     = "/run/determined/ssh/id_rsa.pub"
	pubKeyMode     = 0600

	shellAuthorizedKeysFile = "/run/determined/ssh/authorized_keys_unmodified"
)
