package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestLoopAgent 演示如何使用 LoopAgent（循环执行智能体）
//
// LoopAgent 循环执行多个子智能体，直到满足退出条件或达到最大迭代次数。
//
// 使用场景：
// - 迭代优化（生成 → 评估 → 改进 → 再评估）
// - 反思模式（执行 → 反思 → 改进）
// - 多轮对话（问答 → 追问 → 再答）
// - 渐进式改进（初稿 → 修改 → 再修改）
//
// 典型应用：
// - 代码生成与优化：生成代码 → 检查错误 → 修复 → 再检查
// - 内容创作：写作 → 评审 → 修改 → 再评审
// - 问题求解：尝试方案 → 验证 → 调整 → 再验证
// - 学习系统：回答 → 评分 → 改进 → 再回答
func TestLoopAgent(t *testing.T) {
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

	// 1. 创建主执行智能体：内容生成器
	mainAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer",
		Description: "负责生成或改进内容",
		Instruction: `你是一个专业的内容创作者。
根据用户的要求或评审意见，生成或改进内容。

如果这是第一次生成：
- 根据用户要求创作内容
- 确保内容完整、清晰

如果收到了评审意见：
- 仔细阅读评审意见
- 针对性地改进内容
- 说明你做了哪些改进

请直接输出内容，不要添加额外的说明。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建主智能体失败: %v", err)
	}

	// 2. 创建评审智能体：内容评审器
	critiqueAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "critic",
		Description: "负责评审内容并提供改进建议",
		Instruction: `你是一个严格的内容评审专家。
评审提供的内容，从以下角度给出意见：
1. 内容完整性
2. 逻辑清晰度
3. 语言表达
4. 是否有改进空间

评审标准：
- 如果内容已经很好，明确说"内容已达标，无需改进"
- 如果还有改进空间，具体指出问题并给出建议

请直接输出评审意见。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建评审智能体失败: %v", err)
	}

	// 3. 创建 LoopAgent，循环执行"生成 → 评审"
	loopAgent, err := NewLoopAgent(ctx, &LoopAgentConfig{
		Name:        "reflection_loop",
		Description: "反思循环：生成内容 → 评审 → 改进 → 再评审",
		SubAgents: []adk.Agent{
			mainAgent,     // 第一步：生成或改进内容
			critiqueAgent, // 第二步：评审内容
		},
		MaxIterations: 3, // 最多循环 3 次
	})
	if err != nil {
		t.Fatalf("创建循环智能体失败: %v", err)
	}

	// 4. 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           loopAgent,
	})

	// 5. 发送查询
	query := "请写一段关于'时间管理'的建议，要求简洁实用。"
	fmt.Println("用户:", query)
	fmt.Println("\n开始反思循环...")

	iter := runner.Query(ctx, query)

	// 6. 处理响应
	iteration := 0
	currentAgent := ""

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 追踪迭代次数
		if event.AgentName == "writer" && event.AgentName != currentAgent {
			iteration++
			fmt.Printf("\n=== 第 %d 轮迭代 ===\n", iteration)
		}

		currentAgent = event.AgentName

		// 获取输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Printf("\n[%s]\n%s\n", event.AgentName, msg.Content)
			}
		}

		// 检查是否提前退出
		if event.Action != nil && event.Action.BreakLoop != nil {
			fmt.Printf("\n循环在第 %d 轮后提前结束\n", event.Action.BreakLoop.CurrentIterations+1)
		}
	}

	fmt.Println("\n反思循环完成！")
}

// TestLoopAgentWithBreakCondition 演示如何使用 BreakLoop 提前退出循环
//
// 当满足特定条件时，可以使用 BreakLoopAction 提前终止循环。
func TestLoopAgentWithBreakCondition(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 问题求解器
	solverAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "solver",
		Description: "尝试解决问题",
		Instruction: `你是一个问题求解专家。
根据问题和之前的尝试（如果有），提出解决方案。
如果这不是第一次尝试，请根据验证结果调整方案。`,
		Model: chatModel,
	})

	// 验证器
	validatorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "validator",
		Description: "验证解决方案",
		Instruction: `你是一个方案验证专家。
验证提供的解决方案是否可行。

如果方案可行：
- 明确说"方案可行"
- 说明理由

如果方案不可行：
- 明确说"方案需要改进"
- 指出问题
- 给出改进建议`,
		Model: chatModel,
	})

	loopAgent, _ := NewLoopAgent(ctx, &LoopAgentConfig{
		Name:        "problem_solving_loop",
		Description: "问题求解循环：提出方案 → 验证 → 改进",
		SubAgents: []adk.Agent{
			solverAgent,
			validatorAgent,
		},
		MaxIterations: 5,
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: loopAgent,
	})

	query := "如何在一个月内学会 Python 编程？"
	fmt.Println("问题:", query)
	fmt.Println("\n开始求解...\n")

	iter := runner.Query(ctx, query)

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

		// 检查是否提前退出
		if event.Action != nil && event.Action.BreakLoop != nil {
			fmt.Println("✓ 找到可行方案，提前退出循环")
		}
	}
}

// TestLoopAgentCodeGeneration 演示代码生成与修复的循环
//
// 这是 LoopAgent 的经典应用：生成代码 → 检查错误 → 修复 → 再检查
func TestLoopAgentCodeGeneration(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 代码生成器
	codeGeneratorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "code_generator",
		Description: "生成或修复代码",
		Instruction: `你是一个 Python 代码生成专家。

如果这是第一次生成：
- 根据需求生成完整的 Python 代码
- 包含必要的注释
- 确保代码格式正确

如果收到了错误报告：
- 分析错误原因
- 修复代码
- 说明修复了什么

只输出代码，使用 markdown 代码块格式。`,
		Model: chatModel,
	})

	// 代码检查器
	codeReviewerAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "code_reviewer",
		Description: "检查代码质量和错误",
		Instruction: `你是一个代码审查专家。
检查提供的 Python 代码，从以下角度评估：
1. 语法错误
2. 逻辑错误
3. 代码规范
4. 性能问题
5. 安全隐患

如果代码没有问题：
- 明确说"代码质量良好，无需修改"

如果有问题：
- 列出所有问题
- 给出具体的修改建议`,
		Model: chatModel,
	})

	loopAgent, _ := NewLoopAgent(ctx, &LoopAgentConfig{
		Name:        "code_generation_loop",
		Description: "代码生成循环：生成 → 检查 → 修复",
		SubAgents: []adk.Agent{
			codeGeneratorAgent,
			codeReviewerAgent,
		},
		MaxIterations: 4,
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: loopAgent,
	})

	query := "写一个 Python 函数，计算斐波那契数列的第 n 项，要求使用递归实现。"
	fmt.Println("需求:", query)
	fmt.Println("\n开始生成代码...\n")

	iter := runner.Query(ctx, query)

	iteration := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			fmt.Printf("错误: %v\n", event.Err)
			break
		}

		if event.AgentName == "code_generator" {
			iteration++
			fmt.Printf("=== 第 %d 次生成 ===\n", iteration)
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Printf("[%s]\n%s\n\n", event.AgentName, msg.Content)
			}
		}
	}

	fmt.Println("代码生成完成！")
}

// 使用建议：
//
// 1. 循环设计：
//    - 明确循环的目的（优化、验证、改进等）
//    - 设计合理的退出条件
//    - 避免无限循环（设置 MaxIterations）
//    - 每次迭代应该有明确的改进
//
// 2. 子智能体配置：
//    - 第一个智能体：执行主要任务（生成、求解等）
//    - 第二个智能体：评估或验证（评审、检查等）
//    - 可以有多个子智能体，按顺序循环执行
//    - 评估智能体应该能判断是否需要继续循环
//
// 3. 退出条件：
//    - 自然退出：达到 MaxIterations
//    - 提前退出：使用 BreakLoopAction
//    - 评估智能体可以决定何时退出
//    - 考虑设置质量阈值
//
// 4. 性能考虑：
//    - 每次迭代都会调用所有子智能体
//    - 总时间 = 单次迭代时间 × 实际迭代次数
//    - 合理设置 MaxIterations（通常 3-5 次）
//    - 监控迭代次数，避免过度优化
//
// 5. 典型模式：
//    - 反思模式：执行 → 反思 → 改进
//    - 验证模式：生成 → 验证 → 修复
//    - 优化模式：初始方案 → 评估 → 优化
//    - 对话模式：回答 → 追问 → 再答
//
// 6. 与其他 Agent 的对比：
//    - LoopAgent：需要多轮迭代改进
//    - SequentialAgent：一次性顺序执行
//    - ParallelAgent：并行执行，无迭代
//
// 7. 调试技巧：
//    - 记录每次迭代的输入和输出
//    - 追踪迭代次数和退出原因
//    - 分析是否真的在改进（避免原地打转）
//    - 检查是否过早或过晚退出
//
// 8. 常见问题：
//    - 无限循环：设置合理的 MaxIterations
//    - 原地打转：改进评估智能体的判断逻辑
//    - 过度优化：设置明确的"足够好"标准
//    - 退出太早：调整评估标准或增加迭代次数
//
// 9. 最佳实践：
//    - 第一次迭代应该产生基本可用的结果
//    - 后续迭代应该逐步改进
//    - 评估智能体应该给出具体的改进建议
//    - 执行智能体应该能理解并应用改进建议
//    - 设置合理的迭代次数（3-5次通常足够）
