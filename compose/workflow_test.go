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
Eino Workflow 测试示例
场景：请假申请流程（带审批）
===========================================

## 核心概念

1. **Workflow（工作流）**：更高级的流程抽象，支持字段级数据映射
2. **FieldMapping（字段映射）**：精细控制节点间的数据传递
3. **AddInput**：添加数据输入源
4. **AddDependency**：添加控制依赖（不传递数据）
5. **Parallel（并行）**：通过 FieldMapping 实现并行任务

## 请假场景中的 Workflow 结构

```
START
  ↓
collectInfo (收集请假信息)
  ↓
[并行验证 - 通过 FieldMapping 实现]
  ├─ validateDates (验证日期) ← 从 collectInfo 获取日期
  ├─ checkBalance (检查余额) ← 从 collectInfo 获取原因和天数
  └─ checkConflict (检查冲突) ← 从 collectInfo 获取日期
  ↓
combineValidation (合并验证结果)
  ↓
managerApproval (经理审批)
  ├─ approved → hrApproval
  └─ rejected → END
  ↓
hrApproval (HR审批)
  ├─ approved → submitLeave
  └─ rejected → END
  ↓
submitLeave (提交请假)
  ↓
END
```

## Workflow vs Graph

**Graph（图）**：
- 基础的有向图结构
- 节点间全量数据传递
- 适合中等复杂度的流程

**Workflow（工作流）**：
- 更高级的抽象
- 支持字段级数据映射（FieldMapping）
- 支持并行任务和复杂数据流
- 适合复杂的业务流程

## 使用方法

运行测试：
	go test -v -run TestLeaveWorkflow
*/

// ==================== 数据结构 ====================

// LeaveRequest 请假申请输入
type LeaveRequest struct {
	Reason    string // 请假原因
	StartDate string // 开始日期
	EndDate   string // 结束日期
}

// LeaveInfo 请假信息（collectInfo 输出）
type LeaveInfo struct {
	Reason    string // 请假原因
	StartDate string // 开始日期
	EndDate   string // 结束日期
	Days      int    // 请假天数
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid   bool   // 是否有效
	Message string // 验证消息
}

// CombinedValidation 合并的验证结果
type CombinedValidation struct {
	LeaveInfo          *LeaveInfo       // 请假信息（改为指针）
	DateValidation     *ValidationResult // 日期验证（改为指针）
	BalanceValidation  *ValidationResult // 余额验证（改为指针）
	ConflictValidation *ValidationResult // 冲突验证（改为指针）
	AllValid           bool              // 所有验证是否通过
}

// ApprovalResult 审批结果
type ApprovalResult struct {
	Approved bool   // 是否批准
	Comment  string // 审批意见
	Approver string // 审批人
}

// LeaveApprovalState 审批状态
type LeaveApprovalState struct {
	CombinedValidation CombinedValidation // 验证结果
	ManagerApproval    ApprovalResult     // 经理审批
}

// LeaveResponse 请假申请输出
type LeaveResponse struct {
	Success         bool            // 是否成功
	Message         string          // 消息
	LeaveInfo       *LeaveInfo      // 请假信息（改为指针）
	ManagerApproval ApprovalResult  // 经理审批
	HRApproval      ApprovalResult  // HR审批
}

// ==================== 节点函数 ====================

// collectInfoWorkflowNode 收集请假信息
func collectInfoWorkflowNode(ctx context.Context, req *LeaveRequest) (*LeaveInfo, error) {
	fmt.Println("\n📋 收集请假信息...")

	// 计算请假天数
	var days int
	if req.StartDate != "" && req.EndDate != "" {
		start, _ := time.Parse("2006-01-02", req.StartDate)
		end, _ := time.Parse("2006-01-02", req.EndDate)
		days = int(end.Sub(start).Hours()/24) + 1
	}

	info := &LeaveInfo{
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Days:      days,
	}

	fmt.Printf("  原因：%s\n", info.Reason)
	fmt.Printf("  日期：%s 至 %s\n", info.StartDate, info.EndDate)
	fmt.Printf("  天数：%d 天\n", info.Days)

	return info, nil
}

// validateDatesWorkflowNode 验证日期
func validateDatesWorkflowNode(ctx context.Context, info *LeaveInfo) (*ValidationResult, error) {
	fmt.Println("\n📅 验证日期...")

	result := &ValidationResult{Valid: true}

	// 验证日期格式
	start, err1 := time.Parse("2006-01-02", info.StartDate)
	end, err2 := time.Parse("2006-01-02", info.EndDate)

	if err1 != nil || err2 != nil {
		result.Valid = false
		result.Message = "日期格式错误"
		fmt.Println("  ✗ 日期格式错误")
		return result, nil
	}

	// 验证日期逻辑
	if end.Before(start) {
		result.Valid = false
		result.Message = "结束日期早于开始日期"
		fmt.Println("  ✗ 结束日期早于开始日期")
		return result, nil
	}

	result.Message = "日期验证通过"
	fmt.Println("  ✓ 日期验证通过")
	return result, nil
}

// checkBalanceWorkflowNode 检查假期余额
func checkBalanceWorkflowNode(ctx context.Context, info *LeaveInfo) (*ValidationResult, error) {
	fmt.Println("\n💰 检查假期余额...")

	result := &ValidationResult{Valid: true}

	// 模拟查询假期余额
	balances := map[string]int{
		"年假": 10,
		"病假": 5,
		"事假": 3,
	}

	available, ok := balances[info.Reason]
	if !ok {
		available = 0
	}

	fmt.Printf("  %s余额：%d 天\n", info.Reason, available)

	if info.Days > available {
		result.Valid = false
		result.Message = fmt.Sprintf("余额不足（需要%d天，剩余%d天）", info.Days, available)
		fmt.Printf("  ✗ 余额不足（需要%d天，剩余%d天）\n", info.Days, available)
		return result, nil
	}

	result.Message = "余额充足"
	fmt.Println("  ✓ 余额充足")
	return result, nil
}

// checkConflictWorkflowNode 检查日程冲突
func checkConflictWorkflowNode(ctx context.Context, info *LeaveInfo) (*ValidationResult, error) {
	fmt.Println("\n📆 检查日程冲突...")

	result := &ValidationResult{Valid: true}

	// 模拟检查日程冲突
	start, _ := time.Parse("2006-01-02", info.StartDate)
	end, _ := time.Parse("2006-01-02", info.EndDate)
	importantDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	if !importantDate.Before(start) && !importantDate.After(end) {
		result.Valid = true // 允许继续，但会在审批时提醒
		result.Message = "存在日程冲突：2024-06-15 有重要会议"
		fmt.Println("  ⚠️  存在日程冲突：2024-06-15 有重要会议")
		return result, nil
	}

	result.Message = "无日程冲突"
	fmt.Println("  ✓ 无日程冲突")
	return result, nil
}

// combineValidationWorkflowNode 合并验证结果
func combineValidationWorkflowNode(ctx context.Context, input *CombinedValidation) (*CombinedValidation, error) {
	fmt.Println("\n📝 生成申请摘要...")

	// 检查所有验证是否通过
	input.AllValid = input.DateValidation.Valid &&
		input.BalanceValidation.Valid &&
		input.ConflictValidation.Valid

	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("请假申请摘要")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("请假原因：%s\n", input.LeaveInfo.Reason)
	fmt.Printf("请假日期：%s 至 %s\n", input.LeaveInfo.StartDate, input.LeaveInfo.EndDate)
	fmt.Printf("请假天数：%d 天\n", input.LeaveInfo.Days)
	fmt.Println("\n验证结果：")
	fmt.Printf("  日期验证：%s - %s\n", boolToStatus(input.DateValidation.Valid), input.DateValidation.Message)
	fmt.Printf("  余额验证：%s - %s\n", boolToStatus(input.BalanceValidation.Valid), input.BalanceValidation.Message)
	fmt.Printf("  冲突检查：%s - %s\n", boolToStatus(input.ConflictValidation.Valid), input.ConflictValidation.Message)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")

	return input, nil
}

// managerApprovalWorkflowNode 经理审批
func managerApprovalWorkflowNode(ctx context.Context, validation *CombinedValidation) (*LeaveApprovalState, error) {
	fmt.Println("\n👔 等待经理审批...")

	state := &LeaveApprovalState{
		CombinedValidation: *validation,
	}

	// 模拟经理审批逻辑
	if !validation.AllValid {
		state.ManagerApproval = ApprovalResult{
			Approved: false,
			Comment:  "验证未通过，请修改后重新提交",
			Approver: "Manager",
		}
		fmt.Println("  ✗ 经理拒绝：验证未通过")
		return state, nil
	}

	state.ManagerApproval = ApprovalResult{
		Approved: true,
		Comment:  "同意请假申请",
		Approver: "Manager",
	}
	fmt.Println("  ✓ 经理批准")

	return state, nil
}

// hrApprovalWorkflowNode HR审批
func hrApprovalWorkflowNode(ctx context.Context, state *LeaveApprovalState) (*LeaveResponse, error) {
	fmt.Println("\n👥 等待HR审批...")

	response := &LeaveResponse{
		LeaveInfo:       state.CombinedValidation.LeaveInfo,
		ManagerApproval: state.ManagerApproval,
	}

	// 模拟HR审批逻辑
	if !state.ManagerApproval.Approved {
		response.Success = false
		response.Message = "经理未批准"
		response.HRApproval = ApprovalResult{
			Approved: false,
			Comment:  "经理未批准",
			Approver: "HR",
		}
		fmt.Println("  ✗ HR拒绝：经理未批准")
		return response, nil
	}

	response.Success = true
	response.Message = "请假申请已批准"
	response.HRApproval = ApprovalResult{
		Approved: true,
		Comment:  "HR审批通过",
		Approver: "HR",
	}
	fmt.Println("  ✓ HR批准")

	return response, nil
}

// submitLeaveWorkflowNode 提交请假
func submitLeaveWorkflowNode(ctx context.Context, response *LeaveResponse) (*LeaveResponse, error) {
	if response.Success {
		fmt.Println("\n✅ 请假申请提交成功！")
	} else {
		fmt.Println("\n❌ 请假申请被拒绝")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("最终审批结果")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📝 请假原因：%s\n", response.LeaveInfo.Reason)
	fmt.Printf("📅 请假日期：%s 至 %s\n", response.LeaveInfo.StartDate, response.LeaveInfo.EndDate)
	fmt.Printf("📆 请假天数：%d 天\n", response.LeaveInfo.Days)
	fmt.Println("\n审批记录：")
	fmt.Printf("  经理审批：%s - %s\n", boolToStatus(response.ManagerApproval.Approved), response.ManagerApproval.Comment)
	fmt.Printf("  HR审批：%s - %s\n", boolToStatus(response.HRApproval.Approved), response.HRApproval.Comment)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")

	return response, nil
}

// ==================== 辅助函数 ====================

func boolToStatus(b bool) string {
	if b {
		return "✓ 通过"
	}
	return "✗ 未通过"
}

// ==================== 构建 Workflow ====================

// buildLeaveWorkflow 构建请假工作流
func buildLeaveWorkflow() (compose.Runnable[*LeaveRequest, *LeaveResponse], error) {
	wf := compose.NewWorkflow[*LeaveRequest, *LeaveResponse]()

	// 节点1: 收集信息
	wf.AddLambdaNode("collectInfo", compose.InvokableLambda(collectInfoWorkflowNode)).
		AddInput(compose.START)

	// 节点2-4: 并行验证（通过 FieldMapping 实现）
	// 这三个节点都从 collectInfo 获取数据，但不相互依赖
	wf.AddLambdaNode("validateDates", compose.InvokableLambda(validateDatesWorkflowNode)).
		AddInput("collectInfo")

	wf.AddLambdaNode("checkBalance", compose.InvokableLambda(checkBalanceWorkflowNode)).
		AddInput("collectInfo")

	wf.AddLambdaNode("checkConflict", compose.InvokableLambda(checkConflictWorkflowNode)).
		AddInput("collectInfo")

	// 节点5: 合并验证结果
	// 使用 FieldMapping 从多个源组合数据
	wf.AddLambdaNode("combineValidation", compose.InvokableLambda(combineValidationWorkflowNode)).
		AddInput("collectInfo", compose.ToField("LeaveInfo")).
		AddInput("validateDates", compose.ToField("DateValidation")).
		AddInput("checkBalance", compose.ToField("BalanceValidation")).
		AddInput("checkConflict", compose.ToField("ConflictValidation"))

	// 节点6: 经理审批
	wf.AddLambdaNode("managerApproval", compose.InvokableLambda(managerApprovalWorkflowNode)).
		AddInput("combineValidation")

	// 节点7: HR审批
	wf.AddLambdaNode("hrApproval", compose.InvokableLambda(hrApprovalWorkflowNode)).
		AddInput("managerApproval")

	// 节点8: 提交请假
	wf.AddLambdaNode("submitLeave", compose.InvokableLambda(submitLeaveWorkflowNode)).
		AddInput("hrApproval")

	// 连接到 END
	wf.End().AddInput("submitLeave")

	// 编译 Workflow
	runnable, err := wf.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("compile workflow failed: %w", err)
	}

	return runnable, nil
}

// ==================== 测试用例 ====================

// TestLeaveWorkflow_ApprovedFlow 测试审批通过的流程
func TestLeaveWorkflow_ApprovedFlow(t *testing.T) {
	ctx := context.Background()

	workflow, err := buildLeaveWorkflow()
	if err != nil {
		t.Fatalf("构建Workflow失败: %v", err)
	}

	t.Log("=== 测试场景：审批通过 ===")

	// 初始状态（所有条件都满足）
	request := &LeaveRequest{
		Reason:    "年假",
		StartDate: "2024-06-01",
		EndDate:   "2024-06-05",
	}

	// 执行 Workflow
	result, err := workflow.Invoke(ctx, request)
	if err != nil {
		t.Fatalf("执行Workflow失败: %v", err)
	}

	// 验证结果
	if !result.Success {
		t.Errorf("期望 Success=true，实际: %v", result.Success)
	}

	if !result.ManagerApproval.Approved {
		t.Error("期望经理批准")
	}

	if !result.HRApproval.Approved {
		t.Error("期望HR批准")
	}

	t.Log("✓ 审批通过流程测试通过")
}

// TestLeaveWorkflow_RejectedByValidation 测试验证失败的流程
func TestLeaveWorkflow_RejectedByValidation(t *testing.T) {
	ctx := context.Background()

	workflow, err := buildLeaveWorkflow()
	if err != nil {
		t.Fatalf("构建Workflow失败: %v", err)
	}

	t.Log("=== 测试场景：验证失败 ===")

	// 初始状态（日期格式错误）
	request := &LeaveRequest{
		Reason:    "年假",
		StartDate: "2024-06-10",
		EndDate:   "2024-06-05", // 结束日期早于开始日期
	}

	// 执行 Workflow
	result, err := workflow.Invoke(ctx, request)
	if err != nil {
		t.Fatalf("执行Workflow失败: %v", err)
	}

	// 验证结果
	if result.Success {
		t.Errorf("期望 Success=false，实际: %v", result.Success)
	}

	if result.ManagerApproval.Approved {
		t.Error("期望经理拒绝")
	}

	t.Log("✓ 验证失败流程测试通过")
}

// TestLeaveWorkflow_InsufficientBalance 测试余额不足的流程
func TestLeaveWorkflow_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	workflow, err := buildLeaveWorkflow()
	if err != nil {
		t.Fatalf("构建Workflow失败: %v", err)
	}

	t.Log("=== 测试场景：余额不足 ===")

	// 初始状态（请假天数超过余额）
	request := &LeaveRequest{
		Reason:    "年假",
		StartDate: "2024-06-01",
		EndDate:   "2024-06-20", // 20天，超过余额
	}

	// 执行 Workflow
	result, err := workflow.Invoke(ctx, request)
	if err != nil {
		t.Fatalf("执行Workflow失败: %v", err)
	}

	// 验证结果
	if result.Success {
		t.Errorf("期望 Success=false，实际: %v", result.Success)
	}

	if result.ManagerApproval.Approved {
		t.Error("期望经理拒绝")
	}

	t.Log("✓ 余额不足流程测试通过")
}

// TestLeaveWorkflow_CompleteFlow 测试完整的工作流
func TestLeaveWorkflow_CompleteFlow(t *testing.T) {
	ctx := context.Background()

	workflow, err := buildLeaveWorkflow()
	if err != nil {
		t.Fatalf("构建Workflow失败: %v", err)
	}

	t.Log("=== 测试场景：完整的工作流 ===")

	testCases := []struct {
		name          string
		request       *LeaveRequest
		expectSuccess bool
	}{
		{
			name: "正常年假申请",
			request: &LeaveRequest{
				Reason:    "年假",
				StartDate: "2024-06-01",
				EndDate:   "2024-06-05",
			},
			expectSuccess: true,
		},
		{
			name: "病假申请",
			request: &LeaveRequest{
				Reason:    "病假",
				StartDate: "2024-06-10",
				EndDate:   "2024-06-12",
			},
			expectSuccess: true,
		},
		{
			name: "日期错误",
			request: &LeaveRequest{
				Reason:    "事假",
				StartDate: "2024-06-20",
				EndDate:   "2024-06-15",
			},
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := workflow.Invoke(ctx, tc.request)
			if err != nil {
				t.Fatalf("执行Workflow失败: %v", err)
			}

			if result.Success != tc.expectSuccess {
				t.Errorf("期望 Success=%v，实际: %v", tc.expectSuccess, result.Success)
			}

			t.Logf("✓ %s 测试通过（状态：%v）", tc.name, result.Success)
		})
	}

	t.Log("\n✓ 完整工作流测试通过")
}

// ==================== 最佳实践总结 ====================

/*
## Workflow 最佳实践

### 1. 何时使用 Workflow

✓ 适合使用的场景：
- 需要多级审批的流程
- 需要并行验证的流程
- 需要精细的字段级数据控制
- 需要组合多个数据源

✗ 不适合使用的场景：
- 简单的线性流程（用 Chain）
- 只需要条件分支的流程（用 Graph）

### 2. FieldMapping 使用原则

**ToField**：将整个输出映射到目标字段
```go
wf.AddInput("node1", compose.ToField("FieldName"))
```

**MapFields**：映射特定字段
```go
wf.AddInput("node1", compose.MapFields("SourceField", "TargetField"))
```

**组合多个数据源**：
```go
wf.AddLambdaNode("combine", ...).
    AddInput("node1", compose.ToField("Field1")).
    AddInput("node2", compose.ToField("Field2")).
    AddInput("node3", compose.ToField("Field3"))
```

### 3. 并行任务实现

通过 FieldMapping 实现并行：
```go
// 三个节点都从同一个源获取数据，但不相互依赖
wf.AddLambdaNode("validate1", ...).AddInput("source")
wf.AddLambdaNode("validate2", ...).AddInput("source")
wf.AddLambdaNode("validate3", ...).AddInput("source")

// 合并结果
wf.AddLambdaNode("combine", ...).
    AddInput("validate1", compose.ToField("Result1")).
    AddInput("validate2", compose.ToField("Result2")).
    AddInput("validate3", compose.ToField("Result3"))
```

### 4. 数据结构设计

**输入输出类型化**：
- 每个节点都有明确的输入输出类型
- 使用结构体而不是 map[string]any
- 便于类型检查和IDE提示

**组合结构**：
- 使用组合而不是继承
- 每个阶段有独立的数据结构
- 通过 FieldMapping 组合数据

### 5. 错误处理

**节点级错误处理**：
- 每个节点返回 error
- 验证失败通过返回值表示，不抛出错误
- 只有系统级错误才返回 error

**流程级错误处理**：
- Workflow 会在任何节点返回 error 时停止
- 业务逻辑失败（如审批拒绝）不应该返回 error
- 使用状态字段（如 Success）表示业务结果

### 6. 测试策略

**场景测试**：
- 测试正常审批通过的流程
- 测试各种拒绝场景
- 测试边界条件

**数据流测试**：
- 验证 FieldMapping 正确传递数据
- 验证并行节点独立执行
- 验证数据组合正确

## Chain vs Graph vs Workflow 对比

| 特性 | Chain | Graph | Workflow |
|------|-------|-------|----------|
| 复杂度 | 低 | 中 | 高 |
| 条件分支 | ✗ | ✓ | ✓ |
| 循环重试 | ✗ | ✓ | ✓ |
| 并行执行 | ✗ | 有限 | ✓ |
| 字段映射 | ✗ | ✗ | ✓ |
| 多级审批 | ✗ | 有限 | ✓ |
| 数据控制 | 全量 | 全量 | 精细 |
| 适用场景 | 简单流程 | 中等流程 | 复杂流程 |

## 实际应用建议

1. **从简单开始**：先用 Chain，需要分支时升级到 Graph，需要复杂数据流时升级到 Workflow
2. **合理使用 FieldMapping**：避免不必要的数据传递，提高性能
3. **状态管理**：设计清晰的数据结构，每个阶段有独立的类型
4. **错误处理**：区分业务错误和系统错误
5. **测试覆盖**：编写完整的测试用例，覆盖各种场景
*/
