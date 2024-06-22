package utils

import "context"

type CtxKey string

const (
	CtxKeyAgentLog      CtxKey = "agent_log"
	CtxKeyAgentIdentity CtxKey = "agent_id"
)

// InjectAgentLogKey 将 bot 的 prefabName 注入到 context 中
func InjectAgentLogKey(ctx context.Context, logKey string) context.Context {
	return context.WithValue(ctx, CtxKeyAgentLog, logKey)
}

// InjectAgentID 将 bot 的 prefabName 注入到 context 中
func InjectAgentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyAgentIdentity, id)
}

// ExtractAgentLogKey 从 context 中获取 bot 的 prefabName
func ExtractAgentLogKey(ctx context.Context) (string, bool) {
	botID, ok := ctx.Value(CtxKeyAgentLog).(string)
	return botID, ok
}

// ExtractAgentID 从 context 中获取 bot 的 id
func ExtractAgentID(ctx context.Context) (string, bool) {
	botID, ok := ctx.Value(CtxKeyAgentIdentity).(string)
	return botID, ok
}
