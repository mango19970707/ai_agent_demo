# Eino Compose 示例 - 请假申请流程

本目录包含基于请假场景的 Eino Compose 示例代码，展示了如何使用 Chain、Graph 和 Workflow 来构建不同复杂度的业务流程。

## 📁 文件说明

| 文件 | 说明 | 复杂度 |
|------|------|--------|
| `chain.go` | 使用 Chain 实现的简单线性流程 | ⭐ 低 |
| `graph_test.go` | 使用 Graph 实现的条件分支流程 | ⭐⭐ 中 |
| `workflow_test.go` | 使用 Workflow 实现的多级审批流程 | ⭐⭐⭐ 高 |

## 🚀 快速开始

### 运行 Chain 示例（交互式）

```bash
cd D:/code/eino-study/ai_agent_demo/compose
go run chain.go
```

**交互示例：**
```
🤖 欢迎使用 Go-Eino 请假系统（输入 exit 退出）
🤖 请问你请假的原因是什么？（如：生病、事假、年假）
你：年假
🤖 请输入请假开始日期（格式：YYYY-MM-DD）
你：2024-06-01
🤖 请输入请假结束日期（格式：YYYY-MM-DD）
你：2024-06-05

✅ 请假申请提交成功！
━━━━━━━━━━━━━━━━━━━━
📝 请假原因：年假
📅 开始日期：2024-06-01
📅 结束日期：2024-06-05
📆 请假天数：5 天
━━━━━━━━━━━━━━━━━━━━
```

### 运行 Graph 测试

```bash
# 运行所有 Graph 测试
go test -v -run TestLeaveGraph

# 运行特定测试
go test -v -run TestLeaveGraph_BasicFlow
go test -v -run TestLeaveGraph_InvalidStartDate
```

### 运行 Workflow 测试

```bash
# 运行所有 Workflow 测试
go test -v -run TestLeaveWorkflow

# 运行特定测试
go test -v -run TestLeaveWorkflow_ApprovedFlow
go test -v -run TestLeaveWorkflow_RejectedByValidation
```

## 📊 三种方式对比

### Chain（链）

**特点：**
- ✅ 简单直观，按顺序执行
- ✅ 适合线性流程
- ❌ 不支持条件分支
- ❌ 不支持循环重试

**使用场景：**
- 简单的顺序处理流程
- 数据转换管道
- 无需条件判断的任务

**代码示例：**
```go
chain := compose.NewChain[*LeaveState, *LeaveState]()
chain.AppendLambda(compose.InvokableLambda(askReasonStep))
chain.AppendLambda(compose.InvokableLambda(askStartDateStep))
chain.AppendLambda(compose.InvokableLambda(askEndDateStep))
chain.AppendLambda(compose.InvokableLambda(submitLeaveStep))
runnable, _ := chain.Compile(ctx)
```

**流程图：**
```
START → askReason → askStartDate → askEndDate → submitLeave → END
```

---

### Graph（图）

**特点：**
- ✅ 支持条件分支
- ✅ 支持循环重试
- ✅ 灵活的路由控制
- ⚠️ 需要注意最大步数限制

**使用场景：**
- 需要条件判断的流程
- 需要验证和重试的流程
- 复杂的业务逻辑

**代码示例：**
```go
graph := compose.NewGraph[*LeaveGraphState, *LeaveGraphState]()

// 添加节点
graph.AddLambdaNode("validateStartDate", compose.InvokableLambda(validateStartDateNode))

// 添加分支（条件路由）
graph.AddBranch("validateStartDate", compose.NewGraphBranch(
    routeAfterStartDateValidation,
    map[string]bool{
        "askStartDate": true,  // 验证失败，重新询问
        "askEndDate":   true,  // 验证成功，继续下一步
    },
))
```

**流程图：**
```
START
  ↓
askReason
  ↓
askStartDate
  ↓
validateStartDate
  ├─ valid → askEndDate
  └─ invalid → askStartDate (重新询问)
  ↓
askEndDate
  ↓
validateEndDate
  ├─ valid → submitLeave
  └─ invalid → askEndDate (重新询问)
  ↓
submitLeave
  ↓
END
```

**重要提示：**

Graph 在单次 `Invoke()` 调用中会自动循环执行，如果数据一直错误会超过最大步数限制。

**正确的使用方式：**
```go
// ❌ 错误：数据一直错误会导致无限循环
state := &LeaveGraphState{StartDate: "invalid-date"}
result, err := graph.Invoke(ctx, state)  // 会超过最大步数

// ✅ 正确：在外部循环中修正数据
state := &LeaveGraphState{StartDate: "invalid-date"}
_, err := graph.Invoke(ctx, state)
if err != nil {
    // 修正数据
    state.StartDate = "2024-06-01"
    state.CurrentNode = ""  // 重置状态
    result, err := graph.Invoke(ctx, state)  // 重新执行
}
```

---

### Workflow（工作流）

**特点：**
- ✅ 支持多级审批
- ✅ 支持并行验证（通过 FieldMapping）
- ✅ 支持字段级数据映射
- ✅ 精细的数据流控制
- ✅ 适合复杂业务流程

**使用场景：**
- 多级审批流程
- 需要并行验证的流程
- 需要组合多个数据源
- 需要精细控制数据传递

**代码示例：**
```go
wf := compose.NewWorkflow[*LeaveRequest, *LeaveResponse]()

// 节点1: 收集信息
wf.AddLambdaNode("collectInfo", compose.InvokableLambda(collectInfoNode)).
    AddInput(compose.START)

// 节点2-4: 并行验证（通过 FieldMapping 实现）
wf.AddLambdaNode("validateDates", compose.InvokableLambda(validateDatesNode)).
    AddInput("collectInfo")

wf.AddLambdaNode("checkBalance", compose.InvokableLambda(checkBalanceNode)).
    AddInput("collectInfo")

wf.AddLambdaNode("checkConflict", compose.InvokableLambda(checkConflictNode)).
    AddInput("collectInfo")

// 节点5: 合并验证结果（使用 FieldMapping 组合数据）
wf.AddLambdaNode("combineValidation", compose.InvokableLambda(combineValidationNode)).
    AddInput("collectInfo", compose.ToField("LeaveInfo")).
    AddInput("validateDates", compose.ToField("DateValidation")).
    AddInput("checkBalance", compose.ToField("BalanceValidation")).
    AddInput("checkConflict", compose.ToField("ConflictValidation"))

// 节点6-8: 审批流程
wf.AddLambdaNode("managerApproval", compose.InvokableLambda(managerApprovalNode)).
    AddInput("combineValidation")

wf.AddLambdaNode("hrApproval", compose.InvokableLambda(hrApprovalNode)).
    AddInput("managerApproval")

wf.AddLambdaNode("submitLeave", compose.InvokableLambda(submitLeaveNode)).
    AddInput("hrApproval")

wf.End().AddInput("submitLeave")
```

**流程图：**
```
START
  ↓
collectInfo (收集信息)
  ↓
[并行验证 - 通过 FieldMapping 实现]
  ├─ validateDates (验证日期) ← 从 collectInfo 获取
  ├─ checkBalance (检查余额) ← 从 collectInfo 获取
  └─ checkConflict (检查冲突) ← 从 collectInfo 获取
  ↓
combineValidation (合并验证结果)
  ← 使用 FieldMapping 组合多个数据源
  ↓
managerApproval (经理审批)
  ↓
hrApproval (HR审批)
  ↓
submitLeave (提交请假)
  ↓
END
```

**关键特性：FieldMapping**

Workflow 的核心优势是支持字段级数据映射：

```go
// ToField: 将整个输出映射到目标字段
wf.AddInput("node1", compose.ToField("FieldName"))

// MapFields: 映射特定字段
wf.AddInput("node1", compose.MapFields("SourceField", "TargetField"))

// 组合多个数据源
wf.AddLambdaNode("combine", ...).
    AddInput("node1", compose.ToField("Field1")).
    AddInput("node2", compose.ToField("Field2")).
    AddInput("node3", compose.ToField("Field3"))
```

---

## 🎯 选择指南

### 何时使用 Chain？

- ✅ 简单的顺序流程
- ✅ 不需要条件判断
- ✅ 不需要循环重试
- ✅ 快速原型开发

**示例：**
- 数据转换管道
- 简单的消息处理
- 日志记录流程

### 何时使用 Graph？

- ✅ 需要条件分支
- ✅ 需要验证和重试
- ✅ 需要根据状态选择路径
- ✅ 中等复杂度的业务流程

**示例：**
- 表单验证流程
- 订单处理流程
- 用户注册流程

### 何时使用 Workflow？

- ✅ 多级审批流程
- ✅ 需要并行验证
- ✅ 复杂的业务流程
- ✅ 需要人工介入

**示例：**
- 请假审批系统
- 采购审批流程
- 合同审批流程
- 报销审批系统

---

## 📝 测试用例说明

### Graph 测试用例

| 测试 | 说明 | 验证点 |
|------|------|--------|
| `TestLeaveGraph_BasicFlow` | 基本流程（所有输入正确） | 完整流程执行 |
| `TestLeaveGraph_InvalidStartDate` | 开始日期格式错误 | 循环重试机制 |
| `TestLeaveGraph_InvalidEndDate` | 结束日期早于开始日期 | 日期逻辑验证 |
| `TestLeaveGraph_MultipleRetries` | 多次重试 | 外部循环控制 |
| `TestLeaveGraph_CompleteFlow` | 完整流程 | 多种场景覆盖 |

### Workflow 测试用例

| 测试 | 说明 | 验证点 |
|------|------|--------|
| `TestLeaveWorkflow_ApprovedFlow` | 审批通过流程 | 多级审批 |
| `TestLeaveWorkflow_RejectedByValidation` | 验证失败 | 验证逻辑 |
| `TestLeaveWorkflow_InsufficientBalance` | 余额不足 | 余额检查 |
| `TestLeaveWorkflow_CompleteFlow` | 完整工作流 | 多种场景 |

---

## 🔧 常见问题

### Q1: Graph 执行时超过最大步数怎么办？

**原因：** Graph 在单次 `Invoke()` 中会循环执行，如果验证一直失败会超过最大步数限制（默认10步）。

**解决方案：**

1. **增加最大步数：**
```go
runnable, err := graph.Compile(ctx, compose.WithMaxRunSteps(50))
```

2. **在外部循环中控制重试：**
```go
for {
    result, err := graph.Invoke(ctx, state)
    if err != nil {
        // 修正数据
        state = fixData(state)
        continue
    }
    break
}
```

### Q2: 如何实现并行验证？

在 Workflow 中，通过 FieldMapping 实现并行验证：

```go
// 三个验证节点都从同一个源获取数据，但不相互依赖
wf.AddLambdaNode("validateDates", ...).AddInput("collectInfo")
wf.AddLambdaNode("checkBalance", ...).AddInput("collectInfo")
wf.AddLambdaNode("checkConflict", ...).AddInput("collectInfo")

// 合并验证结果（使用 FieldMapping 组合数据）
wf.AddLambdaNode("combineValidation", ...).
    AddInput("collectInfo", compose.ToField("LeaveInfo")).
    AddInput("validateDates", compose.ToField("DateValidation")).
    AddInput("checkBalance", compose.ToField("BalanceValidation")).
    AddInput("checkConflict", compose.ToField("ConflictValidation"))
```

这样三个验证节点可以并行执行，因为它们：
1. 都从 `collectInfo` 获取数据
2. 彼此之间没有依赖关系
3. 结果在 `combineValidation` 节点中组合

### Q3: 如何在节点之间传递数据？

使用状态对象（State）在节点之间传递数据：

```go
type LeaveState struct {
    Reason    string  // 请假原因
    StartDate string  // 开始日期
    EndDate   string  // 结束日期
    ErrorMsg  string  // 错误信息
}

func node1(ctx context.Context, state *LeaveState) (*LeaveState, error) {
    state.Reason = "年假"
    return state, nil
}

func node2(ctx context.Context, state *LeaveState) (*LeaveState, error) {
    // 可以访问 node1 设置的 Reason
    fmt.Println(state.Reason)
    return state, nil
}
```

---

## 📚 相关资源

- [Eino 官方文档](https://github.com/cloudwego/eino)
- [Eino Examples](https://github.com/cloudwego/eino-examples)
- [Compose 包文档](https://pkg.go.dev/github.com/cloudwego/eino/compose)

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

## 📄 许可证

本示例代码遵循 Apache 2.0 许可证。
