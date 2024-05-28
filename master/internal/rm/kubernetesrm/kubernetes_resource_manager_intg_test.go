//go:build integration
// +build integration

package kubernetesrm

import (
	"encoding/json"
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
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	pgDB, _, err := db.ResolveTestPostgres()
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
			agentsResp := test.jobsService.handleGetAgentsRequest()
			require.Equal(t, len(test.wantedAgentIDs), len(agentsResp.Agents))
			for _, agent := range agentsResp.Agents {
				_, ok := test.wantedAgentIDs[agent.Id]
				require.True(t, ok,
					fmt.Sprintf("name %s is not present in agent id list", agent.Id))
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
				k8sV1.ResourceName(ResourceTypeNvidia): *resource.NewQuantity(
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
			agentResp := test.jobsService.handleGetAgentRequest(test.wantedAgentID)
			if agentResp == nil {
				require.True(t, !test.agentExists)
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
			slotsResp := test.jobsService.handleGetSlotsRequest(test.agentID)
			if slotsResp == nil {
				require.True(t, !test.agentExists)
				return
			}
			require.Equal(t, test.wantedSlotsNum, len(slotsResp.Slots))

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

			slotResp := test.jobsService.handleGetSlotRequest(test.agentID, test.wantedSlotNum)
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
				Name:                       expectedName,
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
			ResourceManagerName:     expectedName,
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
			ResourceManagerName:     expectedName,
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
			require.Equal(t, tt.expected, len(res.Results))
		})
	}
}

func TestHealthCheck(t *testing.T) {
	mockPodInterface := &mocks.PodInterface{}
	kubernetesRM := &ResourceManager{
		config: &config.KubernetesResourceManagerConfig{
			Name: "testname",
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
				Name:   "testname",
				Status: model.Healthy,
			},
		}, kubernetesRM.HealthCheck())
	})

	t.Run("unhealthy", func(t *testing.T) {
		mockPodInterface.On("List", mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("error")).Once()
		require.Equal(t, []model.ResourceManagerHealth{
			{
				Name:   "testname",
				Status: model.Unhealthy,
			},
		}, kubernetesRM.HealthCheck())
	})
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

func testROCMGetAgents() {
	ps := createMockJobsService(createCompNodeMap(), device.ROCM, false)
	ps.handleGetAgentsRequest()
}

func testROCMGetAgent() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	ps.handleGetAgentRequest(compNode1Name)
}

func testROCMGetSlots() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	ps.handleGetSlotsRequest(compNode1Name)
}

func testROCMGetSlot() {
	nodes := createCompNodeMap()
	ps := createMockJobsService(nodes, device.ROCM, false)
	for i := 0; i < int(nodeNumSlots); i++ {
		ps.handleGetSlotRequest(compNode1Name, strconv.Itoa(i))
	}
}

func setupNodes() (*k8sV1.Node, *k8sV1.Node, *k8sV1.Node, *k8sV1.Node) {
	auxResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceCPU: *resource.NewQuantity(nodeNumSlotsCPU, resource.DecimalSI),
	}

	compResourceList := map[k8sV1.ResourceName]resource.Quantity{
		k8sV1.ResourceName(ResourceTypeNvidia): *resource.NewQuantity(
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
			resourceList[k8sV1.ResourceName(ResourceTypeNvidia)] = *resource.NewQuantity(nodeNumSlots,
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
	podsClientSet := &mocks.K8sClientsetInterface{}
	coreV1Interface := &mocks.K8sCoreV1Interface{}
	podsInterface := &mocks.PodInterface{}
	podsInterface.On("List", mock.Anything, mock.Anything).Return(podsList, nil)
	batchV1Interface := &mocks.K8sBatchV1Interface{}
	jobsInterface := &mocks.JobInterface{}
	jobsInterface.On("List", mock.Anything, mock.Anything).Return(jobsList, nil)
	coreV1Interface.On("Pods", mock.Anything).Return(podsInterface)
	batchV1Interface.On("Jobs", mock.Anything).Return(jobsInterface)
	podsClientSet.On("CoreV1").Return(coreV1Interface)
	podsClientSet.On("BatchV1").Return(batchV1Interface)

	return &jobsService{
		namespace:           "default",
		namespaceToPoolName: make(map[string]string),
		currentNodes:        nodes,
		jobNameToJobHandler: jobHandlers,
		slotType:            devSlotType,
		syslog:              logrus.WithField("namespace", namespace),
		nodeToSystemResourceRequests: map[string]int64{
			auxNode1Name: cpuResourceRequests,
			auxNode2Name: cpuResourceRequests,
		},
		slotResourceRequests: config.PodSlotResourceRequests{CPU: 2},
		clientSet:            podsClientSet,
	}
}
