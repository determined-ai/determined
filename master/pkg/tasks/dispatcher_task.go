package tasks

import (
	"archive/tar"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/sirupsen/logrus"
	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	trueValue   = "true"
	tmp         = "/tmp"
	varTmp      = "/var/tmp"
	singularity = "singularity"
	podman      = "podman"
	enroot      = "enroot"
	// dispatcherEntrypointScriptResource is the script to handle container initialization
	// before transferring to the defined entrypoint script.
	dispatcherEntrypointScriptResource = "dispatcher-wrapper.sh"
	dispatcherEntrypointScriptMode     = 0o700

	// Content managed by dispatcher-wrapper.sh script for container-local volumes.
	determinedLocalFs = "/determined_local_fs"
	// Location of container-local temporary directory.
	containerTmpDeterminedDir = "/determined/"
	singularityCarrierSlurm   = "com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm"
	podmanCarrierSlurm        = "com.cray.analytics.capsules.carriers.hpc.slurm.PodmanOverSlurm"
	enrootCarrierSlurm        = "com.cray.analytics.capsules.carriers.hpc.slurm.EnrootOverSlurm"
	singularityCarrierPbs     = "com.cray.analytics.capsules.carriers.hpc.pbs.SingularityOverPbs"
	podmanCarrierPbs          = "com.cray.analytics.capsules.carriers.hpc.pbs.PodmanOverPbs"
	enrootCarrierPbs          = "com.cray.analytics.capsules.carriers.hpc.pbs.EnrootOverPbs"
	unspecifiedSlotsPerNode   = 0

	pbsDisallowedProjectCharacters = "/[]\";:|<>+,?*"

	// ManifestName is the name used by DAI RM when creating HPC job manifests.
	ManifestName = "det"
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
//
//	Manifest, launchingUserName, PayloadName, err
//
// Note: Cannot pass "req *sproto.AllocateRequest" as an argument, as it requires
// import of "github.com/determined-ai/determined/master/internal/sproto", which
// results in an "import cycle not allowed" error.
func (t *TaskSpec) ToDispatcherManifest(
	syslog *logrus.Entry,
	allocationID string,
	masterHost string,
	masterPort int,
	certificateName string,
	numSlots int,
	slotType device.Type,
	slurmPartition string,
	tresSupported bool,
	gresSupported bool,
	containerRunType string,
	isPbsLauncher bool,
	labelMode *string,
	disabledNodes []string,
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

	// will add case for enroot over pbs
	switch {
	case isPbsLauncher && containerRunType == podman:
		payload.SetCarriers([]string{podmanCarrierPbs})
	case !isPbsLauncher && containerRunType == podman:
		payload.SetCarriers([]string{podmanCarrierSlurm})
	case isPbsLauncher && containerRunType == singularity:
		payload.SetCarriers([]string{singularityCarrierPbs})
	case !isPbsLauncher && containerRunType == singularity:
		payload.SetCarriers([]string{singularityCarrierSlurm})
	case isPbsLauncher && containerRunType == enroot:
		payload.SetCarriers([]string{enrootCarrierPbs})
	case !isPbsLauncher && containerRunType == enroot:
		payload.SetCarriers([]string{enrootCarrierSlurm})
	default:
		payload.SetCarriers([]string{singularityCarrierSlurm})
	}

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("batch")

	mounts, userWantsDirMountedOnTmp, varTmpExists, err := getDataVolumes(t.Mounts)
	if err != nil {
		return nil, "", "", err
	}

	// When the container run type is enroot, we need a binding for the
	// "/var/tmp" folder.
	// Check if the container run type is enroot and that "/var/tmp" is not
	// already defined.
	// If so, addTmpFs will add the binding for the "/var/tmp" folder.
	if containerRunType == enroot && !varTmpExists {
		mounts = addTmpFs(mounts, "varTmp", varTmp)
	}

	/*
	 * We need a per-container-private link directory to host /run/determined.
	 * This is the target of a number of softlinks that are remapped to per-container
	 * disk location for each rank.
	 * Singularity/PodMan use /
	 * On Enroot, use /tmp (/var/tmp is not writable by default -- we could enable this,
	 * but will require a custom tmpfs mount)
	 */
	localTmp := "/"
	if containerRunType == enroot {
		localTmp = varTmp
	}

	// Use the specified workDir if it is user-specified.
	// If the workdir is the the default (/run/determined/workdir)
	// it does not exist on the launcher node so causes and error log.
	// Instead it will be set dispatcher-wrapper.sh using setting DET_WORKDIR
	// So use /var/tmp here to eliminate spurious error logs.  We avoid using /tmp
	// here because dispatcher-wrapper.sh by default relinks /tmp to
	// a container-private directory and if it is in use we faile with EBUSY.
	// nolint:dupword
	workDir := t.WorkDir
	if workDir == DefaultWorkDir {
		workDir = varTmp
	}

	launchConfig := t.computeLaunchConfig(syslog,
		allocationID, slotType, workDir, slurmPartition,
		containerRunType, impersonatedUser)
	launchParameters.SetConfiguration(*launchConfig)

	// Determined generates tar archives including initialization, garbage collection,
	// and security configuration and then maps them into generic containers when
	// they are launched.   The equivalent capability  is provided by the launcher
	// via the --custom Archive capsules argument.   Encode the archives
	// into a format that can be set as custom launch arguments.
	allArchives := *getAllArchives(t)
	customParams, err := encodeArchiveParameters(
		syslog,
		allocationID,
		dispatcherArchive(syslog, allocationID, t.AgentUserGroup,
			generateRunDeterminedLinkNames(allArchives), localTmp+"/"), allArchives)
	if err != nil {
		return nil, "", "", err
	}

	pbsProj, slurmProj := t.jobAndProjectLabels(labelMode)

	resources := t.computeResources(syslog, allocationID, tresSupported, numSlots,
		slotType, gresSupported, isPbsLauncher)

	var slurmArgs []string
	if !isPbsLauncher && len(disabledNodes) > 0 {
		slurmArgs = append(slurmArgs, "--exclude="+strings.Join(disabledNodes, ","))
	}

	slurmArgs = append(slurmArgs, t.SlurmConfig.SbatchArgs()...)

	syslog.WithField("allocation-id", allocationID).Debugf("Custom slurm arguments: %s", slurmArgs)

	errList := ValidateSlurm(slurmArgs)
	if len(errList) > 0 {
		syslog.WithField("allocation-id", allocationID).
			WithError(errList[0]).Error("Forbidden slurm option specified")
		return nil, "", "", errList[0]
	}
	slurmArgs = append(slurmArgs, slurmProj...)
	customParams["slurmArgs"] = removeDuplicates(slurmArgs)

	var pbsArgs []string

	pbsArgs = append(pbsArgs, t.PbsConfig.SbatchArgs()...)
	syslog.WithField("allocation-id", allocationID).Debugf("Custom pbs arguments: %s", pbsArgs)
	errList = ValidatePbs(pbsArgs)
	if len(errList) > 0 {
		syslog.WithField("allocation-id", allocationID).
			WithError(errList[0]).Error("Forbidden PBS option specified")
		return nil, "", "", errList[0]
	}
	pbsArgs = append(pbsArgs, pbsProj...)
	customParams["pbsArgs"] = removeDuplicates(pbsArgs)

	if containerRunType == podman {
		portMappings := *getPortMappings(t)
		if len(portMappings) != 0 {
			customParams["ports"] = portMappings
		}
	}

	launchParameters.SetCustom(customParams)

	// Add entrypoint command as argument
	wrappedEntryPoint := append(
		[]string{determinedLocalFs + "/" + dispatcherEntrypointScriptResource},
		t.Entrypoint...)
	launchParameters.SetArguments(wrappedEntryPoint)

	// We just pass through the image reference here.  It may be any scheme that
	// singularity supports including (docker, library, file path, etc).   If
	// a docker reference without scheme (the default), the launcher will attempt
	// to match to a locally cached image.
	image := t.Environment.Image().For(slotType)
	if len(image) == 0 {
		return nil, "", "", fmt.Errorf("no image is configured for slot_type: %s", slotType)
	}
	launchParameters.SetImages(map[string]string{
		"default": image,
	})

	// Add some data volumes
	launchParameters.SetData(mounts)

	envVars, err := getEnvVarsForLauncherManifest(
		syslog, allocationID,
		t, masterHost, masterPort, certificateName, userWantsDirMountedOnTmp,
		slotType, containerRunType, localTmp, t.slotsPerNode(isPbsLauncher))
	if err != nil {
		return nil, "", "", err
	}

	launchParameters.SetEnvironment(envVars)

	payload.SetLaunchParameters(*launchParameters)

	payload.SetResourceRequirements(*resources)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName(ManifestName)

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata) // Manifest | The manifest to launch
	manifest.SetPayloads([]launcher.Payload{*payload})

	return &manifest, impersonatedUser, payloadName, err
}

// WarnUnsupportedOptions gives warnings for user configurations that
// are not supported by HPC launcher.
func (t *TaskSpec) WarnUnsupportedOptions(
	userConfiguredPriority bool,
	containerRunType string,
) string {
	var warnings []string

	ignored := "is ignored with the HPC launcher"
	if t.ResourcesConfig.MaxSlots() != nil {
		warnings = append(warnings, fmt.Sprintf("resources.max_slots %s", ignored))
	}

	// Dispatcher RM had recorded if user had configured resources.priority before it
	// modified the resources.prioroty value. Here instead of checking t.ResourcesConfig.Priority(),
	// we check the flag userConfiguredPriority.
	if userConfiguredPriority {
		warnings = append(warnings, fmt.Sprintf("resources.priority %s", ignored))
	}
	if t.ResourcesConfig.Weight() != 1 {
		// DAI set default weight value to 1 when not specified
		warnings = append(warnings, fmt.Sprintf("resources.weight %s", ignored))
	}

	notSupported := "is not supported with Singularity"
	if containerRunType == podman {
		notSupported = "is not supported with Podman"
	} else if containerRunType == enroot {
		notSupported = "is not supported with Enroot"
	}
	if len(t.ResourcesConfig.Devices()) > 0 {
		warnings = append(warnings, fmt.Sprintf("resources.devices %s", notSupported))
	}
	if t.ResourcesConfig.ShmSize() != nil {
		warnings = append(warnings, fmt.Sprintf("resources.shm_size %s", notSupported))
	}

	if t.Environment.RegistryAuth() != nil {
		if containerRunType == podman {
			warnings = append(warnings, fmt.Sprintf("environment.registry_auth %s", notSupported))
		} else {
			if len(t.Environment.RegistryAuth().ServerAddress) > 0 {
				warnings = append(warnings,
					fmt.Sprintf("environment.registry_auth.serveraddress %s", notSupported))
			}
			if len(t.Environment.RegistryAuth().Email) > 0 {
				warnings = append(warnings, fmt.Sprintf("environment.registry_auth.email %s", notSupported))
			}
		}
	}

	return strings.Join(warnings, "\n")
}

// removeDuplicates removes duplicated sbatch args from sbatchArgs array.
func removeDuplicates(sbatchArgs []string) (result []string) {
	argsMap := make(map[string]bool)
	for _, arg := range sbatchArgs {
		if !argsMap[arg] {
			argsMap[arg] = true
			result = append(result, arg)
		}
	}
	return result
}

// jobAndProjectLabels returns as command options the strings necessary to label
// the job in the specified mode.
func (t *TaskSpec) jobAndProjectLabels(mode *string) (pbsResult, slurmResult []string) {
	switch {
	case (mode == nil || *mode == config.Project):
		return computeJobProjectResult(t.Project)
	case *mode == config.Workspace:
		return computeJobProjectResult(t.Workspace)
	case *mode == config.Label:
		return computeJobProjectResultForLabels(t.Labels, "")
	case strings.HasPrefix(*mode, config.LabelPrefix):
		prefix := strings.TrimPrefix(*mode, config.LabelPrefix)
		return computeJobProjectResultForLabels(t.Labels, prefix)
	}
	return pbsResult, slurmResult
}

func computeJobProjectResult(labelValue string) (pbsResult, slurmResult []string) {
	if len(labelValue) == 0 {
		return slurmResult, pbsResult
	}
	slurmResult = append(slurmResult, formatSlurmLabelResult(labelValue))
	pbsResult = append(pbsResult, formatPbsLabelResult(labelValue))
	return pbsResult, slurmResult
}

func computeJobProjectResultForLabels(
	labels []string, prefix string,
) (pbsResult, slurmResult []string) {
	if len(labels) == 0 {
		return pbsResult, slurmResult
	}
	var labelNames []string
	for _, labelName := range labels {
		if prefix != "" && !strings.HasPrefix(labelName, prefix) {
			continue
		}
		labelName = strings.TrimPrefix(labelName, prefix)
		labelNames = append(labelNames, labelName)
	}
	if len(labelNames) == 0 {
		return pbsResult, slurmResult
	}
	sort.Strings(labelNames) // to make the tests more reliable
	slurmResult = append(slurmResult, formatSlurmLabelResult(strings.Join(labelNames, ",")))
	pbsResult = append(pbsResult, formatPbsLabelResult(strings.Join(labelNames, "_")))
	return pbsResult, slurmResult
}

func formatPbsLabelResult(label string) string {
	label = strings.Map(mapPbsInvalidChars, label)
	return fmt.Sprintf("-P %s", addQuotes(label))
}

// mapPbsInvalidChars maps any character not valid for a PBS project name to the '_'.
func mapPbsInvalidChars(in rune) rune {
	if strings.ContainsRune(pbsDisallowedProjectCharacters, in) {
		return '_'
	}
	return in
}

func formatSlurmLabelResult(label string) string {
	return fmt.Sprintf("--wckey=%s", addQuotes(label))
}

func addQuotes(label string) string {
	if len(label) > 0 {
		// Remove any surrounding double quotes.
		label = strings.Trim(label, "\"")
	}

	// Surround the string with quotes and escape any embedded quotes.
	return strconv.Quote(label)
}

// computeResources calculates the job resource requirements. It also returns any
// additional qualifiers required for the desired scheduling behavior (required
// for Slurm only at the time of writing).
func (t *TaskSpec) computeResources(
	syslog *logrus.Entry,
	allocationID string,
	tresSupported bool,
	numSlots int,
	slotType device.Type,
	gresSupported bool,
	isPbsLauncher bool,
) *launcher.ResourceRequirements {
	slotsPerNode := t.slotsPerNode(isPbsLauncher)
	haveSlotsPerNode := slotsPerNode != unspecifiedSlotsPerNode

	numNodes := numSlots
	effectiveSlotsPerNode := 1
	if haveSlotsPerNode {
		numNodes = (numSlots + slotsPerNode - 1) / slotsPerNode
		effectiveSlotsPerNode = slotsPerNode
	}

	syslog.WithField("allocation-id", allocationID).
		Debugf("slotsPerNode: %d, numNodes: %d, eSlotsPerNode: %d",
			slotsPerNode, numNodes, effectiveSlotsPerNode)

	resources := launcher.NewResourceRequirementsWithDefaults()
	switch {
	case slotType == device.CPU:
		// Checkpoint GC tasks will always request zero slots and have a device
		// type of CPU. While we could simply check for a "t.TaskType" equal to
		// "CHECKPOINT_GC", there may be other use cases where the number of
		// requested slots is zero, so we check for that instead.
		if numSlots == 0 {
			numNodes = 1
			effectiveSlotsPerNode = 1
			haveSlotsPerNode = false
		}

		resources.SetInstances(map[string]int32{"nodes": int32(numNodes)})

		if haveSlotsPerNode {
			resources.SetCores(map[string]float32{
				"per-node":     float32(effectiveSlotsPerNode),
				"per-instance": float32(effectiveSlotsPerNode),
			})
		} else {
			resources.SetCores(map[string]float32{"per-node": float32(effectiveSlotsPerNode)})
		}
	case gresSupported && (tresSupported || (isPbsLauncher && !haveSlotsPerNode)):
		/*
		 * We can tell the Workload Manager how many total GPUs we need
		 * and that we'd like 1 task per node and the workload manager
		 * will automatically allocate the nodes, such that the sum of
		 * the GPUs on each node equals the total GPUs requested.
		 */
		resources.SetInstances(map[string]int32{"per-node": 1})

		if haveSlotsPerNode {
			resources.SetGpus(map[string]int32{
				"total":        int32(numSlots),
				"per-instance": int32(effectiveSlotsPerNode),
			})
		} else {
			resources.SetGpus(map[string]int32{"total": int32(numSlots)})
		}
	case gresSupported:
		resources.SetInstances(map[string]int32{"nodes": int32(numNodes)})
		resources.SetGpus(map[string]int32{"per-node": int32(effectiveSlotsPerNode)})
	default:
		// GPUs requested, but neither TRES nor GRES supported.
		resources.SetInstances(map[string]int32{"nodes": int32(numNodes)})
	}

	// Commands, Shells, and Notebooks must run on a single node.
	t.restrictCommandsShellsAndNotebooksToSingleNode(resources)

	return resources
}

// Restricts commands, shells, and notebooks to a single node.
//
// Commands, shells, and notebooks are intended to only run on a single
// node.  However, if each compute node has 4 GPUs and, 8 GPUS are
// specified in the configuration, as shown in the example below, then
// more than one node will be allocated, which is not desirable.
//
// det notebook start --config resources.slots=8
//
// Therefore, explicitly request a single node.
//
// If more slots are requested than are available on any compute node
// on the cluster, the behavior of the Workload Manager will be as
// follows:
//
// For Slurm, the job will fail with the following error:
//
//	sbatch: error: Batch job submission failed: Requested node
//	               configuration is not available
//
// For PBS, the job will remain queued with the following reason:
//
//	Not Running: Insufficient amount of resource: ngpus
func (t *TaskSpec) restrictCommandsShellsAndNotebooksToSingleNode(
	resources *launcher.ResourceRequirements,
) {
	switch t.TaskType {
	case model.TaskTypeCommand, model.TaskTypeShell, model.TaskTypeNotebook:
		resources.SetInstances(map[string]int32{
			"nodes": 1,
		})
	}
}

// slotsPerNode returns the number of slots per node specified in the
// configuration (if any), else a value indicating that nothing was specified.
func (t *TaskSpec) slotsPerNode(isPbsLauncher bool) int {
	switch {
	case isPbsLauncher && t.PbsConfig.SlotsPerNode() != nil:
		return *t.PbsConfig.SlotsPerNode()
	case !isPbsLauncher && t.SlurmConfig.SlotsPerNode() != nil:
		return *t.SlurmConfig.SlotsPerNode()
	default:
		return unspecifiedSlotsPerNode
	}
}

// getPortMappings returns all PodMan mappings specified in environment.ports.
func getPortMappings(t *TaskSpec) *[]string {
	var portMappings []string
	if len(t.Environment.Ports()) > 0 {
		for k, v := range t.Environment.Ports() {
			if strings.HasPrefix(strings.ToLower(k), "podman") {
				portMappings = append(portMappings, strconv.Itoa(v))
			}
		}
	}
	return &portMappings
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
	syslog *logrus.Entry,
	allocationID string,
	slotType device.Type, workDir string,
	slurmPartition string, containerRunType string,
	launchingUser string,
) *map[string]string {
	launchConfig := map[string]string{
		"workingDir":          workDir,
		"enableWritableTmpFs": trueValue,
		// Pass along all variables (PBS) otherwise we only inherit a
		// minimal PATH from PBS that is missing /usr/sbin etc.
		"exportAll": "true",
	}
	if slurmPartition != "" {
		// Use queue config as both Slurm/PBS support it
		launchConfig["queue"] = slurmPartition
	}
	if slotType == device.CUDA {
		launchConfig["enableNvidia"] = trueValue
	}
	if slotType == device.ROCM {
		launchConfig["enableROCM"] = trueValue
	}
	if containerRunType == podman {
		launchConfig["networkMode"] = "host"
	}
	if t.SlurmConfig.GpuType() != nil {
		launchConfig["gpuType"] = *t.SlurmConfig.GpuType()
	}
	// From launcher 3.0.16, disableImageCache & add/dropCapabilities are supported, but
	// implemented for podman only. Added to singularity as well for 3.1.4.
	if t.Environment.ForcePullImage() {
		launchConfig["disableImageCache"] = trueValue
	}
	if len(t.Environment.AddCapabilities()) > 0 {
		launchConfig["addCapabilities"] = strings.Join(t.Environment.AddCapabilities(), ",")
	}
	if len(t.Environment.DropCapabilities()) > 0 {
		launchConfig["dropCapabilities"] = strings.Join(t.Environment.DropCapabilities(), ",")
	}
	if containerRunType == podman && t.Environment.RegistryAuth() != nil {
		syslog.WithField("allocation-id", allocationID).
			Warningf("NOT SUPPORTED: podman && environment.registry_auth -- use podman login")
	}
	// Launcher 3.0.17 added support for devices. This is specific to docker/podman carriers.
	if len(t.ResourcesConfig.Devices()) > 0 {
		elements := []string{}
		for _, d := range t.ResourcesConfig.Devices() {
			deviceString := fmt.Sprintf("%s:%s:%s", d.RawHostPath, d.RawContainerPath, *d.RawMode)
			elements = append(elements, deviceString)
		}
		launchConfig["devices"] = strings.Join(elements, ",")
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
	syslog *logrus.Entry,
	allocationID string,
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
			syslog.WithField("allocation-id", allocationID).Error("Failure to create TarGz Archive", err)
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
	syslog *logrus.Entry,
	allocationID string,
	taskSpec *TaskSpec,
	masterHost string,
	masterPort int,
	certificateName string,
	tmpMount bool,
	slotType device.Type,
	containerRunType string,
	localTmp string,
	slotsPerNode int,
) (map[string]string, error) {
	// Hash map containing the environment variables.
	m := make(map[string]string)

	// These represent the environment variables that are set by Determined AI.
	for k, v := range taskSpec.EnvVars() {
		m[k] = v
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
	//     - EMPTY
	//
	// Or
	//
	// environment:
	//   image: "environment:cuda-11.2-tf-2.5-gpu-0.17.7.sif"
	//   environment_variables:
	//     cuda:
	//       - DETECTRON2_DATASETS=/mnt/dtrain-fsx/detectron2
	//       - MY_ENV_VAR1=abc
	//       - MY_ENV_VAR2=xyz
	//       - EMPTY

	envVars := taskSpec.Environment.EnvironmentVariables().For(slotType)

	// Add each user-defined environment variable to the map.
	for _, s := range envVars {
		tokens := strings.Split(s, "=")

		if len(tokens) > 1 {
			m[tokens[0]] = tokens[1]
		} else {
			m[tokens[0]] = ""
		}
	}

	// These environment variables are required in "harness/determined/_info.py". If
	// they are not set, then task container will fail.
	m["DET_MASTER"] = fmt.Sprintf("%s:%d", masterHost, masterPort)
	m["DET_MASTER_HOST"] = masterHost
	m["DET_MASTER_IP"] = masterHost
	m["DET_MASTER_PORT"] = fmt.Sprintf("%d", masterPort)
	m["DET_CLUSTER_ID"] = taskSpec.ClusterID
	// On non-zero exit of any component/step of the sbatch job, terminate with an error
	m["SLURM_KILL_BAD_EXIT"] = "1"

	// If not provided by the user, set default MPI to pmi2
	if _, ok := m["SLURM_MPI_TYPE"]; !ok {
		m["SLURM_MPI_TYPE"] = "pmi2"
	}

	// Some in-container setup in slurm needs to know the slot type to set other envvars correctly.
	m["DET_SLOT_TYPE"] = string(slotType)
	// If slots_per_node is specified, generate a DET_SLOT_IDS value to enable use of the slots
	if slotsPerNode != unspecifiedSlotsPerNode {
		m["DET_SLOT_IDS"] = generatesSlotIdsString(slotsPerNode)
	}

	// The "master/internal/resourcemanagers/kubernetes/spec.go" checks if the
	// certificate name is set before assigning it to an environment variable, so
	// we're duplicating that same behavior here.
	if certificateName != "" {
		m["DET_MASTER_CERT_NAME"] = certificateName
	}

	// Identify a container-private local directory
	m["DET_LOCALTMP"] = localTmp

	// If the user has not configured a bind mount of /tmp trigger
	// dispatcher-wrapper.sh to make it local to the container.
	// This isn't needed with enroot since it is always local.
	if !tmpMount && containerRunType != enroot {
		m["DET_CONTAINER_LOCAL_TMP"] = "1"
	}

	if containerRunType == enroot {
		// By default mount the user's home dir
		m["ENROOT_MOUNT_HOME"] = "y"
	}

	if taskSpec.Environment.RegistryAuth() != nil {
		m["SINGULARITY_DOCKER_USERNAME"] = taskSpec.Environment.RegistryAuth().Username
		m["SINGULARITY_DOCKER_PASSWORD"] = taskSpec.Environment.RegistryAuth().Password
		m["APPTAINER_DOCKER_USERNAME"] = taskSpec.Environment.RegistryAuth().Username
		m["APPTAINER_DOCKER_PASSWORD"] = taskSpec.Environment.RegistryAuth().Password
		if len(taskSpec.Environment.RegistryAuth().ServerAddress) > 0 {
			syslog.WithField("allocation-id", allocationID).Warningf(
				"NOT SUPPORTED: environment.registry_auth.serveraddress: %s ",
				taskSpec.Environment.RegistryAuth().ServerAddress)
		}
		if len(taskSpec.Environment.RegistryAuth().Email) > 0 {
			syslog.WithField("allocation-id", allocationID).Warningf(
				"NOT SUPPORTED: environment.registry_auth.email: %s ",
				taskSpec.Environment.RegistryAuth().Email)
		}
	}

	return m, nil
}

// Return a DET_SLOT_IDS value of the form [0,1,2...] referencing
// the number of slots specified.
func generatesSlotIdsString(slots int) string {
	var slotIds []string
	for i := 0; i < slots; i++ {
		slotIds = append(slotIds, strconv.Itoa(i))
	}

	return fmt.Sprintf("[%s]", strings.Join(slotIds, ","))
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
func getDataVolumes(mounts []mount.Mount) ([]launcher.Data, bool, bool, error) {
	volumes := []launcher.Data{}
	userWantsDirMountedOnTmp := false
	varTmpExists := false
	var err error

	for i, mount := range mounts {
		if strings.HasPrefix(mount.Target, RunDir) {
			err = fmt.Errorf("bind_mounts.container_path: %s not supported."+
				"HPC launcher cannot mount under %s", mount.Target, RunDir)
			return volumes, userWantsDirMountedOnTmp, varTmpExists, err
		}

		volume := *launcher.NewData()
		volume.SetName("ds" + strconv.Itoa(i))
		volume.SetSource(mount.Source)
		volume.SetTarget(mount.Target)
		volume.SetReadOnly(mount.ReadOnly)
		volumes = append(volumes, volume)
		if mount.Target == tmp {
			userWantsDirMountedOnTmp = true
		}
		// Check if the user has already provided a binding for "/var/tmp" folder in the yaml file
		// for the experiment and set value for varTmpExists accordingly.
		if mount.Target == varTmp {
			varTmpExists = true
		}
	}

	return volumes, userWantsDirMountedOnTmp, varTmpExists, err
}

// Used for creating a tmpfs mount type at the target location.
func addTmpFs(volumes []launcher.Data, name string, target string) []launcher.Data {
	volume := *launcher.NewData()
	volume.SetName(name)
	volume.SetSource("tmpfs")

	/*
	 * Set target and add a mount option to enable target directory creation,
	 * if it did not exist
	 */
	volume.SetTarget(target + ":x-create=dir")
	volumes = append(volumes, volume)
	return volumes
}

// Create a softlink archive entry for the specified file name in the
// '/run/determined' directory to the local container temp version.
// Provide a localTmp directory to redirect it elsewhere (must end in /).
func getRunSubdirLink(aug *model.AgentUserGroup, name string, localTmp string) archive.Item {
	return aug.OwnedArchiveItem(RunDir+"/"+name,
		[]byte(localTmp+containerTmpDeterminedDir+name), 0o700, tar.TypeSymlink)
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
					// If not a file, then extract the top-level directory name
					if filepath.Base(contained) != contained {
						dir, _ := filepath.Split(contained)
						contained = filepath.Dir(dir)
					}
					// links are only created for top-level directories under /run/determined
					// If this is a file in a subdir, it will use the parent dir link
					if !strings.Contains(contained, "/") {
						linksSet[contained] = true
					}
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
// The links point at the {localTmp}/run/determined container-private directory
// so each rank can have a different link.
func dispatcherArchive(
	syslog *logrus.Entry,
	allocationID string,
	aug *model.AgentUserGroup,
	linksNeeded []string,
	localTmp string,
) cproto.RunArchive {
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
		dispatherArchive = append(dispatherArchive, getRunSubdirLink(aug, linkName, localTmp))
		syslog.WithField("allocation-id", allocationID).Tracef("Created link for %s", linkName)
	}

	return wrapArchive(dispatherArchive, "/")
}
