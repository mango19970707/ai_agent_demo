package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestSequentialAgent 演示如何使用 SequentialAgent（串行流水线智能体）
//
// SequentialAgent 按顺序执行多个子智能体，前一个智能体的输出会传递给下一个智能体。
//
// 使用场景：
// - 多阶段任务处理（规划 → 执行 → 审查）
// - 流水线式工作流（数据收集 → 数据清洗 → 数据分析）
// - 需要保持执行顺序的任务链
//
// 典型应用：
// - 内容创作：大纲生成 → 内容撰写 → 内容润色
// - 代码开发：需求分析 → 代码实现 → 代码审查
// - 数据处理：数据提取 → 数据转换 → 数据加载（ETL）
func TestSequentialAgent(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	// 创建模型
	chatModel, err := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// 1. 创建第一个子智能体：规划者
	plannerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner",
		Description: "负责制定写作大纲",
		Instruction: `你是一个专业的写作规划师。
根据用户的主题，生成一个清晰的文章大纲。
大纲应包含：
1. 引言
2. 3-5个主要观点
3. 结论

请只输出大纲，不要写具体内容。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建规划智能体失败: %v", err)
	}

	// 2. 创建第二个子智能体：写作者
	writerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer",
		Description: "负责根据大纲撰写文章",
		Instruction: `你是一个专业的内容写作者。
根据提供的大纲，撰写一篇完整的文章。
要求：
- 每个部分都要充分展开
- 语言流畅、逻辑清晰
- 字数在 500-800 字之间

请直接输出文章内容。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建写作智能体失败: %v", err)
	}

	// 3. 创建第三个子智能体：审查者
	reviewerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "reviewer",
		Description: "负责审查和润色文章",
		Instruction: `你是一个专业的内容审查编辑。
审查提供的文章，并进行润色优化。
检查要点：
- 语法和拼写错误
- 逻辑连贯性
- 表达是否清晰
- 是否有改进空间

请输出润色后的最终版本。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建审查智能体失败: %v", err)
	}

	// 4. 创建 SequentialAgent，按顺序执行三个子智能体
	sequentialAgent, err := NewSequentialAgent(ctx, &SequentialAgentConfig{
		Name:        "content_pipeline",
		Description: "内容创作流水线：规划 → 撰写 → 审查",
		SubAgents: []adk.Agent{
			plannerAgent,  // 第一步：制定大纲
			writerAgent,   // 第二步：撰写内容
			reviewerAgent, // 第三步：审查润色
		},
	})
	if err != nil {
		t.Fatalf("创建串行智能体失败: %v", err)
	}

	// 5. 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           sequentialAgent,
	})

	// 6. 发送查询
	query := "请写一篇关于'人工智能对教育的影响'的文章"
	fmt.Println("用户:", query)
	fmt.Println("\n开始执行串行流水线...\n")

	iter := runner.Query(ctx, query)

	// 7. 处理响应
	currentAgent := ""
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 显示当前执行的智能体
		if event.AgentName != currentAgent {
			currentAgent = event.AgentName
			fmt.Printf("\n=== %s 正在工作 ===\n", currentAgent)
		}

		// 获取输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println(msg.Content)
			}
		}
	}

	fmt.Println("\n流水线执行完成！")
}

// TestSequentialAgentWithDataTransform 演示数据在子智能体之间的传递
//
// 在 SequentialAgent 中，每个子智能体的输出会自动成为下一个子智能体的输入。
// 这种模式非常适合数据转换和处理流程。
func TestSequentialAgentWithDataTransform(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 子智能体 1：提取关键信息
	extractorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "extractor",
		Description: "提取文本中的关键信息",
		Instruction: `从输入文本中提取以下信息：
- 主要人物
- 关键事件
- 时间地点
以结构化的方式输出。`,
		Model: chatModel,
	})

	// 子智能体 2：总结内容
	summarizerAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "summarizer",
		Description: "总结提取的信息",
		Instruction: `根据提取的关键信息，生成一段简洁的摘要。
摘要应该在 100 字以内。`,
		Model: chatModel,
	})

	// 子智能体 3：翻译成英文
	translatorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "translator",
		Description: "将摘要翻译成英文",
		Instruction: `将提供的中文摘要翻译成英文。
保持原意，使用地道的英文表达。`,
		Model: chatModel,
	})

	// 创建串行智能体
	sequentialAgent, _ := NewSequentialAgent(ctx, &SequentialAgentConfig{
		Name:        "text_processor",
		Description: "文本处理流水线：提取 → 总结 → 翻译",
		SubAgents: []adk.Agent{
			extractorAgent,
			summarizerAgent,
			translatorAgent,
		},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: sequentialAgent,
	})

	// 输入一段新闻文本
	newsText := `2024年5月7日，在北京举行的人工智能大会上，
OpenAI 公司 CEO Sam Altman 宣布推出最新的 GPT-5 模型。
该模型在推理能力、多模态理解和代码生成方面都有显著提升。
大会吸引了来自全球的 5000 多名 AI 研究者和开发者参与。`

	fmt.Println("原始文本:", newsText)
	fmt.Println("\n开始处理...\n")

	iter := runner.Query(ctx, newsText)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("错误: %v\n", event.Err)
			break
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Printf("[%s] %s\n\n", event.AgentName, msg.Content)
			}
		}
	}
}

// 使用建议：
//
// 1. 子智能体设计：
//    - 每个子智能体应该有明确的单一职责
//    - 子智能体的输出应该是下一个子智能体可以理解的格式
//    - 避免子智能体之间的职责重叠
//
// 2. 执行顺序：
//    - 确保子智能体的执行顺序符合业务逻辑
//    - 前置步骤应该为后续步骤准备好必要的信息
//    - 考虑是否可以并行执行（如果可以，使用 ParallelAgent）
//
// 3. 错误处理：
//    - 如果某个子智能体失败，整个流水线会停止
//    - 考虑在关键步骤添加验证逻辑
//    - 可以使用 LoopAgent 实现重试机制
//
// 4. 性能考虑：
//    - 串行执行意味着总时间是所有子智能体时间之和
//    - 如果子智能体之间没有依赖，考虑使用 ParallelAgent
//    - 合理设置每个子智能体的超时时间
//
// 5. 调试技巧：
//    - 通过 event.AgentName 追踪当前执行的子智能体
//    - 记录每个子智能体的输入和输出
//    - 可以单独测试每个子智能体的功能
//
// 6. 典型模式：
//    - 三阶段模式：准备 → 执行 → 验证
//    - ETL 模式：提取 → 转换 → 加载
//    - 创作模式：规划 → 创作 → 审查
//    - 分析模式：收集 → 分析 → 报告
