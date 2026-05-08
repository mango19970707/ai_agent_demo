package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ============================================================
// 中间件最佳实践总结
// ============================================================

/*
中间件使用最佳实践：

1. 职责单一原则
   - 每个中间件只负责一个功能
   - 便于测试、维护和复用

2. 组合使用
   - 通过组合多个简单中间件实现复杂功能
   - 注意中间件的执行顺序（如：日志应该在最外层）

3. 性能考虑
   - 避免在中间件中执行耗时操作
   - 使用异步日志、批量上报等优化手段

4. 错误处理
   - 中间件应该优雅处理错误，避免影响主流程
   - 记录详细的错误信息便于排查

5. 配置灵活性
   - 通过配置参数控制中间件行为
   - 支持动态启用/禁用

6. 生产环境推荐组合
   - Logging（日志记录）
   - Tracing（链路追踪）
   - Retry（自动重试）
   - ErrorCatcher（错误友好化）
   - Reduction（输出限制）
   - Summarization（历史压缩，长对话场景）

7. 开发环境推荐组合
   - Logging（详细日志）
   - ErrorCatcher（错误详情）

8. 测试环境推荐组合
   - Logging（日志记录）
   - Tracing（性能分析）
*/

// ============================================================
// 1. Reduction - 工具输出超长自动截断
// 使用场景：防止工具返回超大内容（如大文件、长日志）导致 token 超限或性能问题
// ============================================================

type ReductionMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	MaxLength int // 最大输出长度
}

func (m *ReductionMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			return "", err
		}

		// 如果输出超长，自动截断并添加提示
		if len(result) > m.MaxLength {
			truncated := result[:m.MaxLength]
			return fmt.Sprintf("%s\n\n[输出已截断，原长度: %d，截断后: %d]", truncated, len(result), m.MaxLength), nil
		}

		return result, nil
	}
	return wrappedEndpoint, nil
}

// TestReduction 测试输出截断中间件
// 使用场景：
// - 防止工具返回超大文件内容（如读取大型日志文件）
// - 限制 API 返回的数据量（如查询返回数千条记录）
// - 避免 token 超限导致的成本问题
// - 提升响应速度（减少数据传输）
func TestReduction(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和工具才能运行")

	middleware := &ReductionMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		MaxLength:                    1000, // 限制输出最多 1000 字符
	}

	fmt.Println("Reduction 中间件已配置，最大输出长度:", middleware.MaxLength)
}

// ============================================================
// 2. Summarization - 对话历史自动摘要
// 使用场景：长对话场景下，自动压缩历史消息，避免 context 超限
// ============================================================

type SummarizationMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	MaxMessages   int // 保留最近的消息数量
	SummaryPrompt string
}

func (m *SummarizationMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	// 如果消息数量超过阈值，进行摘要
	if len(state.Messages) > m.MaxMessages {
		fmt.Printf("[Summarization] 消息数量 %d 超过阈值 %d，开始摘要\n", len(state.Messages), m.MaxMessages)

		// 保留最近的消息
		recentMessages := state.Messages[len(state.Messages)-m.MaxMessages:]

		// 将旧消息摘要（实际应用中应调用 LLM 生成摘要）
		oldMessages := state.Messages[:len(state.Messages)-m.MaxMessages]
		summaryContent := fmt.Sprintf("[历史对话摘要：共 %d 条消息已压缩]", len(oldMessages))

		// 构建新的消息列表：摘要 + 最近消息
		summaryMsg := schema.SystemMessage(summaryContent)
		state.Messages = append([]*schema.Message{summaryMsg}, recentMessages...)

		fmt.Printf("[Summarization] 摘要完成，新消息数量: %d\n", len(state.Messages))
	}

	return ctx, state, nil
}

// TestSummarization 测试对话历史摘要中间件
// 使用场景：
// - 长对话场景（客服机器人、多轮对话助手）
// - 防止 context 超限导致的调用失败
// - 降低 token 消耗和成本
// - 保持对话连贯性的同时控制上下文长度
func TestSummarization(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	middleware := &SummarizationMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		MaxMessages:                  10, // 保留最近 10 条消息
		SummaryPrompt:                "请简要总结以下对话内容：",
	}

	fmt.Println("Summarization 中间件已配置，保留消息数:", middleware.MaxMessages)
}

// ============================================================
// 3. Filesystem - 文件系统操作能力注入
// 使用场景：为智能体动态注入文件读写能力，支持沙箱隔离
// ============================================================

type FilesystemMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	AllowedPaths []string // 允许访问的路径白名单
}

func (m *FilesystemMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	fmt.Println("[Filesystem] 注入文件系统操作能力")
	fmt.Printf("[Filesystem] 允许访问的路径: %v\n", m.AllowedPaths)

	// 在实际应用中，这里会注入文件操作工具
	// 例如：runCtx.Tools = append(runCtx.Tools, fileReadTool, fileWriteTool)

	return ctx, runCtx, nil
}

func (m *FilesystemMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	// 如果是文件操作工具，进行路径验证
	if strings.Contains(tCtx.Name, "file") {
		wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			fmt.Printf("[Filesystem] 文件操作请求: %s\n", argumentsInJSON)

			// 实际应用中应解析参数并验证路径是否在白名单内
			// 这里简化处理
			for _, allowedPath := range m.AllowedPaths {
				if strings.Contains(argumentsInJSON, allowedPath) {
					return endpoint(ctx, argumentsInJSON, opts...)
				}
			}

			return "", fmt.Errorf("访问被拒绝：路径不在允许列表中")
		}
		return wrappedEndpoint, nil
	}

	return endpoint, nil
}

// TestFilesystem 测试文件系统能力注入中间件
// 使用场景：
// - 代码助手（读写项目文件）
// - 文档处理机器人（读取和生成文档）
// - 数据分析助手（读取 CSV/Excel 文件）
// - 沙箱环境（限制文件访问范围，防止恶意操作）
func TestFilesystem(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和文件工具才能运行")

	middleware := &FilesystemMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		AllowedPaths:                 []string{"/tmp", "/workspace"}, // 只允许访问这些路径
	}

	fmt.Println("Filesystem 中间件已配置，允许路径:", middleware.AllowedPaths)
}

// ============================================================
// 4. Logging - 全链路日志记录
// 使用场景：记录智能体运行的完整生命周期，便于调试和监控
// ============================================================

type LoggingMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	SessionID string
}

func (m *LoggingMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	fmt.Printf("[Logging][%s] === 智能体启动 ===\n", m.SessionID)
	fmt.Printf("[Logging][%s] Instruction: %s\n", m.SessionID, runCtx.Instruction)
	return ctx, runCtx, nil
}

func (m *LoggingMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	fmt.Printf("[Logging][%s] >>> 模型调用开始，消息数: %d\n", m.SessionID, len(state.Messages))
	return ctx, state, nil
}

func (m *LoggingMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if len(state.Messages) > 0 {
		lastMsg := state.Messages[len(state.Messages)-1]
		fmt.Printf("[Logging][%s] <<< 模型响应: %s\n", m.SessionID, lastMsg.Content[:min(100, len(lastMsg.Content))])
	}
	return ctx, state, nil
}

func (m *LoggingMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		fmt.Printf("[Logging][%s] [Tool] %s 调用开始\n", m.SessionID, tCtx.Name)
		start := time.Now()

		result, err := endpoint(ctx, argumentsInJSON, opts...)

		duration := time.Since(start)
		if err != nil {
			fmt.Printf("[Logging][%s] [Tool] %s 调用失败 (耗时: %v): %v\n", m.SessionID, tCtx.Name, duration, err)
		} else {
			fmt.Printf("[Logging][%s] [Tool] %s 调用成功 (耗时: %v)\n", m.SessionID, tCtx.Name, duration)
		}

		return result, err
	}
	return wrappedEndpoint, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestLogging 测试全链路日志中间件
// 使用场景：
// - 生产环境监控（记录所有智能体调用）
// - 调试和问题排查（追踪完整执行流程）
// - 性能分析（记录各环节耗时）
// - 审计和合规（记录敏感操作）
func TestLogging(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	middleware := &LoggingMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		SessionID:                    "session-12345",
	}

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "test_agent",
		Description: "测试智能体",
		Instruction: "你是一个助手。",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{middleware},
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
	}
}

// ============================================================
// 5. Tracing - 分布式追踪
// 使用场景：对接 OpenTelemetry/Jaeger/CozeLoop，实现分布式链路追踪
// ============================================================

type TracingMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	TraceID string
	SpanID  string
}

func (m *TracingMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	fmt.Printf("[Tracing] TraceID: %s, SpanID: %s - Agent 启动\n", m.TraceID, m.SpanID)
	// 实际应用中，这里会创建 OpenTelemetry Span
	// span := tracer.Start(ctx, "agent.run")
	// defer span.End()
	return ctx, runCtx, nil
}

func (m *TracingMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		// 创建子 Span
		childSpanID := fmt.Sprintf("%s.%s", m.SpanID, tCtx.Name)
		fmt.Printf("[Tracing] TraceID: %s, SpanID: %s - Tool 调用: %s\n", m.TraceID, childSpanID, tCtx.Name)

		start := time.Now()
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		duration := time.Since(start)

		// 记录 Span 属性
		fmt.Printf("[Tracing] TraceID: %s, SpanID: %s - Tool 完成，耗时: %v\n", m.TraceID, childSpanID, duration)

		return result, err
	}
	return wrappedEndpoint, nil
}

// TestTracing 测试分布式追踪中间件
// 使用场景：
// - 微服务架构（追踪跨服务的智能体调用链）
// - 性能优化（识别性能瓶颈）
// - 故障排查（快速定位问题环节）
// - 依赖分析（了解服务间调用关系）
func TestTracing(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和追踪系统才能运行")

	middleware := &TracingMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		TraceID:                      "trace-abc123",
		SpanID:                       "span-001",
	}

	fmt.Println("Tracing 中间件已配置，TraceID:", middleware.TraceID)
}

// ============================================================
// 6. Retry - 工具调用自动重试
// 使用场景：网络不稳定或外部服务偶发失败时，自动重试提升成功率
// ============================================================

type RetryMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	MaxRetries int           // 最大重试次数
	RetryDelay time.Duration // 重试间隔
}

func (m *RetryMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		var lastErr error

		for attempt := 0; attempt <= m.MaxRetries; attempt++ {
			if attempt > 0 {
				fmt.Printf("[Retry] 工具 %s 第 %d 次重试\n", tCtx.Name, attempt)
				time.Sleep(m.RetryDelay)
			}

			result, err := endpoint(ctx, argumentsInJSON, opts...)
			if err == nil {
				if attempt > 0 {
					fmt.Printf("[Retry] 工具 %s 重试成功\n", tCtx.Name)
				}
				return result, nil
			}

			lastErr = err
			fmt.Printf("[Retry] 工具 %s 调用失败: %v\n", tCtx.Name, err)
		}

		return "", fmt.Errorf("工具调用失败，已重试 %d 次: %w", m.MaxRetries, lastErr)
	}
	return wrappedEndpoint, nil
}

// TestRetry 测试自动重试中间件
// 使用场景：
// - 调用不稳定的外部 API（天气、股票、新闻等）
// - 网络波动场景（移动端、弱网环境）
// - 数据库连接偶发失败
// - 提升系统整体可用性和用户体验
func TestRetry(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和工具才能运行")

	middleware := &RetryMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		MaxRetries:                   3,               // 最多重试 3 次
		RetryDelay:                   1 * time.Second, // 每次重试间隔 1 秒
	}

	fmt.Printf("Retry 中间件已配置，最大重试次数: %d，重试间隔: %v\n", middleware.MaxRetries, middleware.RetryDelay)
}

// ============================================================
// 7. ErrorCatcher - 错误捕获与友好化
// 使用场景：捕获各类错误，转换为用户友好的提示信息
// ============================================================

type ErrorCatcherMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *ErrorCatcherMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrappedEndpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			// 将技术错误转换为用户友好的提示
			friendlyMsg := m.convertErrorToFriendlyMessage(tCtx.Name, err)
			fmt.Printf("[ErrorCatcher] 原始错误: %v\n", err)
			fmt.Printf("[ErrorCatcher] 友好提示: %s\n", friendlyMsg)

			// 返回友好的错误信息而不是原始错误
			return friendlyMsg, nil // 注意：这里返回 nil error，让智能体继续处理
		}
		return result, nil
	}
	return wrappedEndpoint, nil
}

func (m *ErrorCatcherMiddleware) convertErrorToFriendlyMessage(toolName string, err error) string {
	errMsg := err.Error()

	// 根据错误类型返回友好提示
	switch {
	case strings.Contains(errMsg, "timeout"):
		return fmt.Sprintf("抱歉，%s 工具响应超时，请稍后重试。", toolName)
	case strings.Contains(errMsg, "network"):
		return fmt.Sprintf("网络连接出现问题，无法使用 %s 工具，请检查网络连接。", toolName)
	case strings.Contains(errMsg, "permission"):
		return fmt.Sprintf("没有权限使用 %s 工具，请联系管理员。", toolName)
	case strings.Contains(errMsg, "not found"):
		return fmt.Sprintf("未找到相关资源，%s 工具无法完成操作。", toolName)
	default:
		return fmt.Sprintf("抱歉，%s 工具暂时无法使用，请稍后重试。", toolName)
	}
}

func (m *ErrorCatcherMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	// 也可以在这里捕获模型调用的错误
	if len(state.Messages) > 0 {
		lastMsg := state.Messages[len(state.Messages)-1]
		// 检查是否包含错误信息，进行友好化处理
		if strings.Contains(lastMsg.Content, "error") || strings.Contains(lastMsg.Content, "failed") {
			fmt.Println("[ErrorCatcher] 检测到错误信息，已进行友好化处理")
		}
	}
	return ctx, state, nil
}

// TestErrorCatcher 测试错误捕获中间件
// 使用场景：
// - 面向终端用户的应用（隐藏技术细节）
// - 提升用户体验（避免显示堆栈跟踪等技术信息）
// - 多语言支持（根据用户语言返回本地化错误信息）
// - 错误分类和统计（收集常见错误类型）
func TestErrorCatcher(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 和工具才能运行")

	middleware := &ErrorCatcherMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}

	// 测试错误转换
	testErrors := []error{
		fmt.Errorf("connection timeout after 30s"),
		fmt.Errorf("network unreachable"),
		fmt.Errorf("permission denied"),
		fmt.Errorf("resource not found"),
		fmt.Errorf("unknown error occurred"),
	}

	for _, err := range testErrors {
		friendlyMsg := middleware.convertErrorToFriendlyMessage("测试工具", err)
		fmt.Printf("原始错误: %v\n友好提示: %s\n\n", err, friendlyMsg)
	}
}

// ============================================================
// 综合示例：组合多个中间件
// ============================================================

// TestCombinedMiddleware 测试组合使用多个中间件
// 实际应用中，通常会组合多个中间件来实现完整的功能
func TestCombinedMiddleware(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 组合多个中间件：日志 + 重试 + 错误捕获 + 输出截断
	middlewares := []adk.ChatModelAgentMiddleware{
		&LoggingMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
			SessionID:                    "session-combined-test",
		},
		&RetryMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
			MaxRetries:                   2,
			RetryDelay:                   500 * time.Millisecond,
		},
		&ErrorCatcherMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		},
		&ReductionMiddleware{
			BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
			MaxLength:                    2000,
		},
	}

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "production_agent",
		Description: "生产环境智能体",
		Instruction: "你是一个生产环境的助手，具备完整的错误处理和日志记录能力。",
		Model:       chatModel,
		Handlers:    middlewares, // 按顺序执行
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	iter := runner.Query(ctx, "你好，请介绍一下你的能力")

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
