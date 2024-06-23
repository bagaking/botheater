package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bagaking/goulp/wlog"
	"github.com/khicago/got/util/typer"

	"github.com/bagaking/botheater/bot"
	"github.com/bagaking/botheater/history"
)

func MultiAgentChat(ctx context.Context, h *history.History, question string, bots ...*bot.Bot) {
	l2, ctx := wlog.ByCtxAndRemoveCache(ctx, "MultiAgentChat")
	log := l2.WithField("mode", "auto")

	if len(bots) == 0 {
		return
	}
	bCur := bots[0]
	var bCoordinate *bot.Bot
	for i := range bots {
		b := bots[i]
		if b.AckAs == bot.ActAsCoordinator {
			bCoordinate = b
			break
		}
	}
	if bCoordinate != nil {
		bCur = bCoordinate
	}

	h.EnqueueUserMsg(question)

	for i := 0; i < MaxRound; i++ {
		log = log.WithField("round", i)
		if bCur == nil {
			log.Errorf("bCur cannot be nil")
			return
		}
		log.Infof("enter new round= %d", i)
		content, err := bCur.SendChat(ctx, h)
		if err != nil {
			log.WithError(err).Errorf("chat failed")
			h.EnqueueAssistantMsg("chat failed, err: "+err.Error(), bCur.PrefabName)
		} else {
			h.EnqueueAssistantMsg(content, bCur.PrefabName)
		}

		log.Infof("=== chat answer ===\n\n%s=== chat answer ===\n\n", content)
		// continue chat
		if bot.Caller.HasCall(content) { // 任何有指名 agent 的情况，都调用
			agentName, params, err := bot.Caller.ParseCall(ctx, content)
			log.Debugf("agent call found, name= %s, params= %v", agentName, params)
			if err != nil {
				log.WithError(err).Errorf("chat continue stop at round %d, parse call failed", i)
				break
			}

			q := strings.Join(params, "; ")

			ind := typer.SliceFirstMatch(bots, func(b *bot.Bot) bool {
				return b.PrefabName == agentName
			})
			if ind < 0 { // 找不到 agent 的情况以为着 coordinator 判断出错, 或者 5xx 了
				log.Errorf("cannot find next agent, call failed %s", agentName)
				break
			}

			// 设置下次执行的 agent
			bCur = bots[ind]

			// 如果上一句话是 bCoordinate 的决策过程，那么把决策过程删除，只保留谁来做的结论和任务
			// todo：可能不把 history 中的决策过程踢出去更好
			if last, ok := h.PeekTail(); ok && bCoordinate != nil && last.Identity == bCoordinate.PrefabName {
				t, _ := h.PopTail() // Pop 栈头
				log.Infof("dequeue coordinate agent msg, content= %v", t)
				if last, ok = h.PeekTail(); ok && last.Role == history.RoleUser && last.Content == ContinueMessage {
					t, _ = h.PopTail() // Pop 栈头
					log.Infof("dequeue coordinate user msg, content= %v", t)
				}

				h.EnqueueCoordinateMsg(
					fmt.Sprintf("%s 经过思考，决定接下来 agent::%s 来做:\n %s", bCoordinate.PrefabName, bCur.PrefabName, q),
					bCoordinate.PrefabName,
				)

				//
				//// 如果只是删除这个消息会导致后边判断 Coordinator 的逻辑出错，改成以 coord 身份提出
				//if bot.Caller.HasCall(q) {
				//
				//} else {
				//	h.EnqueueAssistantMsg(bCoordinate.PrefabName+" 想到要"+q+"但还要想想调用哪个 agent", bCoordinate.PrefabName)
				//}
			} else {
				h.EnqueueUserMsg(q)
			}

			log.Infof("find next agent, %s(%s)", agentName, q)
			continue

		}

		// 没有 agent 指名
		if bCoordinate == nil { // 如果又没有协调者就退出
			log.WithError(err).Warnf("chat continue stop at round %d, has no call and no coordinator", i)
			break
		}

		// 如果连续 Coordinator (上次 bCur == bCoordinate)，并且也没指定任何 agent 干活，那说明死循环了，再问也意思不大，应该用户干预
		if bCoordinate == bCur {
			log.Warnf("chat continue stop at round %d, coordinator has no idea", i)
			break
		}

		h.EnqueueUserMsg(ContinueMessage)
		bCur = bCoordinate
		log.Infof("back to coordinate")
	}
}
