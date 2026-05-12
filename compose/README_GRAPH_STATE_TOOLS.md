# Graph with State and Tools 示例

这个示例展示了如何在 Eino Graph 中使用状态管理（State）和工具调用（Tools）功能。

## 核心概念

### 1. State Management（状态管理）

通过 `WithGenLocalState` 为 Graph 添加全局状态：

```go
genState := func(ctx context.Context) *AgentState {
    return &AgentState{
        Messages:      []*schema.Message{},
        ToolCallCount: 0,
    }
}

graph := compose.NewGraph[*AgentInput, *AgentOutput](
    compose.WithGenLocalState(genState),
)
```

### 2. State Handlers（状态处理器）

- **StatePreHandler**: 节点执行前处理状态
- **StatePostHandler**: 节点执行后处理状态

```go
graph.AddLambdaNode("chatModel",
    compose.InvokableLambda(chatModelNode),
    compose.WithStatePreHandler(statePreHandler),
    compose.WithStatePostHandler(statePostHandler),
)
```

### 3. ProcessState（状态访问）

在节点内部访问和修改状态：

```go
err := compose.ProcessState[*AgentState](ctx, func(_ context.Context, state *AgentState) error {
    state.ToolCallCount++
    state.Messages = append(state.Messages, msg)
    return nil
})
```

### 4. Tool Integration（工具集成）

定义工具并在 Graph 中执行：

```go
// 定义工具
tool := &schema.ToolInfo{
    Name: "get_user_info",
    Desc: "获取用户信息",
    ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
        "user_id": {Type: schema.String, Required: true},
    }),
}

// 执行工具
result, err := mockToolExecutor(ctx, toolName, args)
```

## 场景说明

这个示例实现了一个智能助手，可以：

1. **理解用户查询**：ChatModel 分析用户输入
2. **决策工具调用**：判断是否需要调用工具
3. **执行工具**：在 ToolsNode 中执行工具获取信息
4. **生成回答**：基于工具结果生成最终回答
5. **状态跟踪**：记录对话历史和工具调用次数

## Graph 结构

```
START
  ↓
chatModel (AI 决策)
  ├─ need_tool → toolsNode (执行工具)
  │                ↓
  │              finalResponse (生成最终回答)
  │                ↓
  │              END
  └─ no_tool → directResponse (直接返回)
                 ↓
               END
```

## 运行测试

```bash
# 运行所有测试
go test -v -run TestGraphWithStateAndTools

# 运行特定测试
go test -v -run TestGraphWithStateAndTools_WithTools
go test -v -run TestGraphWithStateAndTools_NoTools
```

## 测试场景

### 场景 1：需要工具调用

**输入**：
```
请帮我查询用户12345的信息，以及北京的天气
```

**执行流程**：
1. ChatModel 识别需要调用 `get_user_info` 和 `get_weather` 工具
2. ToolsNode 执行两个工具调用
3. State 记录工具调用次数（2次）
4. ChatModel 基于工具结果生成最终回答

**输出**：
```
根据查询结果，用户张三今年28岁，住在北京。北京今天天气晴朗，温度22°C。
```

### 场景 2：不需要工具调用

**输入**：
```
你好，请介绍一下你自己
```

**执行流程**：
1. ChatModel 判断无需工具调用
2. 直接返回回答
3. State 记录工具调用次数（0次）

**输出**：
```
我是一个智能助手，可以帮你查询用户信息和天气。请问有什么可以帮助你的？
```

## 关键代码说明

### AgentState 结构

```go
type AgentState struct {
    Messages       []*schema.Message // 对话历史
    ToolCallCount  int               // 工具调用次数
    LastToolResult string            // 最后一次工具调用结果
}
```

### 条件路由

```go
func routeAfterChatModel(ctx context.Context, msg *schema.Message) (string, error) {
    if len(msg.ToolCalls) > 0 {
        return "toolsNode", nil  // 需要调用工具
    }
    return "directResponse", nil  // 直接返回
}
```

## 最佳实践

### 1. State 设计

- ✅ 只保存必要的状态信息
- ✅ 使用明确的字段名
- ✅ 避免嵌套过深
- ❌ 不要在状态中保存大对象

### 2. Tool Integration

- ✅ 提供清晰的工具描述
- ✅ 定义必需和可选参数
- ✅ 处理工具执行错误
- ✅ 记录工具调用历史

### 3. 节点设计

- ✅ 每个节点职责单一
- ✅ 使用 ProcessState 访问状态
- ✅ 在 Handler 中更新状态
- ❌ 避免在多个地方修改同一状态

### 4. 错误处理

- ✅ 节点内部捕获可恢复的错误
- ✅ 关键错误应该中断流程
- ✅ 提供清晰的错误信息

## 扩展思考

### 实际应用场景

1. **对话 Agent**
   - 维护多轮对话历史
   - 动态调用外部 API
   - 上下文感知回答

2. **任务执行器**
   - 跟踪任务执行状态
   - 记录每个步骤
   - 支持错误恢复

3. **工作流引擎**
   - 实现状态机
   - 审批流程
   - 事件驱动架构

### 高级特性

- **多轮对话**：在 State 中维护完整对话历史
- **工具链**：一个工具的输出作为另一个工具的输入
- **并行工具调用**：同时执行多个独立的工具
- **条件工具选择**：根据上下文动态选择工具
- **工具结果缓存**：避免重复调用相同工具

## 参考资料

- [Eino 官方文档 - Graph with State](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/chain_graph_introduction/#graph-with-state)
- [graph_test.go](./graph_test.go) - 完整示例代码
- [workflow_test.go](./workflow_test.go) - Workflow 示例
