package chat_completions

import (
	"bytes"
	"context"
	"strconv"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/translator/openai/openai/responses"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type chatCompletionsStreamState struct {
	ThinkTagStream responses.ThinkTagStreamState
}

// ConvertOpenAIResponseToOpenAI normalizes a single chunk of an OpenAI-compatible streaming response.
// If the chunk is an SSE "data:" line, the prefix is stripped and the remaining JSON payload is returned.
// The "[DONE]" marker yields no output.
// When think tag parsing is enabled, <think> tags in content are split into reasoning_content.
func ConvertOpenAIResponseToOpenAI(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) [][]byte {
	if bytes.HasPrefix(rawJSON, []byte("data:")) {
		rawJSON = bytes.TrimSpace(rawJSON[5:])
	}
	if bytes.Equal(rawJSON, []byte("[DONE]")) {
		return [][]byte{}
	}

	if !responses.ShouldParseThinkTags() {
		return [][]byte{rawJSON}
	}

	root := gjson.ParseBytes(rawJSON)
	choices := root.Get("choices")
	if !choices.Exists() || !choices.IsArray() {
		return [][]byte{rawJSON}
	}

	if *param == nil {
		*param = &chatCompletionsStreamState{}
	}
	st := (*param).(*chatCompletionsStreamState)

	var modified bool
	choices.ForEach(func(i, choice gjson.Result) bool {
		delta := choice.Get("delta")
		if !delta.Exists() {
			return true
		}
		content := delta.Get("content")
		if !content.Exists() || content.String() == "" {
			return true
		}

		reasoningDelta, messageDelta := responses.ProcessThinkTagStream(&st.ThinkTagStream, content.String())

		if reasoningDelta == "" && messageDelta == content.String() {
			return true
		}

		modified = true
		prefix := "choices." + strconv.Itoa(int(i.Int())) + "."

		if reasoningDelta != "" {
			existing := gjson.GetBytes(rawJSON, prefix+"delta.reasoning_content").String()
			newVal := existing + reasoningDelta
			rawJSON, _ = sjson.SetBytes(rawJSON, prefix+"delta.reasoning_content", newVal)
		}

		if messageDelta != content.String() {
			rawJSON, _ = sjson.SetBytes(rawJSON, prefix+"delta.content", messageDelta)
		}

		return true
	})

	if !modified {
		return [][]byte{rawJSON}
	}
	return [][]byte{rawJSON}
}

// ConvertOpenAIResponseToOpenAINonStream passes through a non-streaming OpenAI response.
// When think tag parsing is enabled, <think> tags in content are split into reasoning_content.
func ConvertOpenAIResponseToOpenAINonStream(ctx context.Context, modelName string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []byte {
	if !responses.ShouldParseThinkTags() {
		return rawJSON
	}

	root := gjson.ParseBytes(rawJSON)
	choices := root.Get("choices")
	if !choices.Exists() || !choices.IsArray() {
		return rawJSON
	}

	var modified bool
	choices.ForEach(func(i, choice gjson.Result) bool {
		msg := choice.Get("message")
		if !msg.Exists() {
			return true
		}
		content := msg.Get("content")
		if !content.Exists() || content.String() == "" {
			return true
		}

		reasoning, message, hasThink := responses.ExtractThinkContent(content.String())
		if !hasThink || reasoning == "" {
			return true
		}

		modified = true
		prefix := "choices." + strconv.Itoa(int(i.Int())) + "."

		rawJSON, _ = sjson.SetBytes(rawJSON, prefix+"message.content", message)

		existing := gjson.GetBytes(rawJSON, prefix+"message.reasoning_content").String()
		if existing != "" {
			reasoning = existing + "\n" + reasoning
		}
		rawJSON, _ = sjson.SetBytes(rawJSON, prefix+"message.reasoning_content", reasoning)

		return true
	})

	if !modified {
		return rawJSON
	}
	return rawJSON
}
