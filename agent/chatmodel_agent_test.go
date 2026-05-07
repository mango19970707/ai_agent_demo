package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestChatModelAgent 演示如何使用 ChatModelAgent（ReAct 通用对话智能体）
//
// ChatModelAgent 是最基础的智能体类型，支持：
// - 与大模型进行多轮对话
// - 使用工具（Tool Calling）
// - 流式输出
// - ReAct 模式（推理-行动循环）
//
// 使用场景：
// - 通用问答助手
// - 带工具调用的智能助手
// - 单一职责的专业智能体（如代码生成、文本分析等）
func TestChatModelAgent(t *testing.T) {
	ctx := context.Background()

	// 1. 创建聊天模型
	// 这里使用 OpenAI 模型作为示例，你也可以使用其他模型（如 Ark、Ollama 等）
	chatModel, err := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini", // 使用 GPT-4o-mini 模型
		APIKey: "your-api-key-here",
	})
	if err != nil {
		t.Skipf("跳过测试：无法创建模型 - %v", err)
		return
	}

	// 2. 配置智能体
	agentConfig := &AgentConfig{
		Name:        "assistant",
		Description: "一个通用的 AI 助手",
		Instruction: `你是一个友好且专业的 AI 助手。
请用简洁、准确的方式回答用户的问题。
如果不确定答案，请诚实地告知用户。`,
	}

	// 3. 创建 ChatModelAgent
	agent, err := NewChatModelAgent(ctx, chatModel, agentConfig)
	if err != nil {
		t.Fatalf("创建智能体失败: %v", err)
	}

	// 4. 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false, // 设置为 true 可以启用流式输出
		Agent:           agent,
	})

	// 5. 发送查询
	query := "什么是人工智能？请用一句话简单解释。"
	iter := runner.Query(ctx, query)

	// 6. 处理响应
	fmt.Println("用户:", query)
	fmt.Println("助手:", "")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Print(msg.Content)
			}
		}
	}

	fmt.Println()
}

// TestChatModelAgentWithTools 演示如何创建带工具的 ChatModelAgent
//
// 工具（Tools）允许智能体执行特定操作，如：
// - 搜索网络
// - 查询数据库
// - 调用 API
// - 执行计算
func TestChatModelAgentWithTools(t *testing.T) {
	t.Skip("这是一个示例，需要配置实际的工具才能运行")

	ctx := context.Background()

	// 创建模型
	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 创建带工具的智能体
	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "tool_agent",
		Description: "一个可以使用工具的智能体",
		Instruction: "你可以使用提供的工具来帮助用户完成任务。",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			// 在这里配置工具
			// Tools: []tool.BaseTool{weatherTool, calculatorTool},
		},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "今天北京的天气怎么样？")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("错误: %v\n", event.Err)
			break
		}

		// 处理消息输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Print(msg.Content)
			}
		}
	}
}

// TestChatModelAgentStreaming 演示流式输出
//
// 流式输出可以：
// - 提供更好的用户体验（逐字显示）
// - 减少首字延迟
// - 适合长文本生成场景
func TestChatModelAgentStreaming(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	agent, _ := NewChatModelAgent(ctx, chatModel, &AgentConfig{
		Name:        "streaming_agent",
		Description: "支持流式输出的智能体",
		Instruction: "请详细回答用户的问题。",
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // 启用流式输出
		Agent:           agent,
	})

	iter := runner.Query(ctx, "请写一首关于春天的诗。")

	fmt.Println("智能体正在生成回复...")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("错误: %v\n", event.Err)
			break
		}

		// 处理输出（流式或非流式）
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Print(msg.Content)
			}
		}
	}

	fmt.Println()
}

// 使用建议：
//
// 1. 模型选择：
//    - 简单任务：使用 gpt-4o-mini 或 claude-haiku（成本低、速度快）
//    - 复杂任务：使用 gpt-4o 或 claude-sonnet（能力强、准确度高）
//    - 推理任务：使用 o1 系列模型（深度推理能力）
//
// 2. Instruction 编写：
//    - 明确角色定位（你是一个...）
//    - 说明任务要求（请...）
//    - 设定输出格式（以...格式输出）
//    - 添加约束条件（不要...、必须...）
//
// 3. 工具配置：
//    - 只添加必要的工具（避免混淆模型）
//    - 工具描述要清晰（帮助模型正确选择）
//    - 考虑工具的执行成本和延迟
//
// 4. 错误处理：
//    - 始终检查 event.Err
//    - 处理模型限流（rate limit）
//    - 处理超时和网络错误
//
// 5. 性能优化：
//    - 使用流式输出提升体验
//    - 合理设置 MaxIterations（防止无限循环）
//    - 考虑使用缓存减少重复调用
