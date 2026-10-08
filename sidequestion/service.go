package sidequestion

import (
	"context"
	"errors"
	"strings"

	"github.com/Autumn-27/norma/llm"
)

// 面向用户的旁路提问输出一律为简体中文。agent 大脑的提示词
// （request.go:instruction、context.go:summaryInstruction）保持原有的中文基准版本不变，
// 只有展示给用户的文本与错误做本地化。
var (
	errSideModelInterrupted = errors.New("模型响应中断，请重新提问")
	errSideNoAnswer         = errors.New("模型没有返回回答")
)

// msgSideToolUnavailable 是旁路提问试图触发工具调用时给出的回答文本：
// 旁路提问不能执行工具。
const msgSideToolUnavailable = "当前旁路提问不能执行工具操作，请在主会话中发出操作请求。"

// SideQuestionService has no harness, tool executor, transcript writer or model
// failover chain. Answer is one completion; Respond adds bounded preparation
// and at most one context-overflow recovery around that completion.
type SideQuestionService struct{ Provider llm.Provider }

type Answer struct {
	Text    string
	Usage   llm.Usage
	ToolUse bool
}

func (s SideQuestionService) Answer(ctx context.Context, req llm.CompletionRequest, streaming bool, update func(Answer)) (out Answer, err error) {
	if streaming {
		complete := false
		for ev, streamErr := range s.Provider.Stream(ctx, req) {
			if streamErr != nil {
				err = streamErr
				break
			}
			switch ev.Type {
			case llm.SETextDelta:
				out.Text += ev.Text
			case llm.SEToolUseStart:
				out.ToolUse = true
			case llm.SEMessageStart, llm.SEMessageDelta:
				out.Usage.Add(ev.Usage)
			case llm.SEMessageStop:
				complete = true
			}
			if update != nil {
				update(out)
			}
			if ctx.Err() != nil {
				err = ctx.Err()
				break
			}
		}
		if err == nil && !complete {
			err = errSideModelInterrupted
		}
	} else {
		var msg llm.Message
		msg, _, out.Usage, err = s.Provider.Complete(ctx, req)
		out.Text, out.ToolUse = msg.Text(), len(msg.ToolUses()) > 0
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err == nil && strings.TrimSpace(out.Text) == "" {
		if out.ToolUse {
			out.Text = msgSideToolUnavailable
		} else {
			err = errSideNoAnswer
		}
	}
	return out, err
}
