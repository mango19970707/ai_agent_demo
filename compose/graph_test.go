package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

/*
===========================================
Eino Graph 测试示例
场景：请假申请流程
===========================================

## 核心概念

1. **Graph（图）**：由节点（Node）和边（Edge）组成的有向图
2. **Node（节点）**：执行具体逻辑的单元
3. **Edge（边）**：定义节点之间的连接关系
4. **Conditional Edge（条件边）**：根据状态动态决定下一个节点

## 请假场景中的 Graph 结构

```
START
  ↓
askReason (询问原因)
  ↓
askStartDate (询问开始日期)
  ↓
validateStartDate (验证开始日期)
  ├─ valid → askEndDate
  └─ invalid → askStartDate (重新询问)
  ↓
askEndDate (询问结束日期)
  ↓
validateEndDate (验证结束日期)
  ├─ valid → submitLeave
  └─ invalid → askEndDate (重新询问)
  ↓
submitLeave (提交请假)
  ↓
END
```

## Graph vs Chain

**Chain（链）**：
- 线性执行，节点按顺序执行
- 无法实现条件分支
- 适合简单的顺序流程

**Graph（图）**：
- 支持条件分支和循环
- 可以根据状态动态选择路径
- 适合复杂的业务流程

## 使用方法

运行测试：
	go test -v -run TestLeaveGraph
*/

// ==================== 数据结构 ====================

// LeaveGraphState 请假流程状态（Graph版本）
type LeaveGraphState struct {
	CurrentNode string // 当前节点名称
	Reason      string // 请假原因
	StartDate   string // 开始日期
	EndDate     string // 结束日期
	ErrorMsg    string // 错误信息（用于重试）
	RetryCount  int    // 重试次数
}

// ==================== 节点函数 ====================

// askReasonNode 询问请假原因
func askReasonNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	fmt.Println("\n🤖 请问你请假的原因是什么？（如：生病、事假、年假）")
	state.CurrentNode = "askReason"
	return state, nil
}

// askStartDateNode 询问开始日期
func askStartDateNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	if state.ErrorMsg != "" {
		fmt.Printf("❌ %s\n", state.ErrorMsg)
		state.ErrorMsg = ""
	}
	fmt.Println("🤖 请输入请假开始日期（格式：YYYY-MM-DD）")
	state.CurrentNode = "askStartDate"
	return state, nil
}

// validateStartDateNode 验证开始日期
func validateStartDateNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	if !isValidDate(state.StartDate) {
		state.ErrorMsg = "日期格式错误！请重新输入（YYYY-MM-DD）"
		state.RetryCount++
		state.CurrentNode = "validateStartDate_invalid"
		return state, nil
	}
	state.CurrentNode = "validateStartDate_valid"
	state.RetryCount = 0
	return state, nil
}

// askEndDateNode 询问结束日期
func askEndDateNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	if state.ErrorMsg != "" {
		fmt.Printf("❌ %s\n", state.ErrorMsg)
		state.ErrorMsg = ""
	}
	fmt.Println("🤖 请输入请假结束日期（格式：YYYY-MM-DD）")
	state.CurrentNode = "askEndDate"
	return state, nil
}

// validateEndDateNode 验证结束日期
func validateEndDateNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	if !isValidDate(state.EndDate) {
		state.ErrorMsg = "日期格式错误！请重新输入（YYYY-MM-DD）"
		state.RetryCount++
		state.CurrentNode = "validateEndDate_invalid"
		return state, nil
	}

	start, _ := time.Parse("2006-01-02", state.StartDate)
	end, _ := time.Parse("2006-01-02", state.EndDate)
	if end.Before(start) {
		state.ErrorMsg = "结束日期不能早于开始日期！请重新输入"
		state.RetryCount++
		state.CurrentNode = "validateEndDate_invalid"
		return state, nil
	}

	state.CurrentNode = "validateEndDate_valid"
	state.RetryCount = 0
	return state, nil
}

// submitLeaveNode 提交请假申请
func submitLeaveNode(ctx context.Context, state *LeaveGraphState) (*LeaveGraphState, error) {
	start, _ := time.Parse("2006-01-02", state.StartDate)
	end, _ := time.Parse("2006-01-02", state.EndDate)
	days := int(end.Sub(start).Hours()/24) + 1

	fmt.Println("\n✅ 请假申请提交成功！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📝 请假原因：%s\n", state.Reason)
	fmt.Printf("📅 开始日期：%s\n", state.StartDate)
	fmt.Printf("📅 结束日期：%s\n", state.EndDate)
	fmt.Printf("📆 请假天数：%d 天\n", days)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")

	state.CurrentNode = "done"
	return state, nil
}

// ==================== 分支路由函数 ====================

// routeAfterStartDateValidation 开始日期验证后的路由
func routeAfterStartDateValidation(ctx context.Context, state *LeaveGraphState) (string, error) {
	if state.CurrentNode == "validateStartDate_valid" {
		return "askEndDate", nil
	}
	return "askStartDate", nil // 验证失败，重新询问
}

// routeAfterEndDateValidation 结束日期验证后的路由
func routeAfterEndDateValidation(ctx context.Context, state *LeaveGraphState) (string, error) {
	if state.CurrentNode == "validateEndDate_valid" {
		return "submitLeave", nil
	}
	return "askEndDate", nil // 验证失败，重新询问
}

// ==================== 构建 Graph ====================

// buildLeaveGraph 构建请假流程的 Graph
func buildLeaveGraph() (compose.Runnable[*LeaveGraphState, *LeaveGraphState], error) {
	graph := compose.NewGraph[*LeaveGraphState, *LeaveGraphState]()

	// 添加节点
	graph.AddLambdaNode("askReason", compose.InvokableLambda(askReasonNode))
	graph.AddLambdaNode("askStartDate", compose.InvokableLambda(askStartDateNode))
	graph.AddLambdaNode("validateStartDate", compose.InvokableLambda(validateStartDateNode))
	graph.AddLambdaNode("askEndDate", compose.InvokableLambda(askEndDateNode))
	graph.AddLambdaNode("validateEndDate", compose.InvokableLambda(validateEndDateNode))
	graph.AddLambdaNode("submitLeave", compose.InvokableLambda(submitLeaveNode))

	// 添加边
	graph.AddEdge(compose.START, "askReason")
	graph.AddEdge("askReason", "askStartDate")
	graph.AddEdge("askStartDate", "validateStartDate")

	// 分支：根据验证结果决定下一步
	graph.AddBranch("validateStartDate", compose.NewGraphBranch(
		routeAfterStartDateValidation,
		map[string]bool{
			"askStartDate": true,
			"askEndDate":   true,
		},
	))

	graph.AddEdge("askEndDate", "validateEndDate")

	// 分支：根据验证结果决定下一步
	graph.AddBranch("validateEndDate", compose.NewGraphBranch(
		routeAfterEndDateValidation,
		map[string]bool{
			"askEndDate":   true,
			"submitLeave": true,
		},
	))

	graph.AddEdge("submitLeave", compose.END)

	// 编译 Graph，设置更大的最大步数以支持重试
	runnable, err := graph.Compile(context.Background(), compose.WithMaxRunSteps(50))
	if err != nil {
		return nil, fmt.Errorf("compile graph failed: %w", err)
	}

	return runnable, nil
}

// ==================== 测试用例 ====================

// TestLeaveGraph_BasicFlow 测试基本流程（所有输入都正确）
func TestLeaveGraph_BasicFlow(t *testing.T) {
	ctx := context.Background()

	graph, err := buildLeaveGraph()
	if err != nil {
		t.Fatalf("构建Graph失败: %v", err)
	}

	t.Log("=== 测试场景：所有输入都正确 ===")

	// 初始状态
	state := &LeaveGraphState{
		Reason:    "年假",
		StartDate: "2024-06-01",
		EndDate:   "2024-06-05",
	}

	// 执行 Graph
	result, err := graph.Invoke(ctx, state)
	if err != nil {
		t.Fatalf("执行Graph失败: %v", err)
	}

	// 验证结果
	if result.CurrentNode != "done" {
		t.Errorf("期望 CurrentNode=done，实际: %s", result.CurrentNode)
	}

	if result.Reason != "年假" {
		t.Errorf("期望 Reason=年假，实际: %s", result.Reason)
	}

	t.Log("✓ 基本流程测试通过")
}

// TestLeaveGraph_InvalidStartDate 测试开始日期格式错误
func TestLeaveGraph_InvalidStartDate(t *testing.T) {
	ctx := context.Background()

	graph, err := buildLeaveGraph()
	if err != nil {
		t.Fatalf("构建Graph失败: %v", err)
	}

	t.Log("=== 测试场景：开始日期格式错误（演示循环重试机制） ===")

	// 初始状态（开始日期格式错误）
	state := &LeaveGraphState{
		Reason:    "病假",
		StartDate: "2024/06/01", // 错误格式
		EndDate:   "2024-06-05",
	}

	t.Log("注意：由于 Graph 会在单次 Invoke 中循环重试，")
	t.Log("如果数据一直错误，会超过最大步数限制。")
	t.Log("实际应用中，应该在外部循环中修正数据后重新调用。")
	t.Log("")
	t.Log("正确的使用方式：")
	t.Log("1. 调用 Graph.Invoke()")
	t.Log("2. 检查返回的状态")
	t.Log("3. 如果验证失败，修正数据")
	t.Log("4. 重新调用 Graph.Invoke()")

	// 第一次执行：会因为日期格式错误而超过最大步数
	_, err = graph.Invoke(ctx, state)
	if err == nil {
		t.Fatal("期望因超过最大步数而失败")
	}

	t.Logf("✓ 如预期，Graph 因循环重试超过最大步数而失败: %v", err)
	t.Log("\n=== 演示正确的重试方式 ===")

	// 修正日期后重新执行
	state.StartDate = "2024-06-01"
	state.CurrentNode = "" // 重置状态
	state.RetryCount = 0

	result, err := graph.Invoke(ctx, state)
	if err != nil {
		t.Fatalf("执行Graph失败: %v", err)
	}

	if result.CurrentNode != "done" {
		t.Errorf("期望 CurrentNode=done，实际: %s", result.CurrentNode)
	}

	t.Log("✓ 修正数据后执行成功")
}

// TestLeaveGraph_InvalidEndDate 测试结束日期早于开始日期
func TestLeaveGraph_InvalidEndDate(t *testing.T) {
	ctx := context.Background()

	graph, err := buildLeaveGraph()
	if err != nil {
		t.Fatalf("构建Graph失败: %v", err)
	}

	t.Log("=== 测试场景：结束日期早于开始日期（演示循环重试机制） ===")

	// 初始状态（结束日期早于开始日期）
	state := &LeaveGraphState{
		Reason:    "事假",
		StartDate: "2024-06-10",
		EndDate:   "2024-06-05", // 早于开始日期
	}

	// 第一次执行：会因为日期逻辑错误而超过最大步数
	_, err = graph.Invoke(ctx, state)
	if err == nil {
		t.Fatal("期望因超过最大步数而失败")
	}

	t.Logf("✓ 如预期，Graph 因循环重试超过最大步数而失败")

	// 修正日期后重新执行
	t.Log("\n=== 修正日期后重新执行 ===")
	state.EndDate = "2024-06-15"
	state.CurrentNode = ""
	state.RetryCount = 0

	result, err := graph.Invoke(ctx, state)
	if err != nil {
		t.Fatalf("执行Graph失败: %v", err)
	}

	if result.CurrentNode != "done" {
		t.Errorf("期望 CurrentNode=done，实际: %s", result.CurrentNode)
	}

	t.Log("✓ 修正后执行成功")
}

// TestLeaveGraph_MultipleRetries 测试多次重试（演示外部循环控制）
func TestLeaveGraph_MultipleRetries(t *testing.T) {
	ctx := context.Background()

	graph, err := buildLeaveGraph()
	if err != nil {
		t.Fatalf("构建Graph失败: %v", err)
	}

	t.Log("=== 测试场景：演示外部循环控制重试 ===")
	t.Log("说明：Graph 内部的循环重试会导致超过最大步数，")
	t.Log("实际应用中应该在外部循环中控制重试逻辑。")

	// 模拟多次尝试的数据
	attempts := []struct {
		startDate string
		valid     bool
	}{
		{"invalid-date-1", false},
		{"invalid-date-2", false},
		{"2024-06-01", true},
	}

	state := &LeaveGraphState{
		Reason:  "年假",
		EndDate: "2024-06-05",
	}

	for i, attempt := range attempts {
		t.Logf("\n--- 第 %d 次尝试：StartDate=%s ---", i+1, attempt.startDate)

		state.StartDate = attempt.startDate
		state.CurrentNode = ""
		state.RetryCount = 0

		result, err := graph.Invoke(ctx, state)

		if attempt.valid {
			if err != nil {
				t.Fatalf("执行Graph失败: %v", err)
			}
			if result.CurrentNode != "done" {
				t.Errorf("期望 CurrentNode=done，实际: %s", result.CurrentNode)
			}
			t.Log("✓ 数据有效，执行成功")
		} else {
			if err == nil {
				t.Fatal("期望因数据无效而失败")
			}
			t.Logf("✓ 数据无效，如预期失败")
		}
	}

	t.Log("\n✓ 外部循环控制重试测试通过")
}

// TestLeaveGraph_CompleteFlow 测试完整的交互流程
func TestLeaveGraph_CompleteFlow(t *testing.T) {
	ctx := context.Background()

	graph, err := buildLeaveGraph()
	if err != nil {
		t.Fatalf("构建Graph失败: %v", err)
	}

	t.Log("=== 测试场景：完整的交互流程（外部循环控制） ===")

	// 模拟用户输入序列（在外部循环中修正错误）
	testCases := []struct {
		name      string
		state     *LeaveGraphState
		expectErr bool
	}{
		{
			name: "正常流程",
			state: &LeaveGraphState{
				Reason:    "年假",
				StartDate: "2024-06-01",
				EndDate:   "2024-06-10",
			},
			expectErr: false,
		},
		{
			name: "开始日期格式错误",
			state: &LeaveGraphState{
				Reason:    "病假",
				StartDate: "2024/06/01",
				EndDate:   "2024-06-05",
			},
			expectErr: true,
		},
		{
			name: "结束日期早于开始日期",
			state: &LeaveGraphState{
				Reason:    "事假",
				StartDate: "2024-06-20",
				EndDate:   "2024-06-15",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := graph.Invoke(ctx, tc.state)

			if tc.expectErr {
				if err == nil {
					t.Error("期望执行失败，但成功了")
				} else {
					t.Logf("✓ 如预期失败: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("执行失败: %v", err)
				}
				if result.CurrentNode != "done" {
					t.Errorf("期望 CurrentNode=done，实际: %s", result.CurrentNode)
				}
				t.Log("✓ 执行成功")
			}
		})
	}

	t.Log("\n✓ 完整流程测试通过")
}

// ==================== Graph with State and Tools 示例 ====================

/*
===========================================
Eino Graph with State and Tools 测试示例
场景：智能助手（带工具调用和状态管理）
===========================================

## 核心概念

1. **WithGenLocalState**：为 Graph 添加全局状态管理
2. **StatePreHandler**：节点执行前处理状态
3. **StatePostHandler**：节点执行后处理状态
4. **ProcessState**：在节点内部访问和修改状态
5. **Tool Integration**：集成工具调用能力

## 智能助手场景中的 Graph 结构

```
START
  ↓
chatModel (AI 决策是否需要调用工具)
  ├─ need_tool → toolsNode (执行工具)
  │                ↓
  │              chatModel (基于工具结果生成回答)
  └─ no_tool → END (直接返回回答)
```

## State 的作用

- 记录对话历史
- 跟踪工具调用记录
- 在节点间共享上下文信息

## 使用方法

运行测试：
	go test -v -run TestGraphWithStateAndTools
*/

// AgentState 智能助手的状态
type AgentState struct {
	Messages       []*schema.Message // 对话历史
	ToolCallCount  int               // 工具调用次数
	LastToolResult string            // 最后一次工具调用结果
}

// AgentInput 智能助手的输入
type AgentInput struct {
	Query string // 用户查询
}

// AgentOutput 智能助手的输出
type AgentOutput struct {
	Response      string   // AI 回复
	ToolsUsed     []string // 使用的工具列表
	ToolCallCount int      // 工具调用次数
}

// ==================== Mock Tools ====================

// getUserInfoTool 模拟获取用户信息的工具
func getUserInfoTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "get_user_info",
		Desc: "获取用户的详细信息，包括姓名、年龄、城市等",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"user_id": {
				Type:     schema.String,
				Desc:     "用户ID",
				Required: true,
			},
		}),
	}
}

// getWeatherTool 模拟获取天气信息的工具
func getWeatherTool() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "get_weather",
		Desc: "获取指定城市的天气信息",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     schema.String,
				Desc:     "城市名称",
				Required: true,
			},
		}),
	}
}

// mockToolExecutor 模拟工具执行器
func mockToolExecutor(ctx context.Context, toolName string, args map[string]any) (string, error) {
	switch toolName {
	case "get_user_info":
		userID, _ := args["user_id"].(string)
		return fmt.Sprintf(`{"user_id": "%s", "name": "张三", "age": 28, "city": "北京"}`, userID), nil
	case "get_weather":
		city, _ := args["city"].(string)
		return fmt.Sprintf(`{"city": "%s", "temperature": "22°C", "condition": "晴天"}`, city), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// ==================== Mock ChatModel with Tools ====================

// mockChatModelWithTools 支持工具调用的 Mock ChatModel
type mockChatModelWithTools struct {
	tools []*schema.ToolInfo
}

func (m *mockChatModelWithTools) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	lastMsg := messages[len(messages)-1]

	// 如果是工具调用结果，生成最终回答
	if lastMsg.Role == schema.Tool {
		return schema.AssistantMessage("根据查询结果，用户张三今年28岁，住在北京。北京今天天气晴朗，温度22°C。", nil), nil
	}

	// 检查是否需要调用工具
	content := lastMsg.Content
	needTool := false
	var toolCalls []schema.ToolCall

	if contains(content, "用户") || contains(content, "信息") {
		needTool = true
		toolCalls = append(toolCalls, schema.ToolCall{
			ID: "call_1",
			Function: schema.FunctionCall{
				Name:      "get_user_info",
				Arguments: `{"user_id": "12345"}`,
			},
		})
	}

	if contains(content, "天气") {
		needTool = true
		toolCalls = append(toolCalls, schema.ToolCall{
			ID: "call_2",
			Function: schema.FunctionCall{
				Name:      "get_weather",
				Arguments: `{"city": "北京"}`,
			},
		})
	}

	if needTool {
		// 返回工具调用请求
		return &schema.Message{
			Role:      schema.Assistant,
			Content:   "",
			ToolCalls: toolCalls,
		}, nil
	}

	// 直接回答
	return schema.AssistantMessage("我是一个智能助手，可以帮你查询用户信息和天气。请问有什么可以帮助你的？", nil), nil
}

func (m *mockChatModelWithTools) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}
	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockChatModelWithTools) BindTools(tools []*schema.ToolInfo) error {
	m.tools = tools
	return nil
}

func (m *mockChatModelWithTools) GetType(ctx context.Context) string {
	return "mock_chat_model_with_tools"
}

func (m *mockChatModelWithTools) IsCallbacksEnabled() bool {
	return false
}

// ==================== Graph Nodes ====================

// chatModelNode ChatModel 节点
func chatModelNode(ctx context.Context, cm model.BaseChatModel) func(context.Context, *AgentInput) (*schema.Message, error) {
	return func(ctx context.Context, input *AgentInput) (*schema.Message, error) {
		// 从状态中获取对话历史
		var messages []*schema.Message
		err := compose.ProcessState[*AgentState](ctx, func(_ context.Context, state *AgentState) error {
			messages = append(messages, state.Messages...)
			return nil
		})
		if err != nil {
			return nil, err
		}

		// 添加用户消息
		messages = append(messages, schema.UserMessage(input.Query))

		fmt.Printf("\n🤖 ChatModel 处理中...\n")
		fmt.Printf("  消息数: %d\n", len(messages))

		// 调用 ChatModel
		response, err := cm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("chatmodel generate failed: %w", err)
		}

		if len(response.ToolCalls) > 0 {
			fmt.Printf("  需要调用 %d 个工具\n", len(response.ToolCalls))
		} else {
			fmt.Printf("  直接回答: %s\n", response.Content)
		}

		return response, nil
	}
}

// toolsNode 工具执行节点
func toolsNode(ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
	if len(msg.ToolCalls) == 0 {
		return nil, nil
	}

	fmt.Printf("\n🔧 执行工具调用...\n")

	var toolMessages []*schema.Message
	for _, tc := range msg.ToolCalls {
		fmt.Printf("  工具: %s\n", tc.Function.Name)

		// 解析参数
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("parse tool args failed: %w", err)
		}

		// 执行工具
		result, err := mockToolExecutor(ctx, tc.Function.Name, args)
		if err != nil {
			return nil, err
		}

		fmt.Printf("  结果: %s\n", result)

		// 创建工具消息
		toolMessages = append(toolMessages, &schema.Message{
			Role:       schema.Tool,
			Content:    result,
			ToolCallID: tc.ID,
		})

		// 更新状态中的工具调用计数
		_ = compose.ProcessState[*AgentState](ctx, func(_ context.Context, state *AgentState) error {
			state.ToolCallCount++
			state.LastToolResult = result
			return nil
		})
	}

	return toolMessages, nil
}

// finalResponseNode 生成最终回答
func finalResponseNode(ctx context.Context, cm model.BaseChatModel) func(context.Context, []*schema.Message) (*AgentOutput, error) {
	return func(ctx context.Context, toolResults []*schema.Message) (*AgentOutput, error) {
		// 从状态中获取完整对话历史
		var messages []*schema.Message
		var toolCallCount int
		var toolsUsed []string

		err := compose.ProcessState[*AgentState](ctx, func(_ context.Context, state *AgentState) error {
			messages = append(messages, state.Messages...)
			toolCallCount = state.ToolCallCount
			return nil
		})
		if err != nil {
			return nil, err
		}

		// 添加工具结果
		messages = append(messages, toolResults...)

		fmt.Printf("\n🤖 生成最终回答...\n")

		// 调用 ChatModel 生成最终回答
		response, err := cm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("generate final response failed: %w", err)
		}

		// 统计使用的工具
		for _, msg := range messages {
			if msg.Role == schema.Tool {
				toolsUsed = append(toolsUsed, "tool_call")
			}
		}

		fmt.Printf("  最终回答: %s\n", response.Content)

		return &AgentOutput{
			Response:      response.Content,
			ToolsUsed:     toolsUsed,
			ToolCallCount: toolCallCount,
		}, nil
	}
}

// directResponseNode 直接返回回答（无需工具）
func directResponseNode(ctx context.Context, msg *schema.Message) (*AgentOutput, error) {
	return &AgentOutput{
		Response:      msg.Content,
		ToolsUsed:     []string{},
		ToolCallCount: 0,
	}, nil
}

// ==================== State Handlers ====================

// statePreHandler 节点执行前的状态处理
func statePreHandler(ctx context.Context, input *AgentInput, state *AgentState) (*AgentInput, error) {
	fmt.Printf("\n📊 State PreHandler\n")
	fmt.Printf("  当前对话历史: %d 条\n", len(state.Messages))
	fmt.Printf("  工具调用次数: %d\n", state.ToolCallCount)
	return input, nil
}

// statePostHandler 节点执行后的状态处理
func statePostHandler(ctx context.Context, output *schema.Message, state *AgentState) (*schema.Message, error) {
	fmt.Printf("\n📊 State PostHandler\n")
	// 将 AI 回复添加到对话历史
	state.Messages = append(state.Messages, output)
	fmt.Printf("  更新对话历史: %d 条\n", len(state.Messages))
	return output, nil
}

// ==================== Router ====================

// routeAfterChatModel 根据 ChatModel 输出决定下一步
func routeAfterChatModel(ctx context.Context, msg *schema.Message) (string, error) {
	if len(msg.ToolCalls) > 0 {
		fmt.Printf("\n🔀 路由: 需要调用工具\n")
		return "toolsNode", nil
	}
	fmt.Printf("\n🔀 路由: 直接返回回答\n")
	return "directResponse", nil
}

// ==================== Build Graph ====================

// buildGraphWithStateAndTools 构建带状态和工具的 Graph
func buildGraphWithStateAndTools(cm model.BaseChatModel) (compose.Runnable[*AgentInput, *AgentOutput], error) {
	// 创建状态生成函数
	genState := func(ctx context.Context) *AgentState {
		return &AgentState{
			Messages:      []*schema.Message{},
			ToolCallCount: 0,
		}
	}

	// 创建 Graph，启用状态管理
	graph := compose.NewGraph[*AgentInput, *AgentOutput](
		compose.WithGenLocalState(genState),
	)

	// 节点1: ChatModel（带状态处理）
	graph.AddLambdaNode("chatModel",
		compose.InvokableLambda(chatModelNode(context.Background(), cm)),
		compose.WithStatePreHandler(statePreHandler),
		compose.WithStatePostHandler(statePostHandler),
	)

	// 节点2: 工具执行
	graph.AddLambdaNode("toolsNode",
		compose.InvokableLambda(toolsNode),
	)

	// 节点3: 生成最终回答
	graph.AddLambdaNode("finalResponse",
		compose.InvokableLambda(finalResponseNode(context.Background(), cm)),
	)

	// 节点4: 直接返回回答
	graph.AddLambdaNode("directResponse",
		compose.InvokableLambda(directResponseNode),
	)

	// 添加边
	graph.AddEdge(compose.START, "chatModel")

	// 条件分支：根据是否需要工具调用选择路径
	graph.AddBranch("chatModel", compose.NewGraphBranch(
		routeAfterChatModel,
		map[string]bool{
			"toolsNode":      true,
			"directResponse": true,
		},
	))

	// 工具执行后返回 ChatModel 生成最终回答
	graph.AddEdge("toolsNode", "finalResponse")

	// 连接到 END
	graph.AddEdge("finalResponse", compose.END)
	graph.AddEdge("directResponse", compose.END)

	// 编译 Graph
	return graph.Compile(context.Background(), compose.WithMaxRunSteps(20))
}

// ==================== Tests ====================

// TestGraphWithStateAndTools_WithTools 测试需要工具调用的场景
func TestGraphWithStateAndTools_WithTools(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	cm := &mockChatModelWithTools{}

	// 绑定工具
	tools := []*schema.ToolInfo{
		getUserInfoTool(),
		getWeatherTool(),
	}
	if err := cm.BindTools(tools); err != nil {
		t.Fatalf("绑定工具失败: %v", err)
	}

	// 构建 Graph
	graph, err := buildGraphWithStateAndTools(cm)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：需要工具调用 ===")

	// 执行查询
	input := &AgentInput{
		Query: "请帮我查询用户12345的信息，以及北京的天气",
	}

	result, err := graph.Invoke(ctx, input)
	if err != nil {
		t.Fatalf("执行 Graph 失败: %v", err)
	}

	// 验证结果
	t.Logf("\n✅ 执行成功")
	t.Logf("  回答: %s", result.Response)
	t.Logf("  使用的工具数: %d", len(result.ToolsUsed))
	t.Logf("  工具调用次数: %d", result.ToolCallCount)

	if result.Response == "" {
		t.Error("回答为空")
	}

	if result.ToolCallCount == 0 {
		t.Error("期望有工具调用，但实际为0")
	}
}

// TestGraphWithStateAndTools_NoTools 测试不需要工具调用的场景
func TestGraphWithStateAndTools_NoTools(t *testing.T) {
	ctx := context.Background()

	cm := &mockChatModelWithTools{}
	graph, err := buildGraphWithStateAndTools(cm)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：不需要工具调用 ===")

	input := &AgentInput{
		Query: "你好，请介绍一下你自己",
	}

	result, err := graph.Invoke(ctx, input)
	if err != nil {
		t.Fatalf("执行 Graph 失败: %v", err)
	}

	t.Logf("\n✅ 执行成功")
	t.Logf("  回答: %s", result.Response)
	t.Logf("  使用的工具数: %d", len(result.ToolsUsed))
	t.Logf("  工具调用次数: %d", result.ToolCallCount)

	if result.Response == "" {
		t.Error("回答为空")
	}

	if result.ToolCallCount != 0 {
		t.Errorf("期望无工具调用，但实际为 %d", result.ToolCallCount)
	}
}

// ==================== 辅助函数 ====================

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== 最佳实践总结 ====================

/*
## Graph with State and Tools 最佳实践

### 1. 何时使用 State

✓ 适合使用的场景：
- 需要在节点间共享上下文信息
- 需要记录执行历史
- 需要累积统计信息
- 多轮对话场景

### 2. State 设计原则

**状态结构清晰**：
- 只保存必要的状态信息
- 使用明确的字段名
- 避免嵌套过深

**状态访问模式**：
- 使用 ProcessState 读写状态
- 在 PreHandler/PostHandler 中更新状态
- 避免在多个地方修改同一状态

### 3. Tool Integration 模式

**工具定义**：
- 使用 schema.ToolInfo 定义工具
- 提供清晰的工具描述
- 定义必需和可选参数

**工具执行**：
- 在专门的 ToolsNode 中执行
- 处理工具执行错误
- 记录工具调用历史

**工具结果处理**：
- 将结果转换为 schema.Message
- 添加到对话历史
- 供后续节点使用

### 4. Graph with State vs Workflow

| 特性 | Graph with State | Workflow |
|------|------------------|----------|
| 状态管理 | 全局状态 | 节点间传递 |
| 工具集成 | 原生支持 | 需要自定义 |
| 条件分支 | ✓ | ✓ |
| 适用场景 | Agent、对话系统 | 数据处理流水线 |

### 5. 实际应用场景

**对话 Agent**：
- 维护对话历史
- 动态调用工具
- 多轮交互

**任务执行器**：
- 跟踪任务状态
- 记录执行步骤
- 错误恢复

**工作流引擎**：
- 状态机实现
- 审批流程
- 事件驱动

## Graph 最佳实践

### 1. 何时使用 Graph

✓ 适合使用的场景：
- 需要条件分支的流程
- 需要循环重试的流程
- 需要根据状态动态选择路径
- 复杂的业务流程

✗ 不适合使用的场景：
- 简单的线性流程（用 Chain 更简单）
- 不需要条件判断的流程

### 2. Graph 设计原则

**单一职责**：
- 每个节点只做一件事
- 验证逻辑独立成节点
- 业务逻辑和展示逻辑分离

**状态驱动**：
- 使用 CurrentNode 记录当前位置
- 使用 ErrorMsg 传递错误信息
- 使用 RetryCount 记录重试次数

**条件路由**：
- 使用条件边实现分支逻辑
- 路由函数应该简单明确
- 避免在路由函数中修改状态

### 3. Graph vs Chain vs Workflow

| 特性 | Chain | Graph | Workflow |
|------|-------|-------|----------|
| 执行顺序 | 线性 | 有向图 | 复杂流程 |
| 条件分支 | ✗ | ✓ | ✓ |
| 循环重试 | ✗ | ✓ | ✓ |
| 并行执行 | ✗ | ✓ | ✓ |
| 子流程 | ✗ | ✗ | ✓ |
| 复杂度 | 低 | 中 | 高 |

### 4. 错误处理

**验证失败**：
- 使用条件边返回到输入节点
- 在状态中记录错误信息
- 限制重试次数避免死循环

**系统错误**：
- 节点函数返回 error
- Graph 会停止执行并返回错误
- 调用方需要处理错误

### 5. 性能优化

**状态大小**：
- 只保存必要的状态信息
- 避免在状态中保存大对象
- 使用引用而不是复制

**节点粒度**：
- 节点不宜过大（避免单个节点执行时间过长）
- 节点不宜过小（避免过多的节点切换开销）
- 根据业务逻辑合理划分

### 6. 测试策略

**单元测试**：
- 测试每个节点函数
- 测试路由函数的逻辑
- 使用 mock 数据

**集成测试**：
- 测试完整的 Graph 流程
- 测试各种边界情况
- 测试错误处理逻辑

**场景测试**：
- 模拟真实用户输入
- 测试多次重试场景
- 测试异常情况

## 与 Chain 的对比

本示例展示了 Graph 相比 Chain 的优势：

1. **条件分支**：可以根据验证结果选择不同路径
2. **循环重试**：验证失败时可以返回重新输入
3. **状态管理**：更灵活的状态传递和更新
4. **错误处理**：更细粒度的错误处理和恢复

如果你的流程不需要这些特性，使用 Chain 会更简单。
*/
