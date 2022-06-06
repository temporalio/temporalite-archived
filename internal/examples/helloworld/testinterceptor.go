package helloworld

import (
	"time"

	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

var _ interceptor.Interceptor = &Interceptor{}

type Interceptor struct {
	interceptor.InterceptorBase
}

type WorkflowInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
}

func NewTestInterceptor() *Interceptor {
	return &Interceptor{}
}

func (i *Interceptor) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	return i.InterceptorBase.InterceptClient(next)
}

func (i *Interceptor) InterceptWorkflow(ctx workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &WorkflowInterceptor{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{
			Next: next,
		},
	}
}

func (i *WorkflowInterceptor) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(outbound)
}

func (i *WorkflowInterceptor) ExecuteWorkflow(ctx workflow.Context, in *interceptor.ExecuteWorkflowInput) (interface{}, error) {
	version := workflow.GetVersion(ctx, "version", workflow.DefaultVersion, 1)
	var err error

	if version != workflow.DefaultVersion {
		var vpt string
		err = workflow.ExecuteLocalActivity(
			workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{ScheduleToCloseTimeout: time.Second}),
			"TestIntercept",
		).Get(ctx, &vpt)

		if err != nil {
			return nil, err
		}
	}

	return i.Next.ExecuteWorkflow(ctx, in)
}
