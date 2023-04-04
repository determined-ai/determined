package dispatcherrm

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
)

const hpcResourceDetailsRefreshPeriod = time.Minute

var errHPCDetailsCacheEmpty = errors.New("HPC resource details cache is empty")

// hpcResources is a data type describing the HPC resources available
// to Slurm on the Launcher node.
// Example output of the HPC resource details from the Launcher.
// ---
// partitions:
// - totalAvailableNodes: 293
// totalAllocatedNodes: 21
// partitionName: workq
// totalAvailableGpuSlots: 16
// totalNodes: 314
// totalGpuSlots: 16
// - totalAvailableNodes: 293
// ...more partitions.
type hpcResources struct {
	Partitions                  []hpcPartitionDetails `json:"partitions,flow"` //nolint:staticcheck
	Nodes                       []hpcNodeDetails      `json:"nodes,flow"`      //nolint:staticcheck
	DefaultComputePoolPartition string                `json:"defaultComputePoolPartition"`
	DefaultAuxPoolPartition     string                `json:"defaultAuxPoolPartition"`
}

// hpcPartitionDetails holds HPC Slurm partition details.
type hpcPartitionDetails struct {
	TotalAvailableNodes    int    `json:"totalAvailableNodes"`
	PartitionName          string `json:"partitionName"`
	IsDefault              bool   `json:"default"`
	TotalAllocatedNodes    int    `json:"totalAllocatedNodes"`
	TotalAvailableGpuSlots int    `json:"totalAvailableGpuSlots"`
	TotalNodes             int    `json:"totalNodes"`
	TotalGpuSlots          int    `json:"totalGpuSlots"`
	TotalAvailableCPUSlots int    `json:"totalAvailableCpuSlots"`
	TotalCPUSlots          int    `json:"totalCpuSlots"`
	Accelerator            string `json:"accelerator"`
}

// hpcNodeDetails holds HPC Slurm node details.
type hpcNodeDetails struct {
	Partitions    []string `json:"partitions"`
	Addresses     []string `json:"addresses"`
	Draining      bool     `json:"draining"`
	Allocated     bool     `json:"allocated"`
	Name          string   `json:"name"`
	GpuCount      int      `json:"gpuCount"`
	GpuInUseCount int      `json:"gpuInUseCount"`
	CPUCount      int      `json:"cpuCount"`
	CPUInUseCount int      `json:"cpuInUseCount"`
}

// hpcResourceDetailsCache stores details of the HPC resource information cache.
type hpcResourceDetailsCache struct {
	rmConfig *config.DispatcherResourceManagerConfig // TODO: Refactor to not use entire rm conf.
	log      *logrus.Entry
	cl       *launcherAPIClient

	lastSample atomic.Pointer[hpcResources]
	sampled    <-chan struct{}
}

func newHpcResourceDetailsCache(
	rmConfig *config.DispatcherResourceManagerConfig,
	cl *launcherAPIClient,
) *hpcResourceDetailsCache {
	sampled := make(chan struct{})

	c := &hpcResourceDetailsCache{
		rmConfig: rmConfig,
		log:      logrus.WithField("component", "hpc-resource-details-cache"),
		cl:       cl,
		sampled:  sampled,
	}

	go c.periodicallyUpdate(sampled)

	return c
}

func (c *hpcResourceDetailsCache) periodicallyUpdate(sampled chan<- struct{}) {
	for {
		res, ok := c.fetchHpcResourceDetails()
		if !ok {
			time.Sleep(hpcResourceDetailsRefreshPeriod)
			continue
		}

		if c.lastSample.Load() == nil {
			c.lastSample.Store(res)
			close(sampled)
		} else {
			c.lastSample.Store(res)
		}
		time.Sleep(hpcResourceDetailsRefreshPeriod)
	}
}

// load loads the last sample of HPC resource details. Returns error if the cache is empty.
func (c *hpcResourceDetailsCache) load() (*hpcResources, error) {
	res := c.lastSample.Load()
	if res == nil {
		return nil, errHPCDetailsCacheEmpty
	}
	return res, nil
}

// wait was for the cache to be populated with the first sample.
func (c *hpcResourceDetailsCache) wait() {
	<-c.sampled
}

// fetchHpcResourceDetails retrieves the details about HPC Resources.
// This function uses HPC Resources manifest to retrieve the required details.
// This function performs the following steps:
//  1. Launch the manifest.
//  2. Read the log file with details on HPC resources.
//  3. Parse and load the details into a predefined struct - HpcResourceDetails
//  4. Terminate the manifest.
//
// Returns struct with HPC resource details - HpcResourceDetails.
// This function also queries launcher version and warns user if minimum required
// launcher version is not met.
func (c *hpcResourceDetailsCache) fetchHpcResourceDetails() (
	*hpcResources, bool,
) {
	dispatchInfo, resp, err := c.cl.launchHPCResourcesJob() //nolint:bodyclose
	if err != nil {
		c.cl.handleServiceQueryError(resp, err)
		return nil, false
	}
	dispatchID := dispatchInfo.GetDispatchId()
	owner := "launcher"
	c.log.Debugf("Launched Manifest with DispatchID %s", dispatchID)
	defer func() {
		_, _, err := c.cl.terminateDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			c.log.Error(err)
			return
		}

		_, err = c.cl.deleteDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			c.log.Error(err)
			return
		}
	}()

	logFileName := "slurm-resources-info"
	// HPC resource details will be listed in a log file with name
	// 'slurm-resources-info' in YAML format. Use LoadEnvironmentLog()
	// method to retrieve the log file.
	//
	// Because we're using "launch()" instead of "launchAsync()" to get
	// the HPC resources, we can expect that the "slurm-resources-info" log
	// file containing the SLURM partition info will be available, because
	// "launch()" will not return until the "slurm-resources-info" file is
	// written. Had we used "launchAsync()", we would have to poll the launcher
	// for job completion, but that's tricky, because the monitoring API will
	// go through the SlurmCarrier on the launcher side, which expects a job ID.
	// The SlurmCarrier will hang for a while waiting for the SLURM job ID to be
	// written, which it never will, because SlurmResources only queries SLURM
	// to get the partition info and does not create a job, so no job ID is ever
	// generated.  Eventually it will timeout waiting and return, but that's too
	// long of a delay for us to deal with.
	log, _, err := c.cl.loadEnvironmentLog(owner, dispatchID, logFileName) //nolint:bodyclose
	if err != nil {
		c.log.Error(err)
		return nil, false
	}

	// Parse the HPC resources file and extract the details into a
	// HpcResourceDetails object using YAML package.
	resourcesBytes, err := io.ReadAll(log)
	if err != nil {
		c.log.WithError(err).Errorf("failed to read HPC resources environment log file")
		return nil, false
	}

	var newSample hpcResources
	if err = yaml.Unmarshal(resourcesBytes, &newSample); err != nil {
		c.log.WithError(err).Errorf("failed to parse HPC Resource details")
		return nil, false
	}

	computePool, auxPool := selectDefaultPools(
		newSample.Partitions,
		c.rmConfig.DefaultComputeResourcePool,
		c.rmConfig.DefaultAuxResourcePool,
	)
	newSample.DefaultComputePoolPartition = computePool
	newSample.DefaultAuxPoolPartition = auxPool

	c.hpcResourcesToDebugLog(newSample)
	return &newSample, true
}

// selectDefaultPools identifies partitions suitable as default compute and default
// aux partitions (if possible).
func selectDefaultPools(
	hpcResourceDetails []hpcPartitionDetails,
	defaultComputePool *string,
	defaultAuxPool *string,
) (
	string, string,
) {
	// The default compute pool is the default partition if it has any GPUS,
	// otherwise select any partition with GPUs.
	// The AUX partition, use the default partition if available, otherwise any partition.

	defaultComputePar := "" // Selected default Compute/GPU partition
	defaultAuxPar := ""     // Selected default Aux partition

	fallbackComputePar := "" // Fallback Compute/GPU partition (has GPUs)
	fallbackAuxPar := ""     // Fallback partition if no default

	for _, v := range hpcResourceDetails {
		if v.IsDefault {
			defaultAuxPar = v.PartitionName
			if v.TotalGpuSlots > 0 {
				defaultComputePar = v.PartitionName
			}
		} else {
			fallbackAuxPar = v.PartitionName
			if v.TotalGpuSlots > 0 {
				fallbackComputePar = v.PartitionName
			}
		}
	}

	// Ensure we have a default aux, even if no partitions marked as such
	if defaultAuxPar == "" {
		defaultAuxPar = fallbackAuxPar
	}

	// If no default compute/GPU partitions, use a fallback partition
	if defaultComputePar == "" {
		if fallbackComputePar != "" {
			defaultComputePar = fallbackComputePar
		} else {
			defaultComputePar = defaultAuxPar
		}
	}

	// If explicitly configured, just override.
	if defaultComputePool != nil {
		defaultComputePar = *defaultComputePool
	}
	if defaultAuxPool != nil {
		defaultAuxPar = *defaultAuxPool
	}

	return defaultComputePar, defaultAuxPar
}

// hpcResourcesToDebugLog puts a summary of the available HPC resources to the debug log.
func (c *hpcResourceDetailsCache) hpcResourcesToDebugLog(resources hpcResources) {
	if c.log.Logger.Level != logrus.DebugLevel {
		return
	}

	c.log.Debugf(
		"default resource pools are '%s', '%s'",
		resources.DefaultComputePoolPartition,
		resources.DefaultAuxPoolPartition,
	)
	c.log.Debugf("HPC Resource details: %+v", resources.Partitions)
	nodesWithGpu := 0
	gpusFound := 0
	nodesAllocated := 0
	gpusAllocated := 0
	cpusFound := 0
	cpusAllocated := 0
	for _, node := range resources.Nodes {
		gpusFound += node.GpuCount
		cpusFound += node.CPUCount
		if node.GpuCount > 0 {
			nodesWithGpu++
		}
		if node.Allocated {
			nodesAllocated++
		}
		gpusAllocated += node.GpuInUseCount
		cpusAllocated += node.CPUInUseCount
	}
	c.log.
		WithField("nodes", len(resources.Nodes)).
		WithField("allocated", nodesAllocated).
		WithField("nodes with GPU", nodesWithGpu).
		WithField("GPUs", gpusFound).
		WithField("GPUs allocated", gpusAllocated).
		WithField("CPUs", cpusFound).
		WithField("CPUs allocated", cpusAllocated).
		Debug("Node summary")
}
