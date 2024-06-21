package history

import (
	"github.com/bagaking/botheater/tool"

	"github.com/khicago/got/util/typer"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

// PushFunctionCallMSG 如果队列中最后一个是 MSGContinue，则查到其之前
func PushFunctionCallMSG(msgs []*api.Message, callResult string) []*api.Message {
	mCall := &api.Message{
		Content: callResult,
		Name:    tool.CallPrefix,
		Role:    api.ChatRoleAssistant,
	}

	for len(msgs) > 0 && typer.SliceLast(msgs) == MSGContinue { // remote continue cmd
		msgs = msgs[:len(msgs)-1]
	}

	// todo: merge 规则可以调整
	for l := len(msgs); len(msgs) > 0 && msgs[l-1].Name == tool.CallPrefix; l = len(msgs) { // merge calls
		if str, ok := msgs[l-1].Content.(string); ok {
			mCall.Content = str + "\n\n" + callResult
		}
		msgs = msgs[:l-1]
	}
	return append(msgs, mCall)
}

var MSGContinue = &api.Message{
	Role:    api.ChatRoleUser,
	Content: "根据 function 调用结果，继续解决我的问题",
}
