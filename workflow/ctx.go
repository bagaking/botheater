package workflow

import "context"

type contextKey string

const workflowCtxKey contextKey = "__wf_Ctx"

func WithCtx[T any](ctx context.Context, wfCtx T) context.Context {
	return context.WithValue(ctx, workflowCtxKey, wfCtx)
}

func CtxValue[T any](ctx context.Context) (T, bool) {
	v := ctx.Value(workflowCtxKey)
	wfCtx, ok := v.(T)
	return wfCtx, ok
}
