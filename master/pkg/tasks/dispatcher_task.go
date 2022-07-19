package tasks

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/sirupsen/logrus"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	trueValue = "true"
	// dispatcherEntrypointScriptResource is the script to handle container initialization
	// before transferring to the defined entrypoint script.
	dispatcherEntrypointScriptResource = "dispatcher-wrapper.sh"
	dispatcherEntrypointScriptMode     = 0o700

	// Content managed by dispatcher-wrapper.sh script for container-local volumes.
	determinedLocalFs = "/determined_local_fs"
	// Location of container-local temporary directory.
	containerTmpDeterminedDir = "/determined/"
	singularityCarrier        = "com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm"
	podmanCarrier             = "com.cray.analytics.capsules.carriers.hpc.slurm.PodmanOverSlurm"
)

// The "launcher" is very sensitive when it comes to the payload name. There
// are certain characters, such as parenthesis, commas, spaces, etc, that will
// cause the "launcher" to bomb out during the processing of the manifest.
// Therefore, we'll stick to only alpha-numberic characters, plus dashes and
// underscores. This regular expression is used to filter out all characters
// that are NOT alpha-numberic, dashes, or underscores from the task
// description that we use to construct the payload name. Presently, the task
// description looks something like "exp-118-trial-104", which contains all
// legit characters, but we must protect ourselves from any changes in the
// future which may cause this format to change and introduce, say, parenthesis
// or spaces.
var payloadNameCompiledRegEx = regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)

// ToDispatcherManifest creates the manifest that will be ultimately sent to the launcher.
// Returns:
//	 Manifest, launchingUserName, PayloadName, err
//
// Note: Cannot pass "req *sproto.AllocateRequest" as an argument, as it requires
// import of "github.com/determined-ai/determined/master/internal/sproto", which
// results in an "import cycle not allowed" error.
func (t *TaskSpec) ToDispatcherManifest(
	masterHost string,
	masterPort int,
	certificateName string,
	numSlots int,
	slotType device.Type,
	slurmPartition string,
	tresSupported bool,
	containerRunType string,
) (*launcher.Manifest, string, string, error) {
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
		singularityCarrier,
	})
	if containerRunType == "podman" {
		payload.GetCarriers()[0] = podmanCarrier
	}

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("batch")

	mounts, userWantsDirMountedOnTmp := getDataVolumes(t.Mounts)

	// Use the specified workDir if it is user-specified.
	// If the workdir is the the default (/run/determined/workdir)
	// it does not exist on the launcher node so causes and error log.
	// Instead it will be set dispatcher-wrapper.sh using setting DET_WORKDIR
	// So use /var/tmp here to eliminate spurious error logs.  We avoid using /tmp
	// here because dispatcher-wrapper.sh by default relinks /tmp to
	// a container-private directory and if it is in use we faile with EBUSY.
	workDir := t.WorkDir
	if workDir == DefaultWorkDir {
		workDir = "/var/tmp"
	}

	launchConfig := t.computeLaunchConfig(slotType, workDir, slurmPartition)
	launchParameters.SetConfiguration(*launchConfig)

	// Determined generates tar archives including initialization, garbage collection,
	// and security configuration and then maps them into generic containers when
	// they are launched.   The equivalent capability  is provided by the launcher
	// via the --custom Archive capsules argument.   Encode the archives
	// into a format that can be set as custom launch arguments.
	allArchives := *getAllArchives(t)
	encodedArchiveParams, err := encodeArchiveParameters(
		dispatcherArchive(t.AgentUserGroup,
			generateRunDeterminedLinkNames(allArchives)), allArchives)
	if err != nil {
		return nil, "", "", err
	}
	var slurmArgs []string
	slurmArgs = append(slurmArgs, t.TaskContainerDefaults.Slurm...)
	slurmArgs = append(slurmArgs, t.Environment.Slurm()...)
	logrus.Debugf("Custom slurm arguments: %s", slurmArgs)
	encodedArchiveParams["slurmArgs"] = slurmArgs
	errList := model.ValidateSlurm(slurmArgs)
	if len(errList) > 0 {
		logrus.WithError(errList[0]).Error("Forbidden slurm option specified")
		return nil, "", "", errList[0]
	}
	launchParameters.SetCustom(encodedArchiveParams)

	// Add entrypoint command as argument
	wrappedEntryPoint := append(
		[]string{determinedLocalFs + "/" + dispatcherEntrypointScriptResource},
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
	launchParameters.SetData(mounts)

	envVars, err := getEnvVarsForLauncherManifest(
		t, masterHost, masterPort, certificateName, userWantsDirMountedOnTmp, slotType)
	if err != nil {
		return nil, "", "", err
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
			"total": int32(numSlots),
		})
	}
	// Set the required number of GPUs if the device type is CUDA (Nvidia) or ROCM (AMD).
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

	return &manifest, impersonatedUser, payloadName, err
}

// getAllArchives returns all the experiment archives.
func getAllArchives(t *TaskSpec) *[]cproto.RunArchive {
	r, u := t.Archives()
	allArchives := []cproto.RunArchive{}
	allArchives = append(allArchives, r...)
	allArchives = append(allArchives, u...)
	return &allArchives
}

// computeLaunchConfig computes the launch configuration for the Slurm job manifest.
func (t *TaskSpec) computeLaunchConfig(
	slotType device.Type, workDir string,
	slurmPartition string,
) *map[string]string {
	launchConfig := map[string]string{
		"workingDir":          workDir,
		"enableWritableTmpFs": trueValue,
	}
	if slurmPartition != "" {
		launchConfig["partition"] = slurmPartition
	}
	if slotType == device.CUDA {
		launchConfig["enableNvidia"] = trueValue
	}
	if slotType == device.ROCM {
		launchConfig["enableROCM"] = trueValue
	}
	return &launchConfig
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
		filepath.Join(RunDir, etc.CommandEntrypointResource)) {
		return false
	}
	// The helper scripts are read-only, so leave that archive as shared
	if archiveItem.Archive.ContainsFilePrefix(
		filepath.Join(RunDir, etc.ShellEntrypointResource)) {
		return false
	}
	// We create the run dir (/run/determined) to contain links
	if archiveItem.Path == RunDir || archiveItem.Path == DefaultWorkDir {
		return true
	}
	// If the archive maps content under /run/determined, make a local volume
	if archiveItem.Archive.ContainsFilePrefix(RunDir) ||
		archiveItem.Archive.ContainsFilePrefix(DefaultWorkDir) {
		return true
	}
	return false
}

// Return the archives in an argument format for launcher custom Archive args.
// Encoding the files to Base64 string arguments.
func encodeArchiveParameters(
	dispatcherArchive cproto.RunArchive,
	archives []cproto.RunArchive,
) (map[string][]string, error) {
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
	tmpMount bool, slotType device.Type,
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

	// Some in-container setup in slurm needs to know the slot type to set other envvars correctly.
	m["DET_SLOT_TYPE"] = string(slotType)

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

	// If the user has not configured a bind mount of /tmp trigger
	// dispatcher-wrapper.sh to make it local to the container.
	if !tmpMount {
		m["DET_CONTAINER_LOCAL_TMP"] = "1"
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

	// Do not auto mount the host /tmp within the container
	m["SINGULARITY_NO_MOUNT"] = "tmp"

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

// Provide all task mount points as data volumes, and return true if there is a bind for /tmp
// Launcher requires that a Data object has a name; source, target & read-only are all
// that matter to Singularity.
func getDataVolumes(mounts []mount.Mount) ([]launcher.Data, bool) {
	volumes := []launcher.Data{}
	userWantsDirMountedOnTmp := false

	for i, mount := range mounts {
		volume := *launcher.NewData()
		volume.SetName("ds" + strconv.Itoa(i))
		volume.SetSource(mount.Source)
		volume.SetTarget(mount.Target)
		volume.SetReadOnly(mount.ReadOnly)
		volumes = append(volumes, volume)
		if mount.Target == "/tmp" {
			userWantsDirMountedOnTmp = true
		}
	}

	return volumes, userWantsDirMountedOnTmp
}

// Create a softlink archive entry for the specified file name in the
// '/run/determined' directory to the local container temp version.
func getRunSubdirLink(aug *model.AgentUserGroup, name string) archive.Item {
	return aug.OwnedArchiveItem(RunDir+"/"+name,
		[]byte(containerTmpDeterminedDir+name), 0o700, tar.TypeSymlink)
}

// Return any paths that need to be created within /run/determined
// for unshared directories and files.
func generateRunDeterminedLinkNames(
	archives []cproto.RunArchive,
) []string {
	// Use a map as a set to avoid duplicates
	linksSet := make(map[string]bool)

	for _, archive := range archives {
		// If archive will be in a local volume, determine the required links
		if makeLocalVolume(archive) {
			for _, archiveItem := range archive.Archive {
				filePath := filepath.Join(archive.Path, archiveItem.Path)
				// Not the toplevel runDir, but is under it
				if strings.HasPrefix(filePath, RunDir) && filePath != RunDir {
					contained := strings.TrimPrefix(strings.TrimPrefix(filePath, RunDir), "/")
					// If not a file, then extract the directory name
					if filepath.Base(contained) != contained {
						dir, _ := filepath.Split(contained)
						contained = filepath.Dir(dir)
					}
					linksSet[contained] = true
				}
			}
		}
	}

	// Conver the map keys to the list of link names
	linkNames := []string{}
	for k := range linksSet {
		linkNames = append(linkNames, k)
	}
	return linkNames
}

// Archive with dispatcher wrapper entrypoint script,  /run/determined directory,
// and links for each entry under /run/determined for unshared files/directories.
func dispatcherArchive(aug *model.AgentUserGroup, linksNeeded []string) cproto.RunArchive {
	dispatherArchive := archive.Archive{
		// Add the dispatcher wrapper script
		aug.OwnedArchiveItem(
			determinedLocalFs+"/"+dispatcherEntrypointScriptResource,
			etc.MustStaticFile(dispatcherEntrypointScriptResource),
			dispatcherEntrypointScriptMode,
			tar.TypeReg,
		),
		aug.OwnedArchiveItem(RunDir, nil, 0o700, tar.TypeDir),
	}

	// Create and add each link
	for _, linkName := range linksNeeded {
		dispatherArchive = append(dispatherArchive, getRunSubdirLink(aug, linkName))
		logrus.Tracef("Created link for %s", linkName)
	}

	return wrapArchive(dispatherArchive, "/")
}
