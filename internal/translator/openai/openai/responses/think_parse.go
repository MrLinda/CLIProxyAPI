package responses

import (
	"strings"
)

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
	codeFence     = "```"
	backtick      = "`"
)

// ThinkTagStreamState tracks the parsing state for streaming content with <think> tags.
type ThinkTagStreamState struct {
	InThink   bool
	Pending   string
	InCode    bool
	CodeCount int
}

// ExtractThinkContent parses <think>...</think> tags from content and separates
// reasoning from message text. Ignores tags inside code fences or inline code.
func ExtractThinkContent(content string) (reasoning string, message string, hasThink bool) {
	var reasoningBuf, messageBuf strings.Builder
	inInline := false
	inCode := false
	s := content
	for {
		if inCode {
			idx := strings.Index(s, codeFence)
			if idx >= 0 {
				messageBuf.WriteString(s[:idx+len(codeFence)])
				s = s[idx+len(codeFence):]
				inCode = false
				continue
			}
			messageBuf.WriteString(s)
			break
		}
		if !inInline {
			fenceIdx := strings.Index(s, codeFence)
			inlineIdx := strings.Index(s, backtick)
			openIdx := strings.Index(s, thinkOpenTag)

			if openIdx >= 0 && (fenceIdx < 0 || openIdx < fenceIdx) && (inlineIdx < 0 || openIdx < inlineIdx) {
				messageBuf.WriteString(s[:openIdx])
				afterOpen := s[openIdx+len(thinkOpenTag):]
				closeIdx := strings.Index(afterOpen, thinkCloseTag)
				if closeIdx < 0 {
					reasoningBuf.WriteString(afterOpen)
					hasThink = true
					break
				}
				reasoningBuf.WriteString(afterOpen[:closeIdx])
				s = afterOpen[closeIdx+len(thinkCloseTag):]
				hasThink = true
				continue
			}
			if inlineIdx >= 0 && (fenceIdx < 0 || inlineIdx < fenceIdx) {
				messageBuf.WriteString(s[:inlineIdx+1])
				s = s[inlineIdx+1:]
				inInline = true
				continue
			}
			if fenceIdx >= 0 {
				messageBuf.WriteString(s[:fenceIdx+len(codeFence)])
				s = s[fenceIdx+len(codeFence):]
				inCode = true
				continue
			}
			messageBuf.WriteString(s)
			break
		}
		endIdx := strings.Index(s, backtick)
		if endIdx >= 0 {
			messageBuf.WriteString(s[:endIdx+1])
			s = s[endIdx+1:]
			inInline = false
			continue
		}
		messageBuf.WriteString(s)
		break
	}
	return reasoningBuf.String(), messageBuf.String(), hasThink
}

// ProcessThinkTagStream processes a streaming chunk and separates reasoning from message deltas.
// Handles tags that span multiple chunks and ignores tags inside code fences/inline code.
func ProcessThinkTagStream(st *ThinkTagStreamState, chunk string) (reasoningDelta, messageDelta string) {
	s := st.Pending + chunk
	st.Pending = ""

	for len(s) > 0 {
		if st.InCode {
			idx := strings.Index(s, codeFence)
			if idx >= 0 {
				messageDelta += s[:idx+len(codeFence)]
				s = s[idx+len(codeFence):]
				st.InCode = false
				continue
			}
			messageDelta += s
			return
		}

		if !st.InThink {
			fenceIdx := strings.Index(s, codeFence)
			inlineIdx := strings.Index(s, backtick)
			openIdx := strings.Index(s, thinkOpenTag)

			if openIdx >= 0 && (fenceIdx < 0 || openIdx < fenceIdx) && (inlineIdx < 0 || openIdx < inlineIdx) {
				messageDelta += s[:openIdx]
				s = s[openIdx+len(thinkOpenTag):]
				st.InThink = true
				continue
			}

			minIdx := -1
			minLen := 0
			if fenceIdx >= 0 {
				minIdx = fenceIdx
				minLen = len(codeFence)
			}
			if inlineIdx >= 0 && (minIdx < 0 || inlineIdx < minIdx) {
				minIdx = inlineIdx
				minLen = 1
			}

			if minIdx >= 0 {
				partialCheck := s[minIdx:]
				isPartialFence := strings.HasPrefix(codeFence, partialCheck) && len(partialCheck) < len(codeFence)
				isPartialInline := partialCheck == backtick

				if isPartialFence || isPartialInline {
					messageDelta += s[:minIdx]
					st.Pending = s[minIdx:]
					return
				}

				messageDelta += s[:minIdx+minLen]
				s = s[minIdx+minLen:]
				if minIdx == fenceIdx {
					st.InCode = true
				} else {
					st.CodeCount++
				}
				continue
			}

			partialLen := 0
			for i := 1; i < len(thinkOpenTag) && i <= len(s); i++ {
				if strings.HasPrefix(thinkOpenTag, s[len(s)-i:]) {
					partialLen = i
				}
			}
			if partialLen > 0 {
				messageDelta += s[:len(s)-partialLen]
				st.Pending = s[len(s)-partialLen:]
			} else {
				messageDelta += s
			}
			return
		}

		idx := strings.Index(s, thinkCloseTag)
		if idx >= 0 {
			reasoningDelta += s[:idx]
			s = s[idx+len(thinkCloseTag):]
			st.InThink = false
			continue
		}

		partialLen := 0
		for i := 1; i < len(thinkCloseTag) && i <= len(s); i++ {
			if strings.HasPrefix(thinkCloseTag, s[len(s)-i:]) {
				partialLen = i
			}
		}
		if partialLen > 0 {
			reasoningDelta += s[:len(s)-partialLen]
			st.Pending = s[len(s)-partialLen:]
		} else {
			reasoningDelta += s
		}
		return
	}
	return
}
