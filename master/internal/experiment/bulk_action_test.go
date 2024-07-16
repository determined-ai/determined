package experiment

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type bulkActorMock struct {
	mock.Mock
}

func (m *bulkActorMock) experimentsEditableByUser(
	ctx context.Context,
	projectID int32,
	experimentIDs []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]int32, error) {
	returns := m.Called(ctx, projectID, experimentIDs, filters)
	ret0, _ := returns.Get(0).([]int32)
	return ret0, returns.Error(1)
}

// experimentMock inherits all the methods of the Experiment
// interface, but only implements the ones we care about
// for testing
type experimentMock struct {
	mock.Mock
	Experiment
}

func (m *experimentMock) ActivateExperiment() error {
	returns := m.Called()
	return returns.Error(0)
}

func (m *experimentMock) PauseExperiment() error {
	returns := m.Called()
	return returns.Error(0)
}

func (m *experimentMock) CancelExperiment() error {
	returns := m.Called()
	return returns.Error(0)
}

func (m *experimentMock) KillExperiment() error {
	returns := m.Called()
	return returns.Error(0)
}

func TestActivateExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}
	type experimentsEditable struct {
		expIDs []int32
		err    error
	}

	ctx := context.Background()

	tests := []struct {
		name                  string
		args                  args
		experimentsEditable   experimentsEditable
		registeredExperiments []int
		expectedActivates     []int
		expectedResults       []ExperimentActionResult
		expectedErr           bool
	}{
		{
			name:                "user login error",
			args:                args{},
			experimentsEditable: experimentsEditable{err: errors.New("user has no permision to edit experiments")},
			expectedErr:         true,
		},
		{
			name: "three experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{32, 42},
			},
			registeredExperiments: []int{42, 62},
			expectedActivates:     []int{42},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '52' not found"),
					ID:    52,
				},
				{
					Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
					ID:    32,
				},
				{
					ID: 42,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{101, 102},
			},
			registeredExperiments: []int{101, 102},
			expectedActivates:     []int{101, 102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 101,
				},
				{
					ID: 102,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up mocks
			mock := bulkActorMock{}
			mock.On(
				"experimentsEditableByUser",
				ctx,
				tt.args.projectID,
				tt.args.experimentIds,
				tt.args.filters,
			).Return(
				tt.experimentsEditable.expIDs,
				tt.experimentsEditable.err,
			)
			// override the global func with the mock
			experimentsEditableByUser = mock.experimentsEditableByUser

			for _, expID := range tt.registeredExperiments {
				exp := experimentMock{}
				if slices.Contains(tt.expectedActivates, expID) {
					exp.On("ActivateExperiment").Return(nil)
				}
				ExperimentRegistry.Add(expID, &exp)
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID)
			}

			actual, err := ActivateExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("ActivateExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expectedResults)
		})
	}
}

func TestCancelExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}
	type experimentsEditable struct {
		expIDs []int32
		err    error
	}

	ctx := context.Background()

	tests := []struct {
		name                  string
		args                  args
		experimentsEditable   experimentsEditable
		registeredExperiments []int
		expectedCancels       []int
		expectedResults       []ExperimentActionResult
		expectedErr           bool
	}{
		{
			name:                "user login error",
			args:                args{},
			experimentsEditable: experimentsEditable{err: errors.New("user has no permision to edit experiments")},
			expectedErr:         true,
		},
		{
			name: "three experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{32, 42},
			},
			registeredExperiments: []int{42, 62},
			expectedCancels:       []int{32, 42},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '52' not found"),
					ID:    52,
				},
				{
					ID: 32,
				},
				{
					ID: 42,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{101, 102},
			},
			registeredExperiments: []int{101, 102},
			expectedCancels:       []int{101, 102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 101,
				},
				{
					ID: 102,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up mocks
			mock := bulkActorMock{}
			mock.On(
				"experimentsEditableByUser",
				ctx,
				tt.args.projectID,
				tt.args.experimentIds,
				tt.args.filters,
			).Return(
				tt.experimentsEditable.expIDs,
				tt.experimentsEditable.err,
			)
			// override the global func with the mock
			experimentsEditableByUser = mock.experimentsEditableByUser

			for _, expID := range tt.registeredExperiments {
				exp := experimentMock{}
				if slices.Contains(tt.expectedCancels, expID) {
					exp.On("CancelExperiment").Return(nil)
				}
				ExperimentRegistry.Add(expID, &exp)
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID)
			}

			actual, err := CancelExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("CancelExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expectedResults)
		})
	}
}

func TestKillExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}
	type experimentsEditable struct {
		expIDs []int32
		err    error
	}

	ctx := context.Background()

	tests := []struct {
		name                  string
		args                  args
		experimentsEditable   experimentsEditable
		registeredExperiments []int
		expectedKills         []int
		expectedResults       []ExperimentActionResult
		expectedErr           bool
	}{
		{
			name:                "user login error",
			args:                args{},
			experimentsEditable: experimentsEditable{err: errors.New("user has no permision to edit experiments")},
			expectedErr:         true,
		},
		{
			name: "three experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{32, 42},
			},
			registeredExperiments: []int{42, 62},
			expectedKills:         []int{32, 42},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '52' not found"),
					ID:    52,
				},
				{
					ID: 32,
				},
				{
					ID: 42,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{101, 102},
			},
			registeredExperiments: []int{101, 102},
			expectedKills:         []int{101, 102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 101,
				},
				{
					ID: 102,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up mocks
			mock := bulkActorMock{}
			mock.On(
				"experimentsEditableByUser",
				ctx,
				tt.args.projectID,
				tt.args.experimentIds,
				tt.args.filters,
			).Return(
				tt.experimentsEditable.expIDs,
				tt.experimentsEditable.err,
			)
			// override the global func with the mock
			experimentsEditableByUser = mock.experimentsEditableByUser

			for _, expID := range tt.registeredExperiments {
				exp := experimentMock{}
				if slices.Contains(tt.expectedKills, expID) {
					exp.On("KillExperiment").Return(nil)
				}
				ExperimentRegistry.Add(expID, &exp)
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID)
			}

			actual, err := KillExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("KillExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expectedResults)
		})
	}
}

func TestPauseExperiments(t *testing.T) {
	type args struct {
		projectID     int32
		experimentIds []int32
		filters       *apiv1.BulkExperimentFilters
	}
	type experimentsEditable struct {
		expIDs []int32
		err    error
	}

	ctx := context.Background()

	tests := []struct {
		name                  string
		args                  args
		experimentsEditable   experimentsEditable
		registeredExperiments []int
		expectedPauses        []int
		expectedResults       []ExperimentActionResult
		expectedErr           bool
	}{
		{
			name:                "user login error",
			args:                args{},
			experimentsEditable: experimentsEditable{err: errors.New("user has no permision to edit experiments")},
			expectedErr:         true,
		},
		{
			name: "three experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{32, 42},
			},
			registeredExperiments: []int{42, 62},
			expectedPauses:        []int{42},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '52' not found"),
					ID:    52,
				},
				{
					Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
					ID:    32,
				},
				{
					ID: 42,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{32, 42, 52},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{101, 102},
			},
			registeredExperiments: []int{101, 102},
			expectedPauses:        []int{101, 102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 101,
				},
				{
					ID: 102,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up mocks
			mock := bulkActorMock{}
			mock.On(
				"experimentsEditableByUser",
				ctx,
				tt.args.projectID,
				tt.args.experimentIds,
				tt.args.filters,
			).Return(
				tt.experimentsEditable.expIDs,
				tt.experimentsEditable.err,
			)
			// override the global func with the mock
			experimentsEditableByUser = mock.experimentsEditableByUser

			for _, expID := range tt.registeredExperiments {
				exp := experimentMock{}
				if slices.Contains(tt.expectedPauses, expID) {
					exp.On("PauseExperiment").Return(nil)
				}
				ExperimentRegistry.Add(expID, &exp)
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID)
			}

			actual, err := PauseExperiments(ctx, tt.args.projectID, tt.args.experimentIds,
				tt.args.filters)
			if (err != nil) != tt.expectedErr {
				t.Errorf("PauseExperiments() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			require.ElementsMatch(t, actual, tt.expectedResults)
		})
	}
}
