package json_parser

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

type AgentFinalOutput struct {
	Answer    string   `json:"answer" jsonschema:"required,description=最终回答"`
	Score     int      `json:"score" jsonschema:"minimum=0,maximum=100,description=置信度"`
	KeyPoints []string `json:"key_points" jsonschema:"description=核心要点"`
}

func TestMessageJSONParserFromToolCall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := schema.AssistantMessage("", []schema.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "output_final_result",
				Arguments: `{"answer":"AI 会继续深入行业应用","score":88,"key_points":["多模态","自动化","行业落地"]}`,
			},
		},
	})

	parser := schema.NewMessageJSONParser[AgentFinalOutput](
		&schema.MessageJSONParseConfig{
			ParseFrom: schema.MessageParseFromToolCall,
		},
	)

	output, err := parser.Parse(ctx, msg)
	if err != nil {
		t.Fatalf("parse tool-call JSON: %v", err)
	}

	if output.Answer != "AI 会继续深入行业应用" {
		t.Fatalf("unexpected answer: %q", output.Answer)
	}
	if output.Score != 88 {
		t.Fatalf("unexpected score: %d", output.Score)
	}
	if len(output.KeyPoints) != 3 {
		t.Fatalf("expected 3 key points, got %d", len(output.KeyPoints))
	}
}
