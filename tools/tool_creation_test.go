package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type weatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名"`
	Unit string `json:"unit,omitempty" jsonschema_description:"温度单位，如 celsius 或 fahrenheit"`
}

type weatherOutput struct {
	Forecast string `json:"forecast"`
}

type addInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

type echoInput struct {
	Text string `json:"text"`
}

type echoTool struct{}

func (echoTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "echo",
		Desc: "回显输入文本",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"text": {Type: schema.String, Desc: "要回显的文本", Required: true},
		}),
	}, nil
}

func (echoTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	var input echoInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}

	return input.Text, nil
}

// TestCreateEinoTools 演示 Eino 创建 Tool 的常用方法。
//
// 共有 5 类：
// 1. InferTool：从 Go struct tag 推断 schema，适合普通同步函数，最推荐。
// 2. NewTool：手写 ToolInfo/schema，适合动态参数或需要精确控制 schema。
// 3. InferStreamTool/NewStreamTool：返回流式结果，适合搜索、日志、长文本分片。
// 4. InferEnhancedTool/NewEnhancedTool：返回 ToolResult，适合文本、图片、文件等多模态结果。
// 5. 手写接口：直接实现 InvokableTool/StreamableTool，适合封装已有 SDK 或复杂生命周期。
func TestCreateEinoTools(t *testing.T) {
	ctx := context.Background()

	t.Run("InferTool 通过结构体标签自动生成参数 schema", func(t *testing.T) {
		// 使用场景：业务入参已有 Go struct，想少写样板代码。
		// 步骤：定义 input struct 标签 -> 传入工具名、描述和函数 -> 用 JSON 参数调用。
		tool, err := utils.InferTool[weatherInput, weatherOutput](
			"get_weather",
			"查询指定城市天气",
			func(_ context.Context, input weatherInput) (weatherOutput, error) {
				return weatherOutput{Forecast: input.City + " 26 " + input.Unit}, nil
			},
		)
		if err != nil {
			t.Fatalf("InferTool 失败: %v", err)
		}

		info, err := tool.Info(ctx)
		if err != nil {
			t.Fatalf("获取 ToolInfo 失败: %v", err)
		}
		if info.Name != "get_weather" || info.ParamsOneOf == nil {
			t.Fatalf("ToolInfo 不符合预期: %+v", info)
		}

		got, err := tool.InvokableRun(ctx, `{"city":"上海","unit":"celsius"}`)
		if err != nil {
			t.Fatalf("调用工具失败: %v", err)
		}
		if got != `{"forecast":"上海 26 celsius"}` {
			t.Fatalf("返回结果不符合预期: %s", got)
		}
	})

	t.Run("NewTool 手写 ToolInfo 精确控制参数 schema", func(t *testing.T) {
		// 使用场景：参数来自配置、远程接口，或 schema 无法由 struct tag 表达。
		// 步骤：手写 ToolInfo -> 用 NewTool 绑定 typed function -> 调用验证。
		tool := utils.NewTool[addInput, int](
			&schema.ToolInfo{
				Name: "add",
				Desc: "计算两个整数之和",
				ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
					"a": {Type: schema.Integer, Desc: "第一个整数", Required: true},
					"b": {Type: schema.Integer, Desc: "第二个整数", Required: true},
				}),
			},
			func(_ context.Context, input addInput) (int, error) {
				return input.A + input.B, nil
			},
		)

		got, err := tool.InvokableRun(ctx, `{"a":2,"b":3}`)
		if err != nil {
			t.Fatalf("调用工具失败: %v", err)
		}
		if got != "5" {
			t.Fatalf("返回结果不符合预期: %s", got)
		}
	})

	t.Run("InferStreamTool 创建流式工具", func(t *testing.T) {
		// 使用场景：工具需要边查边返回，如搜索结果、长文本生成、日志输出。
		// 步骤：返回 StreamReader -> 调用 StreamableRun -> 循环 Recv 读取分片。
		tool, err := utils.InferStreamTool[weatherInput, string](
			"stream_weather",
			"流式返回天气描述",
			func(_ context.Context, input weatherInput) (*schema.StreamReader[string], error) {
				return schema.StreamReaderFromArray([]string{input.City, " 天气晴"}), nil
			},
		)
		if err != nil {
			t.Fatalf("InferStreamTool 失败: %v", err)
		}

		stream, err := tool.StreamableRun(ctx, `{"city":"北京"}`)
		if err != nil {
			t.Fatalf("调用流式工具失败: %v", err)
		}
		defer stream.Close()

		var chunks []string
		for {
			chunk, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("读取流失败: %v", err)
			}
			chunks = append(chunks, chunk)
		}

		if strings.Join(chunks, "") != "北京 天气晴" {
			t.Fatalf("流式结果不符合预期: %v", chunks)
		}
	})

	t.Run("InferEnhancedTool 创建多模态结果工具", func(t *testing.T) {
		// 使用场景：工具不只返回字符串，还要返回图片、文件、音视频等结构化结果。
		// 步骤：返回 ToolResult -> 用 ToolArgument.Text 传 JSON 参数 -> 读取 Parts。
		tool, err := utils.InferEnhancedTool[weatherInput](
			"weather_report",
			"返回结构化天气报告",
			func(_ context.Context, input weatherInput) (*schema.ToolResult, error) {
				return &schema.ToolResult{Parts: []schema.ToolOutputPart{
					{Type: schema.ToolPartTypeText, Text: input.City + " 空气质量良好"},
				}}, nil
			},
		)
		if err != nil {
			t.Fatalf("InferEnhancedTool 失败: %v", err)
		}

		got, err := tool.InvokableRun(ctx, &schema.ToolArgument{Text: `{"city":"杭州"}`})
		if err != nil {
			t.Fatalf("调用增强工具失败: %v", err)
		}
		if len(got.Parts) != 1 || got.Parts[0].Text != "杭州 空气质量良好" {
			t.Fatalf("增强结果不符合预期: %+v", got)
		}
	})

	t.Run("手写 InvokableTool 接口", func(t *testing.T) {
		// 使用场景：需要完全接管 JSON 解析、鉴权、连接池、重试或复用已有 SDK。
		// 步骤：实现 Info 和 InvokableRun -> 按 tool.InvokableTool 使用。
		var tool einotool.InvokableTool = echoTool{}

		got, err := tool.InvokableRun(ctx, `{"text":"hello eino"}`)
		if err != nil {
			t.Fatalf("调用手写工具失败: %v", err)
		}
		if got != "hello eino" {
			t.Fatalf("手写工具返回不符合预期: %s", got)
		}
	})
}
