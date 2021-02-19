package internal

import (
	"context"
	"github.com/apex/log"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)


func (a *apiServer) PostTelemetryTrialInfo(
	_ context.Context, req *apiv1.PostTelemetryTrialInfoRequest) (*apiv1.PostTelemetryTrialInfoResponse, error) {

	log.Infof("Handling POST to TelemetryTrialInfo (Experiment=%d, Trial=%d, TrialType=%s)", req.TrialInfo.ExperimentId, req.TrialInfo.TrialId, req.TrialInfo.TrialFramework)


	return &apiv1.PostTelemetryTrialInfoResponse{}, nil
}
