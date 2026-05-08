# Eino Graph Tool 测试示例

## 概述

本示例演示了如何使用 Eino 的 `compose.Workflow` 构建复杂的工作流，实现文档问答 RAG 系统。

## 核心概念

### 1. Workflow（工作流）
- 使用 `compose.NewWorkflow` 创建工作流
- 通过 `AddLambdaNode` 添加处理节点
- 使用 `AddInput` 连接节点，形成数据流

### 2. FieldMapping（字段映射）
- 允许跨节点传递数据，无需通过中间节点
- 使用 `AddInputWithOptions` + `FieldMapping` 实现
- 使用 `WithNoDirectDependency` 避免循环依赖

### 3. 工作流节点
本示例包含 5 个处理节点：
1. **load** - 读取文档文件
2. **chunk** - 将文档分块（800字符/块）
3. **score** - 评分每个块与问题的相关性（0-10分）
4. **filter** - 筛选 top-3 相关块
5. **answer** - 基于相关块生成答案

## 执行流程

```
START (FilePath, Question)
  │
  ├─ Question ────────────────────────────┐
  │                                       │
  ▼                                       │
[load] 读取文件 → []*Document            │
  │                                       │
  ▼                                       │
[chunk] 分块 → []*Document                │
  │                                       │
  ├─ Chunks ──────────► [score] ◄────────┘ Question
  │                       │
  │                       ▼ []ScoredChunk
  │                    [filter] 筛选 top-k
  │                       │
  │                       ├─ TopK ──────► [answer] ◄─── Question
  │                                         │
  │                                         ▼
  └─────────────────────────────────────► END
```

## 运行测试

```bash
cd ai_agent_demo/graph_tool
go test -v
```

## 测试用例

### 1. TestDocumentQAWorkflow
测试完整的文档问答工作流：
- 完整工作流测试：验证整个流程能正常执行
- FieldMapping测试：验证跨节点数据传递
- 无相关内容测试：验证低分块的过滤逻辑

### 2. TestChunkSplitting
测试文本分块功能：
- 验证分块策略（段落边界优先）
- 验证块大小限制

### 3. TestWorkflowStructure
测试工作流结构：
- 验证工作流可以成功编译
- 验证节点连接的正确性

## 关键代码示例

### 创建工作流

```go
wf := compose.NewWorkflow[DocumentQAInput, DocumentQAOutput]()

// 添加节点
wf.AddLambdaNode("load", compose.InvokableLambda(
    func(ctx context.Context, in DocumentQAInput) ([]*schema.Document, error) {
        // 处理逻辑
        return documents, nil
    },
)).AddInput(compose.START)
```

### 使用 FieldMapping

```go
// score 节点需要两个输入：
// 1. Chunks 来自 chunk 节点
// 2. Question 来自 START 节点
wf.AddLambdaNode("score", compose.InvokableLambda(
    func(ctx context.Context, in ScoreInput) ([]ScoredChunk, error) {
        // 处理逻辑
        return scored, nil
    },
)).
    AddInputWithOptions("chunk",
        []*compose.FieldMapping{compose.ToField("Chunks")},
        compose.WithNoDirectDependency()).
    AddInputWithOptions(compose.START,
        []*compose.FieldMapping{compose.MapFields("Question", "Question")},
        compose.WithNoDirectDependency())
```

### 编译和执行

```go
// 编译工作流
runnable, err := wf.Compile(ctx)
if err != nil {
    return err
}

// 执行工作流
output, err := runnable.Invoke(ctx, input)
```

## 最佳实践

### 1. 节点设计
- **单一职责**：每个节点只做一件事
- **类型安全**：明确定义输入输出类型
- **错误处理**：在节点内部处理可恢复的错误

### 2. FieldMapping 使用
- **避免冗余传递**：不要在中间节点传递不需要的数据
- **使用 WithNoDirectDependency**：当从非直接前驱获取数据时
- **清晰的数据流**：使用结构体封装复杂输入

### 3. 性能优化
- **合理分块**：根据模型上下文窗口调整块大小
- **并发处理**：对独立任务使用 goroutine
- **错误降级**：评分失败时返回 0 分而不是中断流程

## 真实场景应用

### 与 Agent 集成

在真实应用中，可以将工作流封装为 Tool：

```go
// 1. 构建工作流
wf := buildDocumentQAWorkflow(cm)

// 2. 封装为 Tool（需要 eino-examples/adk/common/tool/graphtool）
tool, err := graphtool.NewInvokableGraphTool[Input, Output](
    wf,
    "answer_from_document",
    "从文档中搜索相关内容并生成答案",
)

// 3. 注册到 Agent
agent, err := deep.New(ctx, &deep.Config{
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{tool},
        },
    },
})
```

### 支持中断恢复

工作流可以与 CheckPointStore 结合，支持中断和恢复：
- 在关键节点保存状态
- 用户确认后恢复执行
- 参考 `interrupt_resume` 示例

## 扩展思考

### 其他应用场景
- **多文档 RAG**：并行处理多个文档
- **数据处理流水线**：ETL、数据清洗、特征提取
- **多步骤决策**：根据条件选择不同分支
- **复杂业务流程**：订单处理、审批流程

### 高级特性
- **条件分支**：根据节点输出选择不同路径
- **循环节点**：重复执行直到满足条件
- **子图嵌套**：将子工作流作为节点
- **流式输出**：支持流式返回中间结果

## 参考资料

- [Eino 官方文档](https://github.com/cloudwego/eino)
- [RAG 示例](../../eino-examples/quickstart/chatwitheino/rag/rag.go)
- [Callback 示例](../callback/callback_test.go)
- [Interrupt/Resume 示例](../interrupt_resume/interrupt_resume_test.go)

## 注意事项

1. **Mock 模型限制**：本示例使用 mock 模型，所有问题返回相同答案。真实场景需要使用真实的 ChatModel。

2. **依赖管理**：本示例只使用 Eino 核心库，不依赖 `eino-examples`。如需使用 `graphtool.NewInvokableGraphTool`，需要引入相应依赖。

3. **性能考虑**：真实场景中，评分步骤可以使用 BatchNode 并行处理，提升性能。

4. **错误处理**：生产环境需要更完善的错误处理和日志记录。
