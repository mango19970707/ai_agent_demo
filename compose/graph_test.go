package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/compose"
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

// ==================== 最佳实践总结 ====================

/*
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
