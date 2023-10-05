package eagerbench

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

func Activity(ctx context.Context, tStart time.Time) (time.Duration, error) {
	return time.Since(tStart), nil
}

func Workflow(ctx workflow.Context, tStart time.Time) (time.Duration, error) {
	logger := workflow.GetLogger(ctx)
	ao := workflow.LocalActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithLocalActivityOptions(ctx, ao)

	var tInteract time.Duration
	err := workflow.ExecuteLocalActivity(ctx, Activity, tStart).Get(ctx, &tInteract)
	if err != nil {
		logger.Error("Local activity failed.", "Error", err)
		return tInteract, err
	}

	return tInteract, nil
}
