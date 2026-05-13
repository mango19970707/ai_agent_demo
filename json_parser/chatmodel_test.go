package json_parser

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

type MyStructuredOutput struct {
	Result     string   `json:"result" jsonschema:"required,description=最终答案"`
	Reason     string   `json:"reason" jsonschema:"required,description=答案理由"`
	KeyPoints  []string `json:"key_points" jsonschema:"description=答案核心要点列表"`
	Confidence int      `json:"confidence" jsonschema:"minimum=0,maximum=100,description=置信度0-100"`
}

func TestMessageJSONParserFromContent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := schema.AssistantMessage(`{
		"result": "Go 适合构建高性能后端服务",
		"reason": "Go 同时提供了简洁语法、并发模型和稳定工具链",
		"key_points": ["并发友好", "编译速度快", "部署简单"],
		"confidence": 95
	}`, nil)

	parser := schema.NewMessageJSONParser[MyStructuredOutput](
		&schema.MessageJSONParseConfig{
			ParseFrom: schema.MessageParseFromContent,
		},
	)

	output, err := parser.Parse(ctx, msg)
	if err != nil {
		t.Fatalf("parse content JSON: %v", err)
	}

	if output.Result != "Go 适合构建高性能后端服务" {
		t.Fatalf("unexpected result: %q", output.Result)
	}
	if output.Reason == "" {
		t.Fatal("expected reason to be populated")
	}
	if len(output.KeyPoints) != 3 {
		t.Fatalf("expected 3 key points, got %d", len(output.KeyPoints))
	}
	if output.Confidence != 95 {
		t.Fatalf("unexpected confidence: %d", output.Confidence)
	}
}
