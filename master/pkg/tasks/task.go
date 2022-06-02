package tasks

import (
	"archive/tar"
	"crypto/tls"
	"fmt"
	"strings"

	docker "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// DefaultWorkDir is the default workdir.
	DefaultWorkDir    = "/run/determined/workdir"
	userPythonBaseDir = "/run/determined/pythonuserbase"
	runDir            = "/run/determined"
	infoDir           = "/run/determined/info"
	trainDir          = "/run/determined/train"
	modelCopy         = "/run/determined/train/model"
	rootDir           = "/"
	passwdPath        = "/run/determined/etc/passwd"
	shadowPath        = "/run/determined/etc/shadow"
	groupPath         = "/run/determined/etc/group"
	certPath          = "/run/determined/etc/ssl/master.crt"
)

// TaskSpec defines the spec of a task.
type TaskSpec struct {
	// Fields that are only for task logics.
	Description string
	// LoggingFields are fields to include in each record of structured (i.e., Fluent Bit) logging.
	LoggingFields map[string]string

	// Fields that are set on the cluster level.
	ClusterID   string
	HarnessPath string
	MasterCert  *tls.Certificate
	SSHRsaSize  int

	SegmentEnabled bool
	SegmentAPIKey  string

	// Fields that are set on the per-request basis.
	// TaskContainerDefaults should be removed from TaskSpec once we move to using the same
	// schema for the cluster-level defaults and the request-level configuration.
	TaskContainerDefaults model.TaskContainerDefaultsConfig
	Environment           expconf.EnvironmentConfig
	ResourcesConfig       expconf.ResourcesConfig
	WorkDir               string
	Owner                 *model.User
	AgentUserGroup        *model.AgentUserGroup
	ExtraArchives         []cproto.RunArchive
	ExtraEnvVars          map[string]string
	Entrypoint            []string
	Mounts                []mount.Mount
	// UseHostMode is whether host mode networking would be desirable for this task.
	// This is used by Docker only.
	UseHostMode bool
	ShmSize     int64

	// The parent task of an allocation.
	TaskID string

	// Fields that are set on per-resources basis.
	AllocationID           string
	AllocationSessionToken string
	ResourcesID            string
	ContainerID            string
	Devices                []device.Device

	UserSessionToken string
	TaskType         model.TaskType
}

// ResolveWorkDir resolves the work dir.
func (t *TaskSpec) ResolveWorkDir() {
	agentUser := ""
	detUser := ""
	if t.AgentUserGroup != nil {
		agentUser = t.AgentUserGroup.User
	}
	if t.Owner != nil {
		detUser = t.Owner.Username
	}
	workDir := strings.ReplaceAll(t.WorkDir, "$AGENT_USER", agentUser)
	t.WorkDir = strings.ReplaceAll(workDir, "$DET_USER", detUser)
}

// Archives returns all the archives.
func (t *TaskSpec) Archives() []cproto.RunArchive {
	res := []cproto.RunArchive{
		workDirArchive(t.AgentUserGroup, t.WorkDir, t.WorkDir == DefaultWorkDir),
		runDirHelpersArchive(t.AgentUserGroup),
		injectUserArchive(t.AgentUserGroup, t.WorkDir),
		harnessArchive(t.HarnessPath, t.AgentUserGroup),
		masterCertArchive(t.MasterCert),
	}
	res = append(res, t.ExtraArchives...)
	return res
}

// EnvVars returns all the environment variables.
func (t TaskSpec) EnvVars() map[string]string {
	e := map[string]string{
		// PYTHONUSERBASE allows us to `pip install --user` into a location guaranteed to be owned by
		// the user inside the container.
		"PYTHONUSERBASE":    userPythonBaseDir,
		"DET_TASK_ID":       t.TaskID,
		"DET_ALLOCATION_ID": t.AllocationID,
		"DET_RESOURCES_ID":  t.ResourcesID,
		"DET_CONTAINER_ID":  t.ContainerID,
		"DET_SESSION_TOKEN": t.AllocationSessionToken,
		"DET_USER_TOKEN":    t.UserSessionToken,
		"DET_USER":          t.Owner.Username,
	}

	networkInterface := t.TaskContainerDefaults.DtrainNetworkInterface
	if networkInterface != "" {
		e["DET_INTER_NODE_NETWORK_INTERFACE"] = networkInterface
	}

	if t.MasterCert != nil {
		e["DET_USE_TLS"] = "true"
		e["DET_MASTER_CERT_FILE"] = certPath
	} else {
		e["DET_USE_TLS"] = "false"
	}

	e["DET_SEGMENT_ENABLED"] = fmt.Sprintf("%v", t.SegmentEnabled)
	if t.SegmentEnabled {
		e["DET_SEGMENT_API_KEY"] = t.SegmentAPIKey
	}

	for k, v := range t.ExtraEnvVars {
		e[k] = v
	}
	return e
}

<<<<<<< HEAD
=======
// ToDispatcherManifest creates the manifest that will be ultimately sent to the launcher.
//
// Note #1: We'll need to do some significant changes to this method to deal with the
// GPU allocation (HAL-2780). Right now we're happy with being able to send a request
// to the launcher and getting back a Dispatch ID. We'll deal with the GPU allocation
// as a separate work item.
//
// Note #2: Cannot pass "req *sproto.AllocateRequest" as an argument, as it requires
// import of "github.com/determined-ai/determined/master/internal/sproto", which
// results in an "import cycle not allowed" error.
//
// The below TODOs document where we lack feature parity with Determined proper, and the work items
// needed for parity. Please expand this list as you find things.
// TODO(HAL-2864): Include t.LoggingFields so the wrapper script can structure the log output.
// TODO(HAL-2865): Support configurable /dev/shm sizes.
// TODO(HAL-2867): Support mounting arbitrary devices into containers.
func (t *TaskSpec) ToDispatcherManifest(
	masterHost string,
	masterPort int,
	certificateName string,
	numSlots int,
	slotType device.Type,
	slurmPartition string,
	tresSupported bool) (*launcher.Manifest, string, error) {
	/*
	 * The user that the "launcher" is going to run the Determined task
	 * container as.  Eventually, the impersonated user will likely come from the
	 * UID and GID that's embedded in the authentication token. But, since we're
	 * not performing authentication currently, pending HAL-2746, we'll just let
	 * the impersonated user be accepted by the "launcher" without worrying about
	 * the lack of security.
	 */
	impersonatedUser := ""

	/*
	 * The "AgentUserGroup.User" will be the username of the user who we will be
	 * launching the Determined task container as.  In launcher lingo, this will
	 * be the "impersonated" user. There needs to be a mapping of the Determined
	 * user to the username that we wish to launch the Determined task container
	 * as.  This mapping can be done via the following command, for example:
	 *
	 * det user link-with-agent-user --agent-uid 504 \
	 *     --agent-gid 20 \
	 *     --agent-user crayuser \
	 *     --agent-group staff \
	 *     determined
	 *
	 * where "determined" is the name of the Determined user and "crayuser" is
	 * the user we're going to be impersonating.
	 *
	 * Note that the command above needs to be run as a privileged Determined
	 * user, such as the "admin" user, so you may need to switch users in order
	 * to execute the command.  For example,
	 *
	 * det user login admin
	 *
	 */
	if t.AgentUserGroup != nil {
		impersonatedUser = t.AgentUserGroup.User
	}

	payloadName := getPayloadName(t)

	// Create a payload
	payload := launcher.NewPayloadWithDefaults()

	payload.SetName(payloadName)
	payload.SetId("com.cray.analytics.capsules.generic.container")
	payload.SetVersion("latest")

	payload.SetCarriers([]string{
		"com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm",
	})

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("batch")

	// Use the specified workDir if it is user-specified.
	// If the workdir is the the default (/run/determined/workdir)
	// it does not exist on the launcher node so causes and error log.
	// Instead it will be set dispatcher-wrapper.sh using setting DET_WORKDIR
	// So use /tmp here to eliminate spurious error logs.
	workDir := t.WorkDir
	if workDir == DefaultWorkDir {
		workDir = "/tmp"
	}

	launchParameters.SetConfiguration(map[string]string{
		"workingDir":          workDir,
		"enableNvidia":        "true", // triggers 'singularity run --nv ...'
		"enableWritableTmpFs": "true", // Make container filesystem writable (needed for /determined)
	})
	if slurmPartition != "" {
		launchParameters.GetConfiguration()["partition"] = slurmPartition
	}

	// Determined generates tar archives including initialization, garbage collection,
	// and security configuration and then maps them into generic containers when
	// they are launched.   The equivalent capability  is provided by the launcher
	// via the --custom Archive capsules argument.   Encode the archives
	// into a format that can be set as custom launch arguments.
	encodedArchiveParams, err := encodeArchiveParameters(
		dispatcherArchive(t.AgentUserGroup,
			generateRunDeterminedLinkNames(t.Archives())), t.Archives())
	if err != nil {
		return nil, "", err
	}
	var slurmArgs []string
	slurmArgs = append(slurmArgs, t.TaskContainerDefaults.Slurm...)
	slurmArgs = append(slurmArgs, t.Environment.Slurm()...)
	logrus.Debugf("Custom slurm arguments: %s", slurmArgs)
	encodedArchiveParams["slurmArgs"] = slurmArgs
	errList := model.ValidateSlurm(slurmArgs)
	if len(errList) > 0 {
		logrus.WithError(errList[0]).Error("Forbidden slurm option specified")
		return nil, "", errList[0]
	}
	launchParameters.SetCustom(encodedArchiveParams)

	// Add entrypoint command as argument
	wrappedEntryPoint := append(
		[]string{determinedLocalFs + "/" + etc.DispatcherEntrypointScriptResource},
		t.Entrypoint...)
	launchParameters.SetArguments(wrappedEntryPoint)

	// We just pass through the image reference here.  It may be any scheme that
	// singularity supports including (docker, library, file path, etc).   If
	// a docker reference without scheme (the default), the launcher will attempt
	// to match to a locally cached image.
	launchParameters.SetImages(map[string]string{
		"default": t.Environment.Image().For(slotType),
	})

	// Add some data volumes
	launchParameters.SetData(getDataVolumes(t.Mounts))

	envVars, err := getEnvVarsForLauncherManifest(t, masterHost, masterPort, certificateName)
	if err != nil {
		return nil, "", err
	}

	launchParameters.SetEnvironment(envVars)

	payload.SetLaunchParameters(*launchParameters)

	// Create payload resource requirements
	resources := launcher.NewResourceRequirementsWithDefaults()

	// One task per node.
	if tresSupported || numSlots == 0 {
		resources.SetInstances(map[string]int32{"per-node": 1})
	} else {
		// When tresSupported==false then we can't use --gpus in slurm, so map the total nodes to
		// the total GPUs which will cause launcher to map SetGpus below into --gres:gpus.
		resources.SetInstances(map[string]int32{
			"nodes": int32(numSlots),
			"total": int32(numSlots)})
	}
	// Set the required number of GPUs if the device type is CUDA (Nvidia) or RCOM (AMD).
	if slotType == device.CUDA || slotType == device.ROCM {
		resources.SetGpus(map[string]int32{"total": int32(numSlots)})
	} else {
		resources.SetCores(map[string]float32{"total": float32(numSlots)})
	}

	payload.SetResourceRequirements(*resources)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName("det")

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata) // Manifest | The manifest to launch
	manifest.SetPayloads([]launcher.Payload{*payload})
	// manifest.SetManifestVersion("latest") //?

	return &manifest, impersonatedUser, err
}

// Return true if the archive specified should be treated
// as per-process and not a shared volume for all processes.
// Unless configured in this list, all items are shared.  It
// saves additional softlinks if we properly identify read-only
// scripts below, but it does not cause breakage if we miss one.
func makeLocalVolume(archiveItem cproto.RunArchive) bool {
	// We cannot clone the ssh config because sshd will not process softlinks
	if archiveItem.Archive.ContainsFilePrefix(sshDir) {
		return false
	}
	// The helper scripts are read-only, so leave that archive as shared
	if archiveItem.Archive.ContainsFilePrefix(etc.TaskLoggingSetupScriptResource) {
		return false
	}
	// The helper scripts are read-only, so leave that archive as shared
	if archiveItem.Archive.ContainsFilePrefix(
		filepath.Join(runDir, etc.CommandEntrypointResource)) {
		return false
	}
	// The helper scripts are read-only, so leave that archive as shared
	if archiveItem.Archive.ContainsFilePrefix(
		filepath.Join(runDir, etc.ShellEntrypointResource)) {
		return false
	}
	// We create the run dir (/run/determined) to contain links
	if archiveItem.Path == runDir {
		return true
	}
	// If the archive maps content under /run/determined, make a local volume
	if archiveItem.Archive.ContainsFilePrefix(runDir) {
		return true
	}
	return false
}

// Return the archives in an argument format for launcher custom Archive args.
// Encoding the files to Base64 string arguments.
func encodeArchiveParameters(
	dispatcherArchive cproto.RunArchive,
	archives []cproto.RunArchive) (map[string][]string, error) {
	// Insert the dispatcherArchive into the list for processing (first in list)
	archives = append([]cproto.RunArchive{dispatcherArchive}, archives...)
	archiveStrings := make([]string, len(archives))

	for idx, archiveItem := range archives {
		runDirPrefix := ""
		// Other than the dispatcherArchive (first in list), if the archive provides files
		// that should be local per-container instance copies, redirect to the /dispatcher
		// directory for processing during container initialization.
		if idx != 0 && makeLocalVolume(archiveItem) {
			runDirPrefix = determinedLocalFs
		}
		bytesString, err := archive.ToRelocatedTarGz(
			runDirPrefix+archiveItem.Path+"/",
			archiveItem.Archive)
		if err != nil {
			logrus.Error("Failure to create TarGz Archive", err)
			return nil, err
		}
		archiveStrings[idx] = base64.StdEncoding.EncodeToString(bytesString)
	}

	customArgs := make(map[string][]string)
	customArgs["Archives"] = archiveStrings
	return customArgs, nil
}

// Gets the environment variables that are to be added to the Launcher's manifest.
func getEnvVarsForLauncherManifest(
	taskSpec *TaskSpec, masterHost string, masterPort int, certificateName string,
) (map[string]string, error) {
	// Hash map containing the environment variables.
	m := make(map[string]string)

	// These represent the environment variables that are set by Determined AI.
	for k, v := range taskSpec.EnvVars() {
		m[k] = v
	}

	// For some reason, getting the user-defined environment variable requires a device type.
	// Merely copying the same code that's in "ToDockerSpec()" without fully understanding
	// the connection between the deviceType and the user-defined environment variables.
	deviceType := device.CPU

	if len(taskSpec.Devices) > 0 {
		deviceType = taskSpec.Devices[0].Type
	}

	// The user-defined environment variables, if any. These come from the experiment's
	// YAML file.  For example,
	//
	// environment:
	//   image: "environment:cuda-11.2-tf-2.5-gpu-0.17.7.sif"
	//   environment_variables:
	//     - DETECTRON2_DATASETS=/mnt/dtrain-fsx/detectron2
	//     - MY_ENV_VAR1=abc
	//     - MY_ENV_VAR2=xyz
	envVars := taskSpec.Environment.EnvironmentVariables().For(deviceType)

	// Add each user-defined environment variable to the map.
	for _, s := range envVars {
		tokens := strings.Split(s, "=")

		if len(tokens) > 1 {
			m[tokens[0]] = tokens[1]
		} else {
			return nil, fmt.Errorf("invalid user-defined environment variable '%s'", s)
		}
	}

	// These environment variables are required in "harness/determined/_info.py". If
	// they are not set, then task container will fail.
	m["DET_MASTER"] = fmt.Sprintf("%s:%d", masterHost, masterPort)
	m["DET_MASTER_HOST"] = masterHost
	m["DET_MASTER_IP"] = masterHost
	m["DET_MASTER_PORT"] = fmt.Sprintf("%d", masterPort)
	m["DET_CONTAINER_ID"] = taskSpec.ContainerID
	m["DET_CLUSTER_ID"] = taskSpec.ClusterID
	// On non-zero exit of any component/step of the sbatch job, terminate with an error
	m["SLURM_KILL_BAD_EXIT"] = "1"

	// The "entrypoint.sh" script that's mounted by the Singularity task container
	// will set the DET_SLOT_IDS environment variable when it sees that DET_AGENT_ID is
	// set to "launcher". So, if you change the value here, you also need to make the
	// corresponding change to "entrypoint.sh".
	m["DET_AGENT_ID"] = "launcher"

	// The "master/internal/resourcemanagers/kubernetes/spec.go" checks if the
	// certificate name is set before assigning it to an environment variable, so
	// we're duplicating that same behavior here.
	if certificateName != "" {
		m["DET_MASTER_CERT_NAME"] = certificateName
	}

	if taskSpec.Environment.RegistryAuth() != nil {
		m["SINGULARITY_DOCKER_USERNAME"] = taskSpec.Environment.RegistryAuth().Username
		m["SINGULARITY_DOCKER_PASSWORD"] = taskSpec.Environment.RegistryAuth().Password
		if len(taskSpec.Environment.RegistryAuth().ServerAddress) > 0 {
			logrus.Warningf(
				"NOT SUPPORTED: environment.registry_auth.serveraddress: %s ",
				taskSpec.Environment.RegistryAuth().ServerAddress)
		}
		if len(taskSpec.Environment.RegistryAuth().Email) > 0 {
			logrus.Warningf(
				"NOT SUPPORTED: environment.registry_auth.email: %s ",
				taskSpec.Environment.RegistryAuth().Email)
		}
	}

	if taskSpec.Environment.ForcePullImage() {
		m["SINGULARITY_DISABLE_CACHE"] = trueValue
	}

	if len(taskSpec.Environment.AddCapabilities()) > 0 {
		m["SINGULARITY_ADD_CAPS"] = strings.Join(taskSpec.Environment.AddCapabilities(), ",")
	}

	if len(taskSpec.Environment.DropCapabilities()) > 0 {
		m["SINGULARITY_DROP_CAPS"] = strings.Join(taskSpec.Environment.DropCapabilities(), ",")
	}

	return m, nil
}

// Assigns the name for the payload we're going to send to the launcher. It's up for
// debate, but I figured we'd give the payload a name that we can associate with the
// experiment that's being run to allow us to better debug problems when associating
// what's in the launcher's log file to what the determined log file may have.
//
// For example, if I'm running the "determined-ee/examples/computer_vision/cifar10_pytorch"
// experiment, and that creates an experiment #107, then the payload name would be:
//
// DAI-task-container_exp-118-trial-104
//
// The launcher, or whatever is processing the manifest sent to the launcher, doesn't
// like certain characters in the name, such as spaces, colons, or commas.
func getPayloadName(taskSpec *TaskSpec) string {
	payloadName := "ai"

	// Remove all characters that are not alpha-numberic, dashes, or spaces.
	experimentDescription := payloadNameCompiledRegEx.ReplaceAllString(taskSpec.Description, "")

	if len(experimentDescription) > 0 {
		payloadName += "_" + experimentDescription
	}

	return payloadName
}

// Provide all task mount points as data volumes.
// Launcher requires that a Data object has a name; source, target & read-only are all
// that matter to Singularity.
func getDataVolumes(mounts []mount.Mount) []launcher.Data {
	volumes := []launcher.Data{}

	for i, mount := range mounts {
		var volume = *launcher.NewData()
		volume.SetName("ds" + strconv.Itoa(i))
		volume.SetSource(mount.Source)
		volume.SetTarget(mount.Target)
		volume.SetReadOnly(mount.ReadOnly)
		volumes = append(volumes, volume)
	}

	return volumes
}

>>>>>>> fd381c494 (feat: FOUNDENG-6 Add slurm options to task_container_defaults)
// ToDockerSpec converts a task spec to a docker container spec.
func (t *TaskSpec) ToDockerSpec() cproto.Spec {
	var envVars []string
	for k, v := range t.EnvVars() {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	env := t.Environment
	deviceType := device.CPU
	if len(t.Devices) > 0 {
		deviceType = t.Devices[0].Type
	}
	envVars = append(envVars, env.EnvironmentVariables().For(deviceType)...)

	network := t.TaskContainerDefaults.NetworkMode
	if t.UseHostMode {
		network = hostMode
	}

	shmSize := t.ShmSize
	if shmSize == 0 {
		shmSize = t.TaskContainerDefaults.ShmSizeBytes
	}

	resources := t.ResourcesConfig
	var devices []docker.DeviceMapping
	for _, device := range resources.Devices() {
		devices = append(devices, docker.DeviceMapping{
			PathOnHost:        device.HostPath(),
			PathInContainer:   device.ContainerPath(),
			CgroupPermissions: device.Mode(),
		})
	}

	spec := cproto.Spec{
		TaskType: string(t.TaskType),
		PullSpec: cproto.PullSpec{
			Registry:  env.RegistryAuth(),
			ForcePull: env.ForcePullImage(),
		},
		RunSpec: cproto.RunSpec{
			ContainerConfig: docker.Config{
				User:         getUser(t.AgentUserGroup),
				ExposedPorts: toPortSet(env.Ports()),
				Env:          envVars,
				Cmd:          t.Entrypoint,
				Image:        env.Image().For(deviceType),
				WorkingDir:   t.WorkDir,
			},
			HostConfig: docker.HostConfig{
				NetworkMode:     network,
				Mounts:          t.Mounts,
				PublishAllPorts: true,
				ShmSize:         shmSize,
				CapAdd:          env.AddCapabilities(),
				CapDrop:         env.DropCapabilities(),

				Resources: docker.Resources{
					Devices: devices,
				},
			},
			Archives:         t.Archives(),
			UseFluentLogging: true,
		},
	}

	return spec
}

// workDirArchive ensures that the workdir is created and owned by the user.
func workDirArchive(
	aug *model.AgentUserGroup, workDir string, createWorkDir bool,
) cproto.RunArchive {
	a := archive.Archive{
		aug.OwnedArchiveItem(runDir, nil, 0700, tar.TypeDir),
		aug.OwnedArchiveItem(infoDir, nil, 0755, tar.TypeDir),
		aug.OwnedArchiveItem(userPythonBaseDir, nil, 0700, tar.TypeDir),
	}
	if createWorkDir {
		a = append(a, aug.OwnedArchiveItem(workDir, nil, 0700, tar.TypeDir))
	}
	return wrapArchive(a, rootDir)
}

// runDirHelpersArchive ensures helper scripts exist in the run dir.
func runDirHelpersArchive(aug *model.AgentUserGroup) cproto.RunArchive {
	return wrapArchive(archive.Archive{
		aug.OwnedArchiveItem(
			taskLoggingSetupScript,
			etc.MustStaticFile(etc.TaskLoggingSetupScriptResource),
			taskLoggingSetupMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskLoggingTeardownScript,
			etc.MustStaticFile(etc.TaskLoggingTeardownScriptResource),
			taskLoggingTeardownMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(
			taskSignalHandlingScript,
			etc.MustStaticFile(etc.TaskSignalHandlingScriptResource),
			taskSignalHandlingMode,
			tar.TypeReg,
		),
	}, runDir)
}

// injectUserArchive creates the user/UID/group/GID for a user by adding passwd/shadow/group files
// to /run/determined/etc, which will be read by libnss_determined inside the container. If
// libnss_determined is not present in the container, these files will be simply ignored and some
// non-root container features will not work properly.
func injectUserArchive(aug *model.AgentUserGroup, workDir string) cproto.RunArchive {
	passwdBytes := []byte(
		fmt.Sprintf("%v:x:%v:%v::%v:/bin/bash\n", aug.User, aug.UID, aug.GID, workDir),
	)
	shadowBytes := []byte(fmt.Sprintf("%v:!!:::::::\n", aug.User))
	groupBytes := []byte(fmt.Sprintf("%v:x:%v:\n", aug.Group, aug.GID))

	return wrapArchive(
		archive.Archive{
			archive.RootItem(passwdPath, passwdBytes, 0644, tar.TypeReg),
			archive.RootItem(shadowPath, shadowBytes, 0600, tar.TypeReg),
			archive.RootItem(groupPath, groupBytes, 0644, tar.TypeReg),
		},
		rootDir,
	)
}

func getUser(agentUserGroup *model.AgentUserGroup) string {
	if agentUserGroup == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", agentUserGroup.UID, agentUserGroup.GID)
}
