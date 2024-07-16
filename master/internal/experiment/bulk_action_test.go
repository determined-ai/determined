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
// for testing.
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
				experimentIds: []int32{132, 142, 152},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{132, 142},
			},
			registeredExperiments: []int{142, 162},
			expectedActivates:     []int{142},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '152' not found"),
					ID:    152,
				},
				{
					Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
					ID:    132,
				},
				{
					ID: 142,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{132, 142, 152},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{1101, 1102},
			},
			registeredExperiments: []int{1101, 1102},
			expectedActivates:     []int{1101, 1102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 1101,
				},
				{
					ID: 1102,
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
				require.NoError(t, ExperimentRegistry.Add(expID, &exp))
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID) //nolint:errcheck
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
				experimentIds: []int32{232, 242, 252},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{232, 242},
			},
			registeredExperiments: []int{242, 262},
			expectedCancels:       []int{232, 242},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '252' not found"),
					ID:    252,
				},
				{
					ID: 232,
				},
				{
					ID: 242,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{232, 242, 252},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{2101, 2102},
			},
			registeredExperiments: []int{2101, 2102},
			expectedCancels:       []int{2101, 2102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 2101,
				},
				{
					ID: 2102,
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
				require.NoError(t, ExperimentRegistry.Add(expID, &exp))
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID) //nolint:errcheck
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
			name: "four experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{332, 342, 352, 362},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{332, 342},
			},
			registeredExperiments: []int{342, 372},
			expectedKills:         []int{332, 342},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '352' not found"),
					ID:    352,
				},
				{
					Error: status.Error(codes.NotFound, "experiment '362' not found"),
					ID:    362,
				},
				{
					ID: 332,
				},
				{
					ID: 342,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{332, 342, 352},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{3101, 3102},
			},
			registeredExperiments: []int{3101, 3102},
			expectedKills:         []int{3101, 3102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 3101,
				},
				{
					ID: 3102,
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
				require.NoError(t, ExperimentRegistry.Add(expID, &exp))
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID) //nolint:errcheck
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
			name: "four experiments selected, one found",
			args: args{
				projectID:     1,
				experimentIds: []int32{432, 442, 452, 462},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{432, 442},
			},
			registeredExperiments: []int{442, 472},
			expectedPauses:        []int{442},
			expectedResults: []ExperimentActionResult{
				{
					Error: status.Error(codes.NotFound, "experiment '452' not found"),
					ID:    452,
				},
				{
					Error: status.Error(codes.NotFound, "experiment '462' not found"),
					ID:    462,
				},
				{
					Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
					ID:    432,
				},
				{
					ID: 442,
				},
			},
		},
		{
			name: "filters are used",
			args: args{
				projectID:     1,
				experimentIds: []int32{432, 442, 452},
				filters: &apiv1.BulkExperimentFilters{
					Description: "default",
					Name:        "test_default",
					Labels:      []string{"test"},
				},
			},
			experimentsEditable: experimentsEditable{
				expIDs: []int32{4101, 4102},
			},
			registeredExperiments: []int{4101, 4102},
			expectedPauses:        []int{4101, 4102},
			expectedResults: []ExperimentActionResult{
				{
					ID: 4101,
				},
				{
					ID: 4102,
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
				require.NoError(t, ExperimentRegistry.Add(expID, &exp))
				defer exp.AssertExpectations(t)
				defer ExperimentRegistry.Delete(expID) //nolint:errcheck
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
