package middleware

import (
	"context"
	"fmt"
	"testing"

	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ============================================================
// 1. BeforeAgent - 在智能体运行前修改配置
// ============================================================

// LoggingMiddleware 演示 BeforeAgent 的使用
// 场景：在智能体运行前记录日志，修改 Instruction
type LoggingMiddleware struct {
	*adk.BaseChatModelAgentMiddleware // 嵌入基础中间件，自动获得所有方法的默认实现
}

func (m *LoggingMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	fmt.Println("[BeforeAgent] 智能体即将运行")
	fmt.Printf("[BeforeAgent] 原始 Instruction: %s\n", runCtx.Instruction)

	// 可以修改 Instruction，添加额外的系统提示
	runCtx.Instruction = runCtx.Instruction + "\n\n注意：请保持回答简洁明了。"

	fmt.Printf("[BeforeAgent] 修改后 Instruction: %s\n", runCtx.Instruction)

	return ctx, runCtx, nil
}

// TestBeforeAgent 测试 BeforeAgent 中间件
// 使用场景：
// - 在智能体运行前记录日志
// - 动态修改 Instruction（如添加安全提示、格式要求等）
// - 修改工具配置
// - 添加上下文信息
func TestBeforeAgent(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 创建带中间件的智能体
	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "test_agent",
		Description: "测试智能体",
		Instruction: "你是一个友好的助手。",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{&LoggingMiddleware{}},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "你好")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			t.Fatalf("错误: %v", event.Err)
		}
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("回复:", msg.Content)
			}
		}
	}
}

// ============================================================
// 2. BeforeModelRewriteState - 在模型调用前修改状态
// ============================================================

// StateModifierMiddleware 演示 BeforeModelRewriteState 的使用
// 场景：在每次模型调用前修改消息历史
type StateModifierMiddleware struct {
	*adk.BaseChatModelAgentMiddleware // 嵌入基础中间件
}

func (m *StateModifierMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	fmt.Println("[BeforeModelRewriteState] 模型即将被调用")
	fmt.Printf("[BeforeModelRewriteState] 当前消息数量: %d\n", len(state.Messages))

	// 可以修改消息历史，例如：
	// - 添加系统消息
	// - 过滤敏感信息
	// - 压缩历史消息
	// - 添加上下文信息

	// 示例：在消息前添加一条系统提示
	systemMsg := schema.SystemMessage("请注意：这是一个测试环境。")
	newMessages := append([]*schema.Message{systemMsg}, state.Messages...)
	state.Messages = newMessages

	return ctx, state, nil
}

// TestBeforeModelRewriteState 测试 BeforeModelRewriteState 中间件
// 使用场景：
// - 在模型调用前修改消息历史
// - 添加系统提示或上下文信息
// - 过滤敏感信息
// - 压缩长对话历史
// - 添加检索到的相关信息（RAG）
func TestBeforeModelRewriteState(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "test_agent",
		Description: "测试智能体",
		Instruction: "你是一个助手。",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{&StateModifierMiddleware{}},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "你好")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			t.Fatalf("错误: %v", event.Err)
		}
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("回复:", msg.Content)
			}
		}
	}
}

// ============================================================
// 3. AfterModelRewriteState - 在模型调用后修改状态
// ============================================================

// ResponseFilterMiddleware 演示 AfterModelRewriteState 的使用
// 场景：在模型返回后过滤或修改响应
type ResponseFilterMiddleware struct {
	*adk.BaseChatModelAgentMiddleware // 嵌入基础中间件
}

func (m *ResponseFilterMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	fmt.Println("[AfterModelRewriteState] 模型已返回响应")

	// 获取最后一条消息（模型的响应）
	if len(state.Messages) > 0 {
		lastMsg := state.Messages[len(state.Messages)-1]
		fmt.Printf("[AfterModelRewriteState] 原始响应: %s\n", lastMsg.Content)

		// 可以修改响应内容，例如：
		// - 过滤敏感词
		// - 添加免责声明
		// - 格式化输出
		// - 添加引用来源

		// 示例：添加免责声明
		lastMsg.Content = lastMsg.Content + "\n\n[免责声明：以上内容由 AI 生成，仅供参考]"
	}

	return ctx, state, nil
}

// TestAfterModelRewriteState 测试 AfterModelRewriteState 中间件
// 使用场景：
// - 在模型返回后修改响应内容
// - 过滤敏感词或不当内容
// - 添加免责声明或引用来源
// - 格式化输出
// - 记录响应日志
func TestAfterModelRewriteState(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "test_agent",
		Description: "测试智能体",
		Instruction: "你是一个助手。",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{&ResponseFilterMiddleware{}},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "介绍一下人工智能")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			t.Fatalf("错误: %v", event.Err)
		}
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("回复:", msg.Content)
			}
		}
	}
}

// ============================================================
// 4. WrapInvokableToolCall - 包装同步工具调用
// ============================================================

// ToolLoggingMiddleware 演示 WrapInvokableToolCall 的使用
// 场景：在工具调用前后记录日志、计时、错误处理
type ToolLoggingMiddleware struct {
	*adk.BaseChatModelAgentMiddleware // 嵌入基础中间件
}

func (m *ToolLoggingMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	// 返回一个包装后的 endpoint
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		fmt.Printf("[WrapInvokableToolCall] 工具调用开始: %s\n", tCtx.Name)
		fmt.Printf("[WrapInvokableToolCall] 参数: %s\n", argumentsInJSON)

		// 调用原始工具
		result, err := endpoint(ctx, argumentsInJSON, opts...)

		if err != nil {
			fmt.Printf("[WrapInvokableToolCall] 工具调用失败: %v\n", err)
			return "", err
		}

		fmt.Printf("[WrapInvokableToolCall] 工具调用成功，结果: %s\n", result)
		return result, nil
	}

	return wrappedEndpoint, nil
}

// TestWrapInvokableToolCall 测试 WrapInvokableToolCall 中间件
// 使用场景：
// - 记录工具调用日志
// - 计算工具执行时间
// - 添加错误处理和重试逻辑
// - 验证工具参数
// - 缓存工具调用结果
func TestWrapInvokableToolCall(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和工具才能运行")

	// 此测试需要配置实际的工具
	// 参考 chatmodel_agent_test.go 中的工具配置示例
}

// ============================================================
// 5. WrapModel - 包装模型调用
// ============================================================

// ModelWrapperMiddleware 演示 WrapModel 的使用
// 场景：在模型调用前后添加自定义逻辑
type ModelWrapperMiddleware struct {
	*adk.BaseChatModelAgentMiddleware // 嵌入基础中间件
}

// LoggingChatModel 包装模型，添加日志功能
type LoggingChatModel struct {
	inner model.BaseChatModel
}

func (m *LoggingChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	fmt.Println("[WrapModel] 模型 Generate 调用开始")
	fmt.Printf("[WrapModel] 输入消息数量: %d\n", len(input))

	result, err := m.inner.Generate(ctx, input, opts...)

	if err != nil {
		fmt.Printf("[WrapModel] 模型调用失败: %v\n", err)
		return nil, err
	}

	fmt.Printf("[WrapModel] 模型调用成功，响应长度: %d\n", len(result.Content))
	return result, nil
}

func (m *LoggingChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	fmt.Println("[WrapModel] 模型 Stream 调用开始")
	return m.inner.Stream(ctx, input, opts...)
}

func (m *ModelWrapperMiddleware) WrapModel(ctx context.Context, chatModel model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	fmt.Println("[WrapModel] 包装模型")
	// 返回包装后的模型
	return &LoggingChatModel{inner: chatModel}, nil
}

// TestWrapModel 测试 WrapModel 中间件
// 使用场景：
// - 记录模型调用日志
// - 计算模型调用时间和成本
// - 添加重试逻辑
// - 实现模型降级（主模型失败时切换到备用模型）
// - 添加缓存层
// - 限流控制
func TestWrapModel(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "test_agent",
		Description: "测试智能体",
		Instruction: "你是一个助手。",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{&ModelWrapperMiddleware{}},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "你好")

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			t.Fatalf("错误: %v", event.Err)
		}
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("回复:", msg.Content)
			}
		}
	}
}

// ============================================================
// 使用建议总结
// ============================================================

// 中间件使用场景总结：
//
// 1. BeforeAgent - 智能体运行前
//    - 修改 Instruction（添加安全提示、格式要求）
//    - 修改工具配置
//    - 添加全局上下文
//    - 记录智能体启动日志
//
// 2. BeforeModelRewriteState - 模型调用前
//    - 修改消息历史（添加系统提示、压缩历史）
//    - 过滤敏感信息
//    - 添加检索到的相关信息（RAG）
//    - 实现对话历史管理策略
//
// 3. AfterModelRewriteState - 模型调用后
//    - 修改模型响应（过滤敏感词、添加免责声明）
//    - 格式化输出
//    - 记录响应日志
//    - 实现内容审核
//
// 4. WrapInvokableToolCall - 同步工具调用
//    - 记录工具调用日志
//    - 计算工具执行时间
//    - 添加错误处理和重试
//    - 验证工具参数
//    - 缓存工具结果
//
// 5. WrapStreamableToolCall - 流式工具调用
//    - 类似 WrapInvokableToolCall，但用于流式工具
//
// 6. WrapEnhancedInvokableToolCall - 增强型同步工具调用
//    - 类似 WrapInvokableToolCall，但用于增强型工具
//
// 7. WrapEnhancedStreamableToolCall - 增强型流式工具调用
//    - 类似 WrapStreamableToolCall，但用于增强型工具
//
// 8. WrapModel - 模型调用
//    - 记录模型调用日志和成本
//    - 添加重试逻辑
//    - 实现模型降级
//    - 添加缓存层
//    - 限流控制
//
// 最佳实践：
// - 所有中间件都嵌入 *adk.BaseChatModelAgentMiddleware，自动获得默认实现
// - 只需覆盖需要自定义的方法，无需实现所有 8 个方法
// - 中间件应该职责单一，每个中间件只做一件事
// - 多个中间件按顺序执行，注意执行顺序
// - 中间件应该是无状态的，或者使用 context 传递状态
// - 错误处理要完善，避免中间件导致整个流程失败
// - 性能敏感的场景要注意中间件的开销
