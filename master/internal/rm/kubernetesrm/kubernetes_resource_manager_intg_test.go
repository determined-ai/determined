//go:build integration
// +build integration

package kubernetesrm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	k8error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	auxNode1Name        = "aux"
	auxNode2Name        = "aux2"
	compNode1Name       = "comp"
	compNode2Name       = "comp2"
	compLabel           = "compLabel"
	pod1NumSlots        = 4
	pod2NumSlots        = 8
	nodeNumSlots        = int64(8)
	nodeNumSlotsCPU     = int64(20)
	slotTypeGPU         = "randomDefault"
	cpuResourceRequests = int64(4000)
	nonDetNodeName      = "NonDetermined"
)

func TestMain(m *testing.M) {
	// Need to set up the DB for TestJobQueueStats
	pgDB, _, err := db.ResolveNewPostgresDatabase()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	setClusterID(uuid.NewString())

	os.Exit(m.Run())
}

func TestGetAgents(t *testing.T) {
	type AgentsTestCase struct {
		Name           string
		jobsService    *jobsService
		wantedAgentIDs map[string]int
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	agentsTests := []AgentsTestCase{
		{
			Name: "GetAgents-CPU-NoPodLabels-NoAgents",
			jobsService: createMockJobsService(make(map[string]*k8sV1.Node),
				device.CPU,
				false,
			),
			wantedAgentIDs: make(map[string]int),
		},
		{
			Name: "GetAgents-CPU-NoPodLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			wantedAgentIDs: map[string]int{auxNode1Name: 0, auxNode2Name: 0},
		},
		{
			Name: "GetAgents-CPU-PodLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			wantedAgentIDs: map[string]int{auxNode1Name: 0, auxNode2Name: 0, nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-GPU-PodLabels-NonDetAgent",
			jobsService: createMockJobsService(make(map[string]*k8sV1.Node),
				slotTypeGPU,
				true,
			),
			wantedAgentIDs: map[string]int{nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-GPU-NoPodNoLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				false,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0},
		},
		{
			Name: "GetAgents-GPU-PodLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0, nonDetNodeName: 0},
		},
		{
			Name: "GetAgents-CUDA-NoPodLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0},
		},
		{
			Name: "GetAgents-CUDA-PodLabels",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			wantedAgentIDs: map[string]int{compNode1Name: 0, compNode2Name: 0, nonDetNodeName: 0},
		},
	}

	for _, test := range agentsTests {
		t.Run(test.Name, func(t *testing.T) {
			agentsResp, err := test.jobsService.getAgents()
			require.NoError(t, err)
			require.Equal(t, len(test.wantedAgentIDs), len(agentsResp.Agents))
			for _, agent := range agentsResp.Agents {
				_, ok := test.wantedAgentIDs[agent.Id]
				require.True(t, ok,
					fmt.Sprintf("name %s is not present in agent id list", agent.Id))
			}
		})
	}
}

// TestGetAgentsNodeSelectors adds node selectors to nodes belonging to the jobs service
// and checks if they belong to the job service's resource pool.
func TestGetAgentsNodeSelectors(t *testing.T) {
	cases := []struct {
		Name          string
		labels        map[string]string
		agentsMatched map[string]int
	}{
		{
			Name:   "GetAgents-GPU-Two-NodeSelectors",
			labels: map[string]string{compNode1Name: compNode1Name, compNode2Name: compNode1Name},
			agentsMatched: map[string]int{
				compNode1Name:  1,
				compNode2Name:  1,
				nonDetNodeName: 0,
			},
		},
		{
			Name:   "GetAgents-GPU-One-NodeSelectors",
			labels: map[string]string{compNode1Name: compNode1Name, compNode2Name: compNode2Name},
			agentsMatched: map[string]int{
				compNode1Name:  1,
				compNode2Name:  0,
				nonDetNodeName: 0,
			},
		},
		{
			Name:   "GetAgents-GPU-No-NodeSelectors",
			labels: map[string]string{},
			agentsMatched: map[string]int{
				compNode1Name:  0,
				compNode2Name:  0,
				nonDetNodeName: 0,
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			_, _, compNode1, compNode2 := setupNodes()

			if label, exists := test.labels[compNode1.Name]; exists {
				compNode1.SetLabels(map[string]string{compLabel: label})
			}

			if label, exists := test.labels[compNode2.Name]; exists {
				compNode2.SetLabels(map[string]string{compLabel: label})
			}

			js := createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			}, device.CUDA, true)

			// Add a node selector to the job service's config.
			js.resourcePoolConfigs = []config.ResourcePoolConfig{{
				PoolName: "test-pool",
				TaskContainerDefaults: &model.TaskContainerDefaultsConfig{GPUPodSpec: &k8sV1.Pod{
					Spec: k8sV1.PodSpec{NodeSelector: map[string]string{compLabel: compNode1Name}},
				}},
			}}

			agentsResp, err := js.getAgents()
			require.NoError(t, err)
			require.Equal(t, len(test.agentsMatched), len(agentsResp.Agents))

			for _, agent := range agentsResp.Agents {
				rp, ok := test.agentsMatched[agent.Id]
				require.True(t, ok,
					fmt.Sprintf("name %s is not present in agent id list", agent.Id))
				require.Len(t, agent.ResourcePools, rp)
			}
		})
	}
}

func TestGetAgent(t *testing.T) {
	type AgentTestCase struct {
		Name          string
		jobsService   *jobsService
		agentExists   bool
		wantedAgentID string
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	largeNode := &k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode1Name,
		},
		Status: k8sV1.NodeStatus{
			Allocatable: map[k8sV1.ResourceName]resource.Quantity{
				k8sV1.ResourceName(resourceTypeNvidia): *resource.NewQuantity(
					16,
					resource.DecimalSI,
				),
			},
		},
	}

	agentTests := []AgentTestCase{
		{
			Name: "GetAgent-CPU-NoPodLabels-Aux1",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			agentExists:   true,
			wantedAgentID: auxNode1Name,
		},
		{
			Name: "GetAgent-CPU-PodLabels-Aux2",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			agentExists:   true,
			wantedAgentID: auxNode2Name,
		},
		{
			Name: "GetAgent-GPU-PodLabels-Comp1",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentExists:   true,
			wantedAgentID: compNode1Name,
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-Comp2",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   true,
			wantedAgentID: compNode2Name,
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-NonexistentAgent",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   false,
			wantedAgentID: uuid.NewString(),
		},
		{
			Name: "GetAgent-CUDA-NoPodLabels-EmptyAgentID",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentExists:   false,
			wantedAgentID: "",
		},
		{
			Name: "GetAgent-CUDA-Large-Node",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: largeNode,
			}, slotTypeGPU, false),
			wantedAgentID: compNode1Name,
			agentExists:   false,
		},
	}

	for _, test := range agentTests {
		t.Run(test.Name, func(t *testing.T) {
			agentResp := test.jobsService.getAgent(test.wantedAgentID)
			if agentResp == nil {
				require.False(t, test.agentExists)
				return
			}
			require.Equal(t, test.wantedAgentID, agentResp.Agent.Id)

			// Check all filled slots come before an empty slot.
			var slotIDs []string
			for _, s := range agentResp.Agent.Slots {
				slotIDs = append(slotIDs, s.Id)
			}
			slices.Sort(slotIDs)
			seenEmptySlot := false
			for _, s := range slotIDs {
				if agentResp.Agent.Slots[s].Container != nil {
					require.False(t, seenEmptySlot, "all filled slots must come before an empty slot")
				} else {
					seenEmptySlot = true
				}
			}
		})
	}
}

func TestGetSlots(t *testing.T) {
	type SlotsTestCase struct {
		Name           string
		jobsService    *jobsService
		agentID        string
		agentExists    bool
		wantedSlotsNum int
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	slotsTests := []SlotsTestCase{
		{
			Name: "GetSlots-CPU-NoPodLabels-Aux1",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				false,
			),
			agentID:        auxNode1Name,
			agentExists:    true,
			wantedSlotsNum: int(nodeNumSlots),
		},
		{
			Name: "GetSlots-GPU-NoPodLabels-Comp2",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				false,
			),
			agentID:        compNode2Name,
			agentExists:    true,
			wantedSlotsNum: 8,
		},
		{
			Name: "GetSlots-CUDA-PodLabels-Comp1",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        compNode1Name,
			agentExists:    true,
			wantedSlotsNum: int(nodeNumSlots),
		},
		{
			Name: "GetSlots-CUDA-PodLabels-NonexistentAgent",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        uuid.NewString(),
			agentExists:    false,
			wantedSlotsNum: 0,
		},
		{
			Name: "GetSlots-CUDA-PodLabels-EmptyAgentID",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:        "",
			agentExists:    false,
			wantedSlotsNum: 0,
		},
	}

	// Number of active slots on given nodes.
	nodeToSlots := map[string]int{
		auxNode1Name:  pod1NumSlots,
		compNode1Name: pod2NumSlots,
		auxNode2Name:  0,
		compNode2Name: 0,
	}
	for _, test := range slotsTests {
		t.Run(test.Name, func(t *testing.T) {
			slotsResp := test.jobsService.getSlots(test.agentID)
			if slotsResp == nil {
				require.False(t, test.agentExists)
				return
			}
			require.Len(t, slotsResp.Slots, test.wantedSlotsNum)

			// Count number of active slots on the node. (Slots allocated to a pod running
			// a container).
			activeSlots := 0
			for _, slot := range slotsResp.Slots {
				slotID, err := strconv.Atoi(slot.Id)
				require.NoError(t, err)
				require.True(t, slotID >= 0 && slotID < int(nodeNumSlots),
					fmt.Sprintf("slot %s is out of range", slot.Id))
				if slot.Container != nil {
					activeSlots++
				}
			}
			require.Equal(t, nodeToSlots[test.agentID], activeSlots)
		})
	}
}

func TestGetSlot(t *testing.T) {
	type SlotTestCase struct {
		Name          string
		jobsService   *jobsService
		agentID       string
		wantedSlotNum string
	}

	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	slotTests := []SlotTestCase{
		{
			Name: "GetSlot-CPU-PodLabels-Aux1-LastId",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				auxNode1Name: auxNode1,
				auxNode2Name: auxNode2,
			},
				device.CPU,
				true,
			),
			agentID:       auxNode1Name,
			wantedSlotNum: strconv.Itoa(7),
		},
		{
			Name: "GetSlot-GPU-PodLabels-Comp1-Id4",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(4),
		},
		{
			Name: "GetSlot-GPU-PodLabels-Comp1-Id4",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentID:       compNode1Name,
			wantedSlotNum: "004",
		},
		{
			Name: "GetSlot-GPU-PodLabels-Comp1-Id0",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				slotTypeGPU,
				true,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(0),
		},
		{
			Name: "GetSlot-CUDA-NoPodLabels-Comp1-BadSlotReq",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				false,
			),
			agentID:       compNode1Name,
			wantedSlotNum: strconv.Itoa(-1),
		},
		{
			Name: "GetSlot-CUDA-PodLabels-Comp2-BadSlotReq",
			jobsService: createMockJobsService(map[string]*k8sV1.Node{
				compNode1Name: compNode1,
				compNode2Name: compNode2,
			},
				device.CUDA,
				true,
			),
			agentID:       compNode2Name,
			wantedSlotNum: strconv.Itoa(0),
		},
	}

	for _, test := range slotTests {
		t.Run(test.Name, func(t *testing.T) {
			wantedSlotInt, err := strconv.Atoi(test.wantedSlotNum)
			require.NoError(t, err)

			slotResp := test.jobsService.getSlot(test.agentID, test.wantedSlotNum)
			if slotResp == nil {
				require.True(t, wantedSlotInt < 0 || wantedSlotInt >= int(nodeNumSlots))
				return
			}

			actualSlotID, err := strconv.Atoi(slotResp.Slot.Id)
			require.NoError(t, err)
			require.Equal(t, wantedSlotInt, actualSlotID)
		})
	}
}

func TestAssignResourcesTime(t *testing.T) {
	taskList := tasklist.New()
	groups := make(map[model.JobID]*tasklist.Group)
	allocateReq := sproto.AllocateRequest{
		JobID:             model.JobID("test-job"),
		JobSubmissionTime: time.Now(),
		SlotsNeeded:       0,
	}
	groups[allocateReq.JobID] = &tasklist.Group{
		JobID: allocateReq.JobID,
	}
	mockPods := createMockJobsService(make(map[string]*k8sV1.Node), device.CUDA, true)
	poolRef := &kubernetesResourcePool{
		poolConfig:                &config.ResourcePoolConfig{PoolName: "cpu-pool"},
		jobsService:               mockPods,
		reqList:                   taskList,
		groups:                    groups,
		jobIDToAllocationID:       map[model.JobID]model.AllocationID{},
		allocationIDToJobID:       map[model.AllocationID]model.JobID{},
		slotsUsedPerGroup:         map[*tasklist.Group]int{},
		allocationIDToRunningPods: map[model.AllocationID]int{},
		syslog:                    logrus.WithField("component", "k8s-rp"),
	}

	poolRef.assignResources(&allocateReq)
	resourcesAllocated := poolRef.reqList.Allocation(allocateReq.AllocationID)
	require.NotNil(t, resourcesAllocated)
	require.False(t, resourcesAllocated.JobSubmissionTime.IsZero())
}

func TestGetResourcePools(t *testing.T) {
	expectedName := "testname"
	expectedMetadata := map[string]string{"x": "y*y"}
	cfg := &config.ResourceConfig{
		RootManagerInternal: &config.ResourceManagerConfig{
			KubernetesRM: &config.KubernetesResourceManagerConfig{
				ClusterName:                expectedName,
				Metadata:                   expectedMetadata,
				MaxSlotsPerPod:             ptrs.Ptr(5),
				DefaultAuxResourcePool:     "cpu-pool",
				DefaultComputeResourcePool: "gpu-pool",
			},
		},
		RootPoolsInternal: []config.ResourcePoolConfig{
			{PoolName: "cpu-pool"},
			{PoolName: "gpu-pool"},
		},
	}

	mockPods := createMockJobsService(make(map[string]*k8sV1.Node), device.CUDA, true)
	cpuPoolRef := &kubernetesResourcePool{
		poolConfig:  &config.ResourcePoolConfig{PoolName: "cpu-pool"},
		jobsService: mockPods,
		reqList:     tasklist.New(),
	}
	gpuPoolRef := &kubernetesResourcePool{
		poolConfig:  &config.ResourcePoolConfig{PoolName: "gpu-pool"},
		jobsService: mockPods,
		reqList:     tasklist.New(),
	}
	kubernetesRM := &ResourceManager{
		config:      cfg.ResourceManagers()[0].ResourceManager.KubernetesRM,
		poolsConfig: cfg.ResourceManagers()[0].ResourcePools,
		taskContainerDefaults: &model.TaskContainerDefaultsConfig{
			Kubernetes: &model.KubernetesTaskContainerDefaults{
				MaxSlotsPerPod: ptrs.Ptr(5),
			},
		},
		pools: map[string]*kubernetesResourcePool{
			"cpu-pool": cpuPoolRef,
			"gpu-pool": gpuPoolRef,
		},
	}

	resp, err := kubernetesRM.GetResourcePools()
	require.NoError(t, err)
	actual, err := json.MarshalIndent(resp.ResourcePools, "", "  ")
	require.NoError(t, err)

	expectedPools := []*resourcepoolv1.ResourcePool{
		{
			Name:                    "cpu-pool",
			Type:                    resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S,
			AuxContainerCapacity:    1,
			SlotsPerAgent:           5,
			DefaultAuxPool:          true,
			SchedulerType:           resourcepoolv1.SchedulerType_SCHEDULER_TYPE_KUBERNETES,
			SchedulerFittingPolicy:  resourcepoolv1.FittingPolicy_FITTING_POLICY_KUBERNETES,
			Location:                "n/a",
			InstanceType:            "n/a",
			Details:                 &resourcepoolv1.ResourcePoolDetail{},
			Stats:                   &jobv1.QueueStats{},
			ClusterName:             expectedName,
			ResourceManagerMetadata: expectedMetadata,
		},
		{
			Name:                    "gpu-pool",
			Type:                    resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S,
			SlotsPerAgent:           5,
			AuxContainerCapacity:    1,
			DefaultComputePool:      true,
			SchedulerType:           resourcepoolv1.SchedulerType_SCHEDULER_TYPE_KUBERNETES,
			SchedulerFittingPolicy:  resourcepoolv1.FittingPolicy_FITTING_POLICY_KUBERNETES,
			Location:                "n/a",
			InstanceType:            "n/a",
			Details:                 &resourcepoolv1.ResourcePoolDetail{},
			Stats:                   &jobv1.QueueStats{},
			ClusterName:             expectedName,
			ResourceManagerMetadata: expectedMetadata,
		},
	}
	expected, err := json.MarshalIndent(expectedPools, "", "  ")
	require.NoError(t, err)

	require.Equal(t, string(expected), string(actual))
}

func TestGetJobQueueStatsRequest(t *testing.T) {
	mockPods := createMockJobsService(make(map[string]*k8sV1.Node), device.CUDA, true)
	pool1 := &kubernetesResourcePool{
		poolConfig:  &config.ResourcePoolConfig{PoolName: "pool1"},
		jobsService: mockPods,
		reqList:     tasklist.New(),
	}
	pool2 := &kubernetesResourcePool{
		poolConfig:  &config.ResourcePoolConfig{PoolName: "pool2"},
		jobsService: mockPods,
		reqList:     tasklist.New(),
	}
	k8sRM := &ResourceManager{
		pools: map[string]*kubernetesResourcePool{
			"pool1": pool1,
			"pool2": pool2,
		},
	}

	cases := []struct {
		name        string
		filteredRPs []string
		expected    int
	}{
		{"empty, return all", []string{}, 2},
		{"filter 1 in", []string{"pool1"}, 1},
		{"filter 2 in", []string{"pool1", "pool2"}, 2},
		{"filter undefined in, return none", []string{"bogus"}, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := k8sRM.GetJobQueueStatsRequest(&apiv1.GetJobQueueStatsRequest{ResourcePools: tt.filteredRPs})
			require.NoError(t, err)
			require.Len(t, res.Results, tt.expected)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	mockPodInterface := &mocks.PodInterface{}
	kubernetesRM := &ResourceManager{
		config: &config.KubernetesResourceManagerConfig{
			ClusterName: "testname",
		},
		jobsService: &jobsService{
			podInterfaces: map[string]typedV1.PodInterface{
				"namespace": mockPodInterface,
			},
			syslog: logrus.WithField("namespace", "test"),
		},
	}

	t.Run("healthy", func(t *testing.T) {
		mockPodInterface.On("List", mock.Anything, mock.Anything).Return(nil, nil).Once()
		require.Equal(t, []model.ResourceManagerHealth{
			{
				ClusterName: "testname",
				Status:      model.Healthy,
			},
		}, kubernetesRM.HealthCheck())
	})

	t.Run("unhealthy", func(t *testing.T) {
		mockPodInterface.On("List", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("error")).Once()
		require.Equal(t, []model.ResourceManagerHealth{
			{
				ClusterName: "testname",
				Status:      model.Unhealthy,
			},
		}, kubernetesRM.HealthCheck())
	})
}

func TestVerifyNamespaceExists(t *testing.T) {
	js := createMockJobsService(nil, device.CPU, false)
	validNamespace := "validNamespace"
	nonexistentNamespaceName := "nonExistentNamespace"
	js.clientSet = setupClientSetForTests(validNamespace, &nonexistentNamespaceName)
	js.namespacesWithInformers[validNamespace] = true
	js.podInterfaces = make(map[string]typedV1.PodInterface)
	js.configMapInterfaces = make(map[string]typedV1.ConfigMapInterface)
	js.jobInterfaces = make(map[string]typedBatchV1.JobInterface)

	channel := make(chan resourcesRequestFailure, 16)
	js.requestQueueWorkers = []*requestProcessingWorker{
		{
			jobInterface:        js.jobInterfaces,
			podInterface:        js.podInterfaces,
			configMapInterfaces: js.configMapInterfaces,
			failures:            channel,
		},
	}

	// Valid namespace name.
	err := js.verifyNamespaceExists(validNamespace, true)
	require.NoError(t, err)

	invalidNamespace := "invalidNamespace"

	// Verify that the namespace was registered by all necessary components of the request
	// processing workers.
	for _, worker := range js.requestQueueWorkers {
		_, ok := worker.podInterface[validNamespace]
		_, notOk := worker.podInterface[invalidNamespace]
		require.True(t, ok)
		require.False(t, notOk)
		_, ok = worker.jobInterface[validNamespace]
		_, notOk = worker.jobInterface[invalidNamespace]
		require.True(t, ok)
		require.False(t, notOk)
		_, ok = worker.configMapInterfaces[validNamespace]
		_, notOk = worker.configMapInterfaces[invalidNamespace]
		require.True(t, ok)
		require.False(t, notOk)
	}

	// Test that a non-existent namespace name.
	err = js.verifyNamespaceExists(nonexistentNamespaceName, true)
	require.Error(t, err)
}

func TestCreateNamespaceHelper(t *testing.T) {
	js := createMockJobsService(nil, device.CPU, false)

	validNamespace := "valNamespace"
	existentNamespace := "nonexistentNamespace"
	erroneousNamespace := "erroneousNamespace"

	validk8sNamespace := &k8sV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:   validNamespace,
			Labels: map[string]string{determinedLabel: validNamespace},
		},
	}

	existentk8sNamespace := &k8sV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:   existentNamespace,
			Labels: map[string]string{determinedLabel: existentNamespace},
		},
	}

	erroneousk8sNamespace := &k8sV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:   erroneousNamespace,
			Labels: map[string]string{determinedLabel: erroneousNamespace},
		},
	}

	jobsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}
	namespaceInterface := &mocks.NamespaceInterface{}
	namespaceInterface.On("Create", context.TODO(), validk8sNamespace, metaV1.CreateOptions{}).
		Return(validk8sNamespace, nil).Once()
	namespaceInterface.On("Create", context.TODO(), existentk8sNamespace, metaV1.CreateOptions{}).
		Return(nil, &k8error.StatusError{ErrStatus: metaV1.Status{Reason: metaV1.StatusReasonAlreadyExists}}).Once()
	namespaceInterface.On("Create", context.TODO(), erroneousk8sNamespace, metaV1.CreateOptions{}).
		Return(nil, errors.New("random error")).Once()
	coreV1Interface.On("Namespaces").Return(namespaceInterface)
	configMapInterace := &mocks.ConfigMapInterface{}
	coreV1Interface.On("ConfigMaps", validNamespace).Return(configMapInterace)
	podsInterface := &mocks.PodInterface{}
	coreV1Interface.On("Pods", validNamespace).Return(podsInterface)
	jobsClientSet.On("CoreV1").Return(coreV1Interface)
	batchV1Interface := &mocks.K8sBatchV1Interface{}
	jobsInterface := &mocks.JobInterface{}
	batchV1Interface.On("Jobs", validNamespace).Return(jobsInterface)
	jobsClientSet.On("BatchV1").Return(batchV1Interface)
	js.clientSet = jobsClientSet

	js.podInterfaces = make(map[string]typedV1.PodInterface)
	js.configMapInterfaces = make(map[string]typedV1.ConfigMapInterface)
	js.jobInterfaces = make(map[string]typedBatchV1.JobInterface)
	channel := make(chan resourcesRequestFailure, 16)
	js.requestQueueWorkers = []*requestProcessingWorker{
		{
			jobInterface:        js.jobInterfaces,
			podInterface:        js.podInterfaces,
			configMapInterfaces: js.configMapInterfaces,
			failures:            channel,
		},
	}

	// test with valid namespace
	err := js.createNamespaceHelper(validNamespace)
	require.NoError(t, err)

	// test with non-existent namespace
	err = js.createNamespaceHelper(existentNamespace)
	require.NoError(t, err)

	// verify that all necessary components have resgistered the namespace
	_, ok := js.podInterfaces[validNamespace]
	require.True(t, ok)
	_, ok = js.podInterfaces[validNamespace]
	require.True(t, ok)
	_, ok = js.podInterfaces[validNamespace]
	require.True(t, ok)
	for _, worker := range js.requestQueueWorkers {
		_, ok := worker.podInterface[validNamespace]
		require.True(t, ok)
		_, ok = worker.jobInterface[validNamespace]
		require.True(t, ok)
		_, ok = worker.configMapInterfaces[validNamespace]
		require.True(t, ok)
	}

	// test with erroneous namespace
	err = js.createNamespace(erroneousNamespace, true)
	require.ErrorContains(t, err, "random error")
}

func TestDeleteNamespace(t *testing.T) {
	js := createMockJobsService(nil, device.CPU, false)

	validNamespace := "validNs"
	nonexistentNamespace := "nonExistentNs"
	erroneousNamespace := "erroneousNs"

	jobsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}
	namespaceInterface := &mocks.NamespaceInterface{}
	namespaceInterface.On("Delete", context.TODO(), validNamespace, metaV1.DeleteOptions{}).
		Return(nil).Once()
	namespaceInterface.On("Delete", context.TODO(), nonexistentNamespace, metaV1.DeleteOptions{}).
		Return(&k8error.StatusError{ErrStatus: metaV1.Status{Reason: metaV1.StatusReasonNotFound, Code: 404}}).Once()
	namespaceInterface.On("Delete", context.TODO(), erroneousNamespace, metaV1.DeleteOptions{}).
		Return(errors.New("random error")).Once()
	coreV1Interface.On("Namespaces").Return(namespaceInterface)
	jobsClientSet.On("CoreV1").Return(coreV1Interface)
	js.clientSet = jobsClientSet

	// test with valid namespace
	err := js.deleteNamespace(validNamespace)
	require.NoError(t, err)

	// test with non-existent namespace
	err = js.deleteNamespace(nonexistentNamespace)
	require.NoError(t, err)

	// test with erroneous namespace
	err = js.deleteNamespace(erroneousNamespace)
	require.ErrorContains(t, err, "random error")
}

func TestRemoveEmptyNamespace(t *testing.T) {
	ctx := context.Background()
	wkspID1, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	namespaceName := "anamespace"
	clusterName := "testing_C1"

	binding := model.WorkspaceNamespace{
		WorkspaceID: wkspID1,
		ClusterName: clusterName,
		Namespace:   namespaceName,
	}

	_, err := db.Bun().NewInsert().Model(&binding).Exec(ctx)
	require.NoError(t, err)

	js := createMockJobsService(nil, device.CPU, false)

	js.clientSet = setupClientSetForTests(namespaceName, nil)
	js.podInterfaces = map[string]typedV1.PodInterface{namespaceName: js.clientSet.CoreV1().Pods(namespaceName)}
	js.configMapInterfaces = map[string]typedV1.ConfigMapInterface{
		namespaceName: js.clientSet.CoreV1().ConfigMaps(namespaceName),
	}
	js.jobInterfaces = map[string]typedBatchV1.JobInterface{namespaceName: js.clientSet.BatchV1().Jobs(namespaceName)}

	channel := make(chan resourcesRequestFailure, 16)
	js.requestQueueWorkers = []*requestProcessingWorker{
		{
			jobInterface:        js.jobInterfaces,
			podInterface:        js.podInterfaces,
			configMapInterfaces: js.configMapInterfaces,
			failures:            channel,
		},
	}
	// Try removing non-empty namespace name.
	err = js.RemoveEmptyNamespace(namespaceName, clusterName)
	require.NoError(t, err)

	// Verify that the namespace was not removed from anywhere.
	_, ok := js.podInterfaces[namespaceName]
	require.True(t, ok)
	_, ok = js.jobInterfaces[namespaceName]
	require.True(t, ok)
	_, ok = js.configMapInterfaces[namespaceName]
	require.True(t, ok)
	for _, worker := range js.requestQueueWorkers {
		_, ok = worker.podInterface[namespaceName]
		require.True(t, ok)
		_, ok = worker.jobInterface[namespaceName]
		require.True(t, ok)
		_, ok = worker.configMapInterfaces[namespaceName]
		require.True(t, ok)
	}

	// Remove namespace from bindings.
	_, err = db.Bun().NewDelete().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID1).
		Exec(ctx)
	require.NoError(t, err)

	// Try removing empty namespace name.
	err = js.RemoveEmptyNamespace(namespaceName, clusterName)
	require.NoError(t, err)

	// Verify that the namespace was removed from everywhere.
	_, notOk := js.podInterfaces[namespaceName]
	require.False(t, notOk)
	_, notOk = js.jobInterfaces[namespaceName]
	require.False(t, notOk)
	_, notOk = js.configMapInterfaces[namespaceName]
	require.False(t, notOk)
	for _, worker := range js.requestQueueWorkers {
		_, notOk = worker.podInterface[namespaceName]
		require.False(t, notOk)
		_, notOk = worker.jobInterface[namespaceName]
		require.False(t, notOk)
		_, notOk = worker.configMapInterfaces[namespaceName]
		require.False(t, notOk)
	}

	// Try removing empty namespace name again.
	err = js.RemoveEmptyNamespace(namespaceName, clusterName)
	require.NoError(t, err)

	// Verify that the namespace is still removed from everywhere.
	_, notOk = js.podInterfaces[namespaceName]
	require.False(t, notOk)
	_, notOk = js.jobInterfaces[namespaceName]
	require.False(t, notOk)
	_, notOk = js.configMapInterfaces[namespaceName]
	require.False(t, notOk)
	for _, worker := range js.requestQueueWorkers {
		_, notOk = worker.podInterface[namespaceName]
		require.False(t, notOk)
		_, notOk = worker.jobInterface[namespaceName]
		require.False(t, notOk)
		_, notOk = worker.configMapInterfaces[namespaceName]
		require.False(t, notOk)
	}
}

func setupClientSetForTests(validNamespace string,
	invalidNamespaceName *string,
) kubernetes.Interface {
	jobsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}

	k8sNamespace := &k8sV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{Name: validNamespace},
	}

	namespaceInterface := &mocks.NamespaceInterface{}
	namespaceInterface.On("Get", context.Background(), validNamespace, metaV1.GetOptions{}).
		Return(k8sNamespace, nil).Once()

	if invalidNamespaceName != nil {
		namespaceInterface.On("Get", context.Background(), *invalidNamespaceName, metaV1.GetOptions{}).
			Return(nil, fmt.Errorf("namespace does not exist")).Once()
	}

	coreV1Interface.On("Namespaces").Return(namespaceInterface)

	configMapInterace := &mocks.ConfigMapInterface{}
	coreV1Interface.On("ConfigMaps", validNamespace).Return(configMapInterace)

	podsInterface := &mocks.PodInterface{}
	coreV1Interface.On("Pods", validNamespace).Return(podsInterface)

	jobsClientSet.On("CoreV1").Return(coreV1Interface)

	batchV1Interface := &mocks.K8sBatchV1Interface{}
	jobsInterface := &mocks.JobInterface{}
	batchV1Interface.On("Jobs", validNamespace).Return(jobsInterface)

	jobsClientSet.On("BatchV1").Return(batchV1Interface)

	return jobsClientSet
}

func TestGetNamespaceResourceQuota(t *testing.T) {
	namespaceName := "bnamespace"

	js := createMockJobsService(nil, device.CPU, false)

	js.clientSet = setupClientSetForResourceQuotaTests(namespaceName, nil, []int{3, 5}, nil, nil, nil)

	resp, err := js.getNamespaceResourceQuota(namespaceName)
	require.NoError(t, err)
	require.Equal(t, 3, int(*resp))

	js.clientSet = setupClientSetForResourceQuotaTests(namespaceName, nil, []int{5}, nil, nil, nil)

	resp, err = js.getNamespaceResourceQuota(namespaceName)
	require.NoError(t, err)
	require.Equal(t, 5, int(*resp))

	js.clientSet = setupClientSetForResourceQuotaTests(namespaceName, nil, []int{5, 10, 15}, nil, nil, nil)

	resp, err = js.getNamespaceResourceQuota(namespaceName)
	require.NoError(t, err)
	require.Equal(t, 5, int(*resp))

	js.clientSet = setupClientSetForResourceQuotaTests(namespaceName, nil, []int{}, nil, nil, nil)

	resp, err = js.getNamespaceResourceQuota(namespaceName)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func setupClientSetForResourceQuotaTests(namespaceName string, k8sNamespace *k8sV1.Namespace,
	resourceQuotaList []int, addExtraDetRQ *int, patchByteArray *[]byte, rqToCreate *k8sV1.ResourceQuota,
) kubernetes.Interface {
	jobsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}

	qlist := []k8sV1.ResourceQuota{}
	for i, v := range resourceQuotaList {
		q := k8sV1.ResourceQuota{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{Name: "q-" + strconv.Itoa(i)},
			Spec: k8sV1.ResourceQuotaSpec{
				Hard: k8sV1.ResourceList{
					k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
						NewQuantity(int64(v), resource.DecimalSI),
				},
			},
		}
		qlist = append(qlist, q)
	}
	if addExtraDetRQ != nil {
		q := &k8sV1.ResourceQuota{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Labels: map[string]string{determinedLabel: namespaceName},
				Name:   namespaceName + "-quota",
			},
			Spec: k8sV1.ResourceQuotaSpec{
				Hard: k8sV1.ResourceList{
					k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
						NewQuantity(int64(*addExtraDetRQ), resource.DecimalSI),
				},
			},
		}
		qlist = append(qlist, *q)
	}
	resourceQuotaInterface := &mocks.ResourceQuotaInterface{}
	k8sResourceQuota := &k8sV1.ResourceQuotaList{
		Items: qlist,
	}
	resourceQuotaInterface.On("List", context.TODO(), metaV1.ListOptions{}).
		Return(k8sResourceQuota, nil).Once()
	if rqToCreate != nil {
		resourceQuotaInterface.On("Create", context.TODO(), rqToCreate, metaV1.CreateOptions{}).
			Return(rqToCreate, nil).Once()
	}
	if patchByteArray != nil {
		resourceQuotaInterface.On(
			"Patch", context.TODO(),
			namespaceName+"-quota",
			types.MergePatchType,
			*patchByteArray,
			metaV1.PatchOptions{},
		).Return(rqToCreate, nil).Once()
	}
	coreV1Interface.On("ResourceQuotas", namespaceName).Return(resourceQuotaInterface)

	if k8sNamespace != nil {
		namespaceInterface := &mocks.NamespaceInterface{}
		namespaceInterface.On("Get", context.TODO(), namespaceName, metaV1.GetOptions{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
		}).
			Return(k8sNamespace, nil).Once()
		coreV1Interface.On("Namespaces").Return(namespaceInterface)
	}

	jobsClientSet.On("CoreV1").Return(coreV1Interface)

	return jobsClientSet
}

func TestDefaultNamespace(t *testing.T) {
	js := createMockJobsService(nil, device.CPU, false)

	newDefaultNamespace := "newNamespace"
	js.namespace = newDefaultNamespace
	ns := js.DefaultNamespace()
	require.Equal(t, newDefaultNamespace, ns)
}

func TestSetResourceQuota(t *testing.T) {
	namespaceName := "NamespaceC"
	js := createMockJobsService(nil, device.CPU, false)

	k8sDeterminedLabel := map[string]string{determinedLabel: namespaceName}

	k8sNamespace := &k8sV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:   namespaceName,
			Labels: k8sDeterminedLabel,
		},
	}
	// No existing Resource Quota
	newRQ := 5
	js.clientSet = setupClientSetForResourceQuotaTests(
		namespaceName, k8sNamespace, []int{}, nil, nil, &k8sV1.ResourceQuota{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{Labels: k8sDeterminedLabel, Name: namespaceName + "-quota"},
			Spec: k8sV1.ResourceQuotaSpec{
				Hard: k8sV1.ResourceList{
					k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
						NewQuantity(int64(newRQ), resource.DecimalSI),
				},
			},
		})
	err := js.setResourceQuota(newRQ, namespaceName)
	require.NoError(t, err)

	// 1 existing non determined Resource Quota that is more than the quota we are trying to set.
	newRQ = 2
	js.clientSet = setupClientSetForResourceQuotaTests(
		namespaceName, k8sNamespace, []int{5}, nil, nil, &k8sV1.ResourceQuota{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Labels: k8sDeterminedLabel,
				Name:   namespaceName + "-quota",
			},
			Spec: k8sV1.ResourceQuotaSpec{
				Hard: k8sV1.ResourceList{
					k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
						NewQuantity(int64(newRQ), resource.DecimalSI),
				},
			},
		})
	err = js.setResourceQuota(newRQ, namespaceName)
	require.NoError(t, err)

	// 1 existing non determined Resource Quota that is less than the quota we are trying to set.
	newRQ = 7
	js.clientSet = setupClientSetForResourceQuotaTests(
		namespaceName, k8sNamespace, []int{2}, nil, nil, &k8sV1.ResourceQuota{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "ResourceQuota",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Labels: k8sDeterminedLabel,
				Name:   namespaceName + "-quota",
			},
			Spec: k8sV1.ResourceQuotaSpec{
				Hard: k8sV1.ResourceList{
					k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
						NewQuantity(int64(newRQ), resource.DecimalSI),
				},
			},
		})
	err = js.setResourceQuota(newRQ, namespaceName)
	require.ErrorContains(t, err, "lower than the request limit you wish to set on this namespace")

	// 1 existing determined Resource Quota.
	detQVal := 2
	newRQ = 9
	detQuota := &k8sV1.ResourceQuota{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ResourceQuota",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Labels: map[string]string{determinedLabel: namespaceName},
			Name:   namespaceName + "-quota",
		},
		Spec: k8sV1.ResourceQuotaSpec{
			Hard: k8sV1.ResourceList{
				k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
					NewQuantity(int64(newRQ), resource.DecimalSI),
			},
		},
	}
	detQuotaToByteArray, err := json.Marshal(detQuota)
	require.NoError(t, err)
	js.clientSet = setupClientSetForResourceQuotaTests(
		namespaceName, k8sNamespace, []int{}, &detQVal, &detQuotaToByteArray, nil)
	err = js.setResourceQuota(newRQ, namespaceName)
	require.NoError(t, err)
}

func TestROCmJobsService(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func()
	}{
		{name: "GetAgentsROCM", testFunc: testROCMGetAgents},
		{name: "GetAgentROCM", testFunc: testROCMGetAgent},
		{name: "GetSlotsROCM", testFunc: testROCMGetSlots},
		{name: "GetSlotROCM", testFunc: testROCMGetSlot},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { require.Panics(t, test.testFunc) })
	}
}

func TestRMValidateResources(t *testing.T) {
	resourcePool := &kubernetesResourcePool{
		poolConfig:     &config.ResourcePoolConfig{PoolName: "test-pool"},
		maxSlotsPerPod: 4,
	}
	kubernetesRM := &ResourceManager{
		pools: map[string]*kubernetesResourcePool{
			"test-pool": resourcePool,
		},
	}

	cases := []struct {
		name            string
		validateRequest sproto.ValidateResourcesRequest
		valid           bool
	}{
		{"single node, valid", sproto.ValidateResourcesRequest{
			IsSingleNode: true,
			Slots:        3,
			ResourcePool: "test-pool",
		}, true},
		{"single node invalid, slots < max_slots_per_pod", sproto.ValidateResourcesRequest{
			IsSingleNode: true,
			Slots:        5,
			ResourcePool: "test-pool",
		}, false},
		{"non-single node valid", sproto.ValidateResourcesRequest{
			IsSingleNode: false,
			Slots:        8,
			ResourcePool: "test-pool",
		}, true},
		{"non-single node invalid, slots not divisible by max_slots_per_pod ", sproto.ValidateResourcesRequest{
			IsSingleNode: false,
			Slots:        7,
			ResourcePool: "test-pool",
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := kubernetesRM.ValidateResources(c.validateRequest)
			if c.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "invalid resource request")
			}
		})
	}
}

func testROCMGetAgents() {
	ps := createMockJobsService(createCompNodeMap(), device.ROCM, false)
	ps.getAgents() // nolint
}

func testROCMGetAgent() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	ps.getAgent(compNode1Name)
}

func testROCMGetSlots() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	ps.getSlots(compNode1Name)
}

func testROCMGetSlot() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	for i := 0; i < int(nodeNumSlots); i++ {
		ps.getSlot(compNode1Name, strconv.Itoa(i))
	}
}

func setupNodes() (*k8sV1.Node, *k8sV1.Node, *k8sV1.Node, *k8sV1.Node) {
	auxResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceCPU: *resource.NewQuantity(nodeNumSlotsCPU, resource.DecimalSI),
	}

	compResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceName(resourceTypeNvidia): *resource.NewQuantity(
			nodeNumSlots,
			resource.DecimalSI,
		),
	}

	auxNode1 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            auxNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: auxResourceList},
	}

	auxNode2 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            auxNode2Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: auxResourceList},
	}

	compNode1 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: compResourceList},
	}

	compNode2 := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode2Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: compResourceList},
	}
	return &auxNode1, &auxNode2, &compNode1, &compNode2
}

func createCompNodeMap() map[string]*k8sV1.Node {
	resourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceName(device.ROCM): *resource.NewQuantity(nodeNumSlots, resource.DecimalSI),
	}

	compNode := k8sV1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			ResourceVersion: "1",
			Name:            compNode1Name,
		},
		Status: k8sV1.NodeStatus{Allocatable: resourceList},
	}
	return map[string]*k8sV1.Node{
		compNode.Name: &compNode,
	}
}

// createMockJobsService creates two pods. One pod is run on the auxiliary node and the other is
// run on the compute node.
func createMockJobsService(nodes map[string]*k8sV1.Node, devSlotType device.Type,
	labels bool,
) *jobsService {
	var jobsList batchV1.JobList
	var podsList k8sV1.PodList
	// Create two pods that are scheduled on a node.
	jobName1 := uuid.NewString()
	job1 := &job{
		allocationID: model.AllocationID(uuid.New().String()),
		jobName:      jobName1,
		slotsPerPod:  pod1NumSlots,
		podNodeNames: map[string]string{
			jobName1: auxNode1Name,
		},
	}
	jobsList.Items = append(jobsList.Items, batchV1.Job{ObjectMeta: metaV1.ObjectMeta{Name: jobName1}})
	podsList.Items = append(podsList.Items, k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: jobName1}})

	jobName2 := uuid.NewString()
	job2 := &job{
		allocationID: model.AllocationID(uuid.New().String()),
		jobName:      jobName2,
		slotsPerPod:  pod2NumSlots,
		podNodeNames: map[string]string{
			uuid.NewString(): compNode1Name,
		},
	}
	jobsList.Items = append(jobsList.Items, batchV1.Job{ObjectMeta: metaV1.ObjectMeta{Name: jobName2}})
	podsList.Items = append(podsList.Items, k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: jobName2}})

	// Create pod that is not yet scheduled on a node.
	jobName3 := uuid.NewString()
	job3 := &job{
		allocationID: model.AllocationID(uuid.New().String()),
		jobName:      jobName3,
		slotsPerPod:  0,
		podNodeNames: map[string]string{},
	}
	jobsList.Items = append(jobsList.Items, batchV1.Job{ObjectMeta: metaV1.ObjectMeta{Name: jobName3}})
	podsList.Items = append(podsList.Items, k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: jobName3}})

	var nonDetPod *job

	if labels {
		// Give labels to all determined pods.
		for _, j := range jobsList.Items {
			j.ObjectMeta = metaV1.ObjectMeta{Labels: map[string]string{"determined": ""}}
		}

		resourceList := make(map[k8sV1.ResourceName]resource.Quantity)

		if devSlotType == device.CPU {
			resourceList[k8sV1.ResourceName(device.CPU)] = *resource.NewQuantity(nodeNumSlotsCPU,
				resource.DecimalSI)
		} else {
			resourceList[k8sV1.ResourceName(resourceTypeNvidia)] = *resource.NewQuantity(nodeNumSlots,
				resource.DecimalSI)
		}
		nonDetNode := k8sV1.Node{
			ObjectMeta: metaV1.ObjectMeta{
				ResourceVersion: "1",
				Name:            nonDetNodeName,
			},
			Status: k8sV1.NodeStatus{Allocatable: resourceList},
		}

		nodes[nonDetNode.Name] = &nonDetNode

		// Create pod without determined label.
		nonDetPod = &job{
			allocationID: model.AllocationID(uuid.New().String()),
			slotsPerPod:  0,
			podNodeNames: map[string]string{
				uuid.NewString(): nonDetNodeName,
			},
		}
		jobsList.Items = append(jobsList.Items, batchV1.Job{})
	}

	jobHandlers := map[string]*job{
		string(job1.allocationID): job1,
		string(job2.allocationID): job2,
		string(job3.allocationID): job3,
	}
	if nonDetPod != nil {
		jobHandlers[string(nonDetPod.allocationID)] = nonDetPod
	}

	// Create pod service client set.
	jobsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}
	podsInterface := &mocks.PodInterface{}
	podsInterface.On("List", mock.Anything, mock.Anything).Return(podsList, nil)
	batchV1Interface := &mocks.K8sBatchV1Interface{}
	jobsInterface := &mocks.JobInterface{}
	jobsInterface.On("List", mock.Anything, mock.Anything).Return(jobsList, nil)
	coreV1Interface.On("Pods", mock.Anything).Return(podsInterface)
	batchV1Interface.On("Jobs", mock.Anything).Return(jobsInterface)
	jobsClientSet.On("CoreV1").Return(coreV1Interface)
	jobsClientSet.On("BatchV1").Return(batchV1Interface)

	emptyNS := &mocks.PodInterface{}
	emptyNS.On("List", mock.Anything, mock.Anything).Return(&podsList, nil)

	podInterfaces := map[string]typedV1.PodInterface{"": emptyNS}
	return &jobsService{
		namespace:           "default",
		clusterName:         "",
		currentNodes:        nodes,
		jobNameToJobHandler: jobHandlers,
		slotType:            devSlotType,
		syslog:              logrus.WithField("namespace", namespace),
		nodeToSystemResourceRequests: map[string]int64{
			auxNode1Name: cpuResourceRequests,
			auxNode2Name: cpuResourceRequests,
		},
		slotResourceRequests:    config.PodSlotResourceRequests{CPU: 2},
		clientSet:               jobsClientSet,
		namespacesWithInformers: make(map[string]bool),
		podInterfaces:           podInterfaces,
	}
}
