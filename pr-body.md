## Problem

When using OpenAI Responses API format with models like minimax-m3 that embed thinking in `<think>...</think>` tags (instead of using the standard `reasoning_content` field), the thinking content is incorrectly included in the message content rather than being separated into a proper reasoning item.

Example of the issue:
```json
// Before (incorrect)
{
  "output": [{
    "type": "message",
    "content": [{"type": "output_text", "text": "<think>thinking...</think>\nactual answer"}]
  }]
}

// After (correct)
{
  "output": [
    {"type": "reasoning", "summary": [{"type": "summary_text", "text": "thinking..."}]},
    {"type": "message", "content": [{"type": "output_text", "text": "actual answer"}]}
  ]
}
```

## Solution

Add a new `think-tag-parsing` configuration option that controls how `<think>` tags in OpenAI-compatible responses are handled:

- `auto` (default): Automatically detect and parse think tags, separating reasoning from message content
- `on`: Always parse think tags (explicit mode)
- `off`: Never parse think tags, keep them as raw content

### Features

- ✅ Parse `<think>...</think>` tags in both streaming and non-streaming responses
- ✅ Separate reasoning content into proper `reasoning` output items
- ✅ Support multiple think blocks in a single response
- ✅ Handle tags that span across streaming chunks
- ✅ Configurable via config.yaml or management API
- ✅ Support hot reload when config changes
- ✅ Comprehensive unit tests (8 test cases + 4 streaming cases)
- ✅ Backward compatible (default `auto` mode only activates when tags are present)

### Configuration

Add to `config.yaml`:
```yaml
think-tag-parsing: "auto"  # auto, on, or off
```

Or via management API:
```bash
curl -X PUT http://localhost:8317/v0/management/think-tag-parsing \
  -H 'Content-Type: application/json' \
  -d '{"value":"auto"}'
```

### Implementation Details

- Modified `internal/translator/openai/openai/responses/` to add think tag parsing logic
- Added `extractThinkContent()` for non-streaming responses
- Added `processThinkTagStream()` for streaming responses (handles cross-chunk tags)
- Added global config variable with thread-safe getter/setter
- Integrated with config hot reload system
- Added management API endpoints (GET/PUT/PATCH)
- Updated `config.example.yaml` with documentation

### Testing

All existing tests pass. Added comprehensive test coverage:
- `TestExtractThinkContent`: 8 sub-tests for non-streaming parsing
- `TestProcessThinkTagStream`: 4 sub-tests for streaming parsing
- `TestConvert...SplitsThinkTags`: End-to-end integration tests
- `TestConvert...ThinkTagParsingOff`: Config off mode test
- `TestShouldParseThinkTags`: Mode validation tests

### Files Changed

```
cmd/server/main.go                                 |   2 +
config.example.yaml                                |   9 +
internal/api/handlers/management/config_basic.go   |  26 ++
internal/api/server.go                             |   4 +
internal/config/config.go                          |  23 ++
internal/translator/openai/openai/responses/init.go |  38 +++
.../responses/openai_openai-responses_response.go  | 236 ++++++++++++++---
.../openai_openai-responses_response_test.go       | 292 +++++++++++++++++++++
internal/watcher/config_reload.go                  |   6 +
```
