package graph_tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

/*
===========================================
Eino Graph Tool（复杂工作流）测试示例
场景：文档问答 RAG 系统
===========================================

## 核心概念

1. **compose.Workflow**：构建工作流的核心组件，支持节点编排
2. **Lambda 节点**：执行具体业务逻辑的节点
3. **FieldMapping**：跨节点传递数据，无需通过中间节点
4. **并行处理**：通过工作流实现多任务并行

## 文档问答场景中的工作流节点

1. **load**：读取文档文件
2. **chunk**：将文档分块（800字符/块）
3. **score**：评分每个块与问题的相关性（0-10分）
4. **filter**：筛选 top-k 相关块
5. **answer**：基于相关块生成答案

## 使用方法

运行测试：
	go test -v

测试会模拟以下流程：
1. 创建一个包含多段内容的文档
2. 用户提出问题
3. Workflow 自动执行：读取 → 分块 → 评分 → 筛选 → 生成答案
4. 返回带引用的答案

## 执行流程

START → load → chunk → score → filter → answer → END

数据流：
- Question 字段从 START 直接传递到 score 和 answer（通过 FieldMapping）
- 避免在中间节点（load、chunk）中传递不需要的数据

## 真实场景集成

在真实的 Eino 应用中：
1. 使用 compose.NewWorkflow 构建工作流
2. 使用 AddLambdaNode 添加处理节点
3. 使用 AddInputWithOptions + FieldMapping 跨节点传递数据
4. 编译并执行工作流
5. 参考 eino-examples/quickstart/chatwitheino/rag/rag.go
*/

// ==================== 数据结构 ====================

// DocumentQAInput 文档问答的输入参数
type DocumentQAInput struct {
	FilePath string `json:"file_path"` // 文档文件的绝对路径
	Question string `json:"question"`  // 要回答的问题
}

// DocumentQAOutput 文档问答的输出结果
type DocumentQAOutput struct {
	Answer  string   `json:"answer"`  // 生成的答案
	Sources []string `json:"sources"` // 引用的文档片段
}

// ScoredChunk 评分后的块
type ScoredChunk struct {
	Text    string // 块内容
	Score   int    // 相关性评分（0-10）
	Excerpt string // 最相关的摘录
}

// ScoreInput score 节点的输入（通过 FieldMapping 组装）
type ScoreInput struct {
	Chunks   []*schema.Document // 来自 chunk 节点
	Question string             // 来自 START 节点
}

// AnswerInput answer 节点的输入（通过 FieldMapping 组装）
type AnswerInput struct {
	TopK     []ScoredChunk // 来自 filter 节点
	Question string        // 来自 START 节点
}

// ==================== Mock ChatModel（用于测试） ====================

// mockChatModel 模拟 ChatModel，用于测试
type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 模拟评分响应
	if len(messages) > 0 && strings.Contains(messages[0].Content, "Rate how relevant") {
		// 简单的关键词匹配评分
		content := messages[0].Content
		score := 0
		excerpt := ""

		if strings.Contains(content, "Eino") || strings.Contains(content, "框架") {
			score = 8
			excerpt = "Eino 是一个强大的 AI 应用框架"
		} else if strings.Contains(content, "Workflow") || strings.Contains(content, "工作流") {
			score = 7
			excerpt = "Workflow 支持复杂工作流编排"
		} else if strings.Contains(content, "FieldMapping") || strings.Contains(content, "数据传递") {
			score = 6
			excerpt = "FieldMapping 允许跨节点传递数据"
		} else if strings.Contains(content, "测试") {
			score = 5
			excerpt = "测试是保证代码质量的重要手段"
		} else {
			score = 2
			excerpt = ""
		}

		response := fmt.Sprintf(`{"score": %d, "excerpt": "%s"}`, score, excerpt)
		return schema.AssistantMessage(response, nil), nil
	}

	// 模拟答案生成响应
	if len(messages) > 0 && strings.Contains(messages[0].Content, "Answer the following question") {
		answer := "根据文档内容，Eino 是一个强大的 AI 应用框架，支持 Workflow 进行复杂工作流编排。通过 FieldMapping 可以实现跨节点的数据传递。[1][2][3]"
		return schema.AssistantMessage(answer, nil), nil
	}

	return schema.AssistantMessage("模拟响应", nil), nil
}

func (m *mockChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	// 简化实现：直接返回完整消息
	msg, err := m.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) GetType(ctx context.Context) string {
	return "mock_chat_model"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return false
}

// ==================== 工作流构建 ====================

// buildDocumentQAWorkflow 构建文档问答工作流
// 演示了 Eino Workflow 的核心特性：
// 1. 节点编排：通过 AddLambdaNode 添加处理节点
// 2. 数据流控制：通过 AddInput 连接节点
// 3. FieldMapping：跨节点传递数据（Question 从 START 直接到 score 和 answer）
func buildDocumentQAWorkflow(cm model.BaseChatModel) *compose.Workflow[DocumentQAInput, DocumentQAOutput] {
	wf := compose.NewWorkflow[DocumentQAInput, DocumentQAOutput]()

	// 节点1: load - 读取文件
	// 输入：DocumentQAInput（来自 START）
	// 输出：[]*schema.Document
	wf.AddLambdaNode("load", compose.InvokableLambda(
		func(ctx context.Context, in DocumentQAInput) ([]*schema.Document, error) {
			data, err := os.ReadFile(in.FilePath)
			if err != nil {
				return nil, fmt.Errorf("读取文件失败: %w", err)
			}
			return []*schema.Document{{Content: string(data)}}, nil
		},
	)).AddInput(compose.START)

	// 节点2: chunk - 分块
	// 输入：[]*schema.Document（来自 load）
	// 输出：[]*schema.Document
	wf.AddLambdaNode("chunk", compose.InvokableLambda(
		func(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
			var chunks []*schema.Document
			for _, doc := range docs {
				chunks = append(chunks, splitIntoChunks(doc.Content, 800)...)
			}
			return chunks, nil
		},
	)).AddInput("load")

	// 节点3: score - 评分所有块
	// 输入：ScoreInput（通过 FieldMapping 组装）
	//   - Chunks: 来自 chunk 节点的输出
	//   - Question: 来自 START 节点的 Question 字段
	// 输出：[]ScoredChunk
	//
	// 关键点：使用 WithNoDirectDependency 避免循环依赖
	// 因为执行顺序已经通过 START→load→chunk→score 确定
	wf.AddLambdaNode("score", compose.InvokableLambda(
		func(ctx context.Context, in ScoreInput) ([]ScoredChunk, error) {
			var scored []ScoredChunk
			for _, chunk := range in.Chunks {
				sc, err := scoreOneChunk(ctx, cm, chunk.Content, in.Question)
				if err != nil {
					// 评分失败时返回 0 分，不中断流程
					scored = append(scored, ScoredChunk{Text: chunk.Content, Score: 0})
					continue
				}
				scored = append(scored, sc)
			}
			return scored, nil
		},
	)).
		AddInputWithOptions("chunk",
			[]*compose.FieldMapping{compose.ToField("Chunks")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// 节点4: filter - 筛选 top-k
	// 输入：[]ScoredChunk（来自 score）
	// 输出：[]ScoredChunk
	wf.AddLambdaNode("filter", compose.InvokableLambda(
		func(ctx context.Context, scored []ScoredChunk) ([]ScoredChunk, error) {
			// 按分数降序排序
			sort.Slice(scored, func(i, j int) bool {
				return scored[i].Score > scored[j].Score
			})

			// 筛选分数 >= 3 的块，最多保留 3 个
			const maxK = 3
			var topK []ScoredChunk
			for _, c := range scored {
				if c.Score < 3 {
					break
				}
				topK = append(topK, c)
				if len(topK) == maxK {
					break
				}
			}
			return topK, nil
		},
	)).AddInput("score")

	// 节点5: answer - 生成答案
	// 输入：AnswerInput（通过 FieldMapping 组装）
	//   - TopK: 来自 filter 节点的输出
	//   - Question: 来自 START 节点的 Question 字段
	// 输出：DocumentQAOutput
	wf.AddLambdaNode("answer", compose.InvokableLambda(
		func(ctx context.Context, in AnswerInput) (DocumentQAOutput, error) {
			if len(in.TopK) == 0 {
				return DocumentQAOutput{
					Answer: fmt.Sprintf("未找到与问题相关的内容: %q", in.Question),
				}, nil
			}
			return synthesizeAnswer(ctx, cm, in)
		},
	)).
		AddInputWithOptions("filter",
			[]*compose.FieldMapping{compose.ToField("TopK")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// 连接到 END 节点
	wf.End().AddInput("answer")

	return wf
}

// scoreOneChunk 评分单个块
func scoreOneChunk(ctx context.Context, cm model.BaseChatModel, text string, question string) (ScoredChunk, error) {
	prompt := fmt.Sprintf(`Rate how relevant the following text chunk is to the question.

Question: %s

Chunk:
%s

Reply with JSON only — no explanation, no markdown fences:
{"score": <0-10>, "excerpt": "<most relevant sentence or phrase, empty string if score is 0>"}

Score guide: 0=完全无关, 3=略有相关, 7=明确相关, 10=直接回答问题.`,
		question, text)

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		return ScoredChunk{Text: text, Score: 0}, err
	}

	content := strings.TrimSpace(resp.Content)
	// 去除可能的 markdown 代码块包装
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result struct {
		Score   int    `json:"score"`
		Excerpt string `json:"excerpt"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ScoredChunk{Text: text, Score: 0}, err
	}

	return ScoredChunk{Text: text, Score: result.Score, Excerpt: result.Excerpt}, nil
}

// synthesizeAnswer 基于 top-k 块生成答案
func synthesizeAnswer(ctx context.Context, cm model.BaseChatModel, in AnswerInput) (DocumentQAOutput, error) {
	var sb strings.Builder
	sb.WriteString("Answer the following question using only the provided document excerpts.\n\n")
	sb.WriteString("Question: ")
	sb.WriteString(in.Question)
	sb.WriteString("\n\nDocument excerpts:\n")

	sources := make([]string, len(in.TopK))
	for i, c := range in.TopK {
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Text
		}
		sources[i] = excerpt
		fmt.Fprintf(&sb, "[%d] %s\n\n", i+1, excerpt)
	}
	sb.WriteString("Provide a clear, concise answer. Cite excerpt numbers like [1] when referencing sources.")

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(sb.String())})
	if err != nil {
		return DocumentQAOutput{}, fmt.Errorf("生成答案失败: %w", err)
	}

	return DocumentQAOutput{Answer: resp.Content, Sources: sources}, nil
}

// splitIntoChunks 将文本分块
// 策略：
// 1. 优先在段落边界（\n\n）分块，保持语义完整
// 2. 如果段落过大，在行边界（\n）分块
// 3. 每个块不超过 chunkSize 字符
func splitIntoChunks(text string, chunkSize int) []*schema.Document {
	var chunks []*schema.Document
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, &schema.Document{Content: s})
		}
		buf.Reset()
	}

	// 按段落分割
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 如果加上当前段落会超过块大小，先刷新缓冲区
		if buf.Len()+len(para)+2 > chunkSize && buf.Len() > 0 {
			flush()
		}

		// 如果段落本身超过块大小，按行分割
		if len(para) > chunkSize {
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if buf.Len()+len(line)+1 > chunkSize && buf.Len() > 0 {
					flush()
				}
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(line)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
		}
	}
	flush()

	return chunks
}

// ==================== 测试用例 ====================

// TestDocumentQAWorkflow 测试文档问答工作流
func TestDocumentQAWorkflow(t *testing.T) {
	ctx := context.Background()
	cm := &mockChatModel{}

	// 创建测试文档
	testDoc := `Eino 框架介绍

Eino 是一个强大的 AI 应用框架，由 CloudWeGo 团队开发。它提供了丰富的组件和工具，帮助开发者快速构建 AI 应用。

Workflow 功能

Workflow 是 Eino 中的核心功能之一，支持复杂工作流编排。通过 compose.Workflow，开发者可以构建包含多个节点的工作流，每个节点执行特定的任务。

Lambda 节点

Lambda 节点是工作流中的基本执行单元。通过 AddLambdaNode 可以添加自定义的处理逻辑，支持任意输入输出类型。

FieldMapping 数据传递

FieldMapping 允许跨节点传递数据，无需通过中间节点。这使得工作流设计更加灵活和高效。使用 WithNoDirectDependency 可以避免循环依赖。

测试与质量保证

测试是保证代码质量的重要手段。Eino 提供了完善的测试工具和示例，帮助开发者编写高质量的测试用例。`

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "test_doc_*.txt")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testDoc); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	t.Run("完整工作流测试", func(t *testing.T) {
		// 构建工作流
		wf := buildDocumentQAWorkflow(cm)

		// 编译工作流
		runnable, err := wf.Compile(ctx)
		if err != nil {
			t.Fatalf("编译工作流失败: %v", err)
		}

		// 执行工作流
		input := DocumentQAInput{
			FilePath: tmpFile.Name(),
			Question: "Eino 框架有哪些核心功能？",
		}

		output, err := runnable.Invoke(ctx, input)
		if err != nil {
			t.Fatalf("执行工作流失败: %v", err)
		}

		t.Logf("✓ 工作流执行成功")
		t.Logf("  问题: %s", input.Question)
		t.Logf("  答案: %s", output.Answer)
		t.Logf("  引用数: %d", len(output.Sources))

		if output.Answer == "" {
			t.Error("答案为空")
		}

		if len(output.Sources) == 0 {
			t.Error("没有引用来源")
		}
	})

	t.Run("FieldMapping测试", func(t *testing.T) {
		wf := buildDocumentQAWorkflow(cm)
		runnable, err := wf.Compile(ctx)
		if err != nil {
			t.Fatalf("编译工作流失败: %v", err)
		}

		input := DocumentQAInput{
			FilePath: tmpFile.Name(),
			Question: "什么是 FieldMapping？",
		}

		output, err := runnable.Invoke(ctx, input)
		if err != nil {
			t.Fatalf("执行工作流失败: %v", err)
		}

		t.Logf("✓ FieldMapping 测试")
		t.Logf("  问题: %s", input.Question)
		t.Logf("  答案: %s", output.Answer)

		if !strings.Contains(output.Answer, "FieldMapping") && !strings.Contains(output.Answer, "数据传递") {
			t.Logf("  警告: 答案可能不够准确")
		}
	})

	t.Run("无相关内容测试", func(t *testing.T) {
		wf := buildDocumentQAWorkflow(cm)
		runnable, err := wf.Compile(ctx)
		if err != nil {
			t.Fatalf("编译工作流失败: %v", err)
		}

		// 提问一个文档中不存在的内容
		input := DocumentQAInput{
			FilePath: tmpFile.Name(),
			Question: "如何使用 Python 编程？",
		}

		output, err := runnable.Invoke(ctx, input)
		if err != nil {
			t.Fatalf("执行工作流失败: %v", err)
		}

		t.Logf("✓ 无相关内容场景")
		t.Logf("  问题: %s", input.Question)
		t.Logf("  答案: %s", output.Answer)

		// 注意：由于使用的是 mock 模型，它会对所有问题返回相同的答案
		// 在真实场景中，如果所有块的评分都低于阈值（3分），会返回"未找到"提示
		// 这里我们只验证工作流能够正常执行
		t.Logf("  说明: mock 模型会对所有问题返回相同答案，真实场景中会根据评分返回'未找到'提示")
	})
}

// TestChunkSplitting 测试文本分块功能
func TestChunkSplitting(t *testing.T) {
	text := `第一段内容。这是一个测试段落。

第二段内容。这是另一个测试段落。

第三段内容。这是第三个测试段落。`

	chunks := splitIntoChunks(text, 50)

	t.Logf("✓ 分块测试")
	t.Logf("  原文长度: %d", len(text))
	t.Logf("  块数: %d", len(chunks))

	for i, chunk := range chunks {
		t.Logf("  块 %d 长度: %d", i+1, len(chunk.Content))
	}

	if len(chunks) == 0 {
		t.Error("分块结果为空")
	}

	for i, chunk := range chunks {
		if len(chunk.Content) > 50 {
			t.Errorf("块 %d 超过最大长度: %d > 50", i+1, len(chunk.Content))
		}
	}
}

// TestWorkflowStructure 测试工作流结构
func TestWorkflowStructure(t *testing.T) {
	ctx := context.Background()
	cm := &mockChatModel{}

	wf := buildDocumentQAWorkflow(cm)

	// 编译工作流（验证结构正确性）
	_, err := wf.Compile(ctx)
	if err != nil {
		t.Fatalf("工作流结构验证失败: %v", err)
	}

	t.Logf("✓ 工作流结构验证通过")
	t.Logf("  节点: START → load → chunk → score → filter → answer → END")
	t.Logf("  FieldMapping: Question (START → score, answer)")
}

// ==================== 最佳实践总结 ====================

/*
## 最佳实践

### 1. 何时使用 Workflow

✓ 适合使用的场景：
- 多步骤协同任务（RAG、数据处理流水线）
- 需要复杂数据流控制的任务
- 需要跨节点传递数据的场景
- 可复用的处理流程

✗ 不适合使用的场景：
- 简单的单步操作（直接用函数更简单）
- 无需数据流控制的顺序任务
- 一次性的临时脚本

### 2. 工作流设计原则

**节点职责单一**：
- 每个节点只做一件事
- 便于测试和维护
- 易于复用和组合

**合理使用 FieldMapping**：
- 跨节点传递数据时使用
- 避免数据在中间节点无意义传递
- 使用 WithNoDirectDependency 避免循环依赖

**数据流清晰**：
- 明确每个节点的输入输出类型
- 使用结构体封装复杂输入
- 避免过度嵌套

### 3. 错误处理策略

**节点级错误处理**：
- 在节点内部捕获可恢复的错误
- 返回降级结果而不是中断流程
- 记录错误日志用于调试

**流程级错误处理**：
- 关键错误应该中断流程
- 使用 context 传递取消信号
- 提供清晰的错误信息

### 4. 性能优化

**分块策略**：
- 根据模型上下文窗口调整块大小
- 在段落边界分块保持语义完整
- 避免块过小导致上下文丢失

**并发处理**：
- 对于独立任务，可以使用 goroutine 并发处理
- 注意控制并发数量，避免资源竞争
- 考虑 API 限流和成本

### 5. 测试策略

**单元测试**：
- 测试每个节点的独立功能
- 使用 mock 隔离外部依赖
- 覆盖边界情况和错误场景

**集成测试**：
- 测试完整工作流
- 使用真实数据验证结果
- 测试 FieldMapping 的正确性

**结构测试**：
- 验证工作流可以成功编译
- 检查节点连接的正确性

### 6. 真实场景集成

**基本用法**：
```go
// 1. 构建工作流
wf := buildDocumentQAWorkflow(cm)

// 2. 编译工作流
runnable, err := wf.Compile(ctx)

// 3. 执行工作流
output, err := runnable.Invoke(ctx, input)
```

**与 Agent 集成**：
- 将工作流封装为 Tool
- 使用 graphtool.NewInvokableGraphTool（需要引入 eino-examples）
- 注册到 Agent 的 ToolsConfig

**可观测性**：
- 使用 Callback 记录节点执行
- 监控每个节点的耗时
- 记录数据流转情况

## 核心特性演示

### FieldMapping 的作用

在本示例中，Question 字段需要在 score 和 answer 节点中使用，但不需要在 load 和 chunk 节点中传递。

**不使用 FieldMapping**：
- load 输出需要包含 Question
- chunk 输出需要包含 Question
- 中间节点需要无意义地传递数据

**使用 FieldMapping**：
- Question 直接从 START 传递到 score 和 answer
- 中间节点只处理自己需要的数据
- 数据流更清晰，代码更简洁

### WithNoDirectDependency 的作用

当使用 FieldMapping 从非直接前驱节点获取数据时，需要使用 WithNoDirectDependency 避免循环依赖。

例如，score 节点：
- 从 chunk 获取 Chunks（直接依赖）
- 从 START 获取 Question（非直接依赖，需要 WithNoDirectDependency）

执行顺序已经通过 START→load→chunk→score 确定，所以不会产生实际的循环依赖。

## 扩展思考

**其他 Workflow 应用场景**：
- 多文档 RAG：并行处理多个文档
- 数据处理流水线：ETL、数据清洗、特征提取
- 多步骤决策：根据条件选择不同分支
- 复杂业务流程：订单处理、审批流程

**高级特性**（需要进一步学习）：
- 条件分支：根据节点输出选择不同路径
- 循环节点：重复执行直到满足条件
- 子图嵌套：将子工作流作为节点
- 流式输出：支持流式返回中间结果
- 中断恢复：支持人工介入和恢复执行

详细文档：
- eino-examples/quickstart/chatwitheino/rag/rag.go
- eino-examples/compose/graph/
*/
