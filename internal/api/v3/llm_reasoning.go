/**
 * 深度思考流式解析器
 *
 * 解析 LLM 流式输出中的 <think>...</think> 标签，
 * 将推理内容和正文内容拆分为不同的 SSE 事件类型。
 *
 * 如果模型不输出 <think> 标签，parser 自动退化为直通模式，
 * 不会阻塞或缓冲正常内容。
 *
 * @author Anner
 * Created on 2026/4/3
 */
package v3

import (
	"strings"
	"unicode/utf8"
)

type reasoningParser struct {
	stream     *llmStreamWriter
	state      parserState
	buf        string
	totalSeen  int  // 已接收的总字节数
	passthrough bool // 退化为直通模式（模型不输出 <think>）
}

type parserState int

const (
	stateDetect   parserState = iota // 检测阶段：判断是否有 <think>
	stateThinking                     // 正在输出推理内容
	stateContent                      // 推理结束，输出正文
)

// 在前 N 个字符内检测 <think>，超过则认为模型不支持
const detectWindow = 30

func newReasoningParser(stream *llmStreamWriter) *reasoningParser {
	return &reasoningParser{stream: stream, state: stateDetect}
}

// Feed 处理一个流式 chunk。
func (p *reasoningParser) Feed(chunk string) error {
	if p.passthrough {
		return p.stream.SendDelta(chunk)
	}

	p.buf += chunk
	p.totalSeen += len(chunk)

	switch p.state {
	case stateDetect:
		return p.detect()
	case stateThinking:
		return p.consumeThinking()
	case stateContent:
		return p.consumeContent()
	}
	return nil
}

// Flush 将缓冲区中剩余内容全部发送。
func (p *reasoningParser) Flush() error {
	if p.buf == "" {
		return nil
	}
	content := p.buf
	p.buf = ""

	if p.state == stateThinking {
		if err := p.emitReasoning(content); err != nil {
			return err
		}
		return p.stream.Send(llmStreamEvent{Type: "reasoning_done"})
	}
	return p.stream.SendDelta(content)
}

// detect 检测模型是否输出 <think>。
// 如果在前 detectWindow 字节内找到 <think>，进入推理模式；
// 否则退化为直通模式，把缓冲内容全部作为正文发出。
func (p *reasoningParser) detect() error {
	idx := strings.Index(p.buf, "<think>")
	if idx >= 0 {
		// 找到了 <think>，发送之前的内容（通常为空或换行）
		if idx > 0 {
			before := strings.TrimSpace(p.buf[:idx])
			if before != "" {
				if err := p.stream.SendDelta(before); err != nil {
					return err
				}
			}
		}
		p.buf = p.buf[idx+len("<think>"):]
		p.state = stateThinking
		_ = p.stream.Send(llmStreamEvent{Type: "reasoning_start"})
		return p.consumeThinking()
	}

	// 超过检测窗口还没找到 <think>，退化为直通
	if p.totalSeen > detectWindow {
		p.passthrough = true
		content := p.buf
		p.buf = ""
		return p.stream.SendDelta(content)
	}

	// 还在检测窗口内，继续缓冲
	return nil
}

// consumeThinking 消费推理内容，寻找 </think>。
func (p *reasoningParser) consumeThinking() error {
	idx := strings.Index(p.buf, "</think>")
	if idx >= 0 {
		if idx > 0 {
			if err := p.emitReasoning(p.buf[:idx]); err != nil {
				return err
			}
		}
		p.buf = p.buf[idx+len("</think>"):]
		p.state = stateContent
		_ = p.stream.Send(llmStreamEvent{Type: "reasoning_done"})
		return p.consumeContent()
	}

	// 保留可能的部分 </think> 匹配，确保不在 UTF-8 字符中间切断
	safeLen := len(p.buf) - len("</think>") + 1
	if safeLen <= 0 {
		return nil
	}
	// 回退到最近的 UTF-8 字符边界
	for safeLen > 0 && !utf8.RuneStart(p.buf[safeLen]) {
		safeLen--
	}
	if safeLen <= 0 {
		return nil
	}
	if err := p.emitReasoning(p.buf[:safeLen]); err != nil {
		return err
	}
	p.buf = p.buf[safeLen:]
	return nil
}

// consumeContent 推理结束后的正文直通。
func (p *reasoningParser) consumeContent() error {
	if p.buf == "" {
		return nil
	}
	content := p.buf
	p.buf = ""
	return p.stream.SendDelta(content)
}

func (p *reasoningParser) emitReasoning(content string) error {
	return p.stream.Send(llmStreamEvent{Type: "reasoning_delta", Content: content})
}
