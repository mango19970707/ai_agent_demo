package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestDeepAgent_BasicUsage 测试 DeepAgent 基础使用
// 场景：使用 DeepAgent 处理复杂的多步骤任务
func TestDeepAgent_BasicUsage(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	// 创建聊天模型
	chatModel, err := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})
	if err != nil {
		t.Fatalf("创建聊天模型失败: %v", err)
	}

	// 创建 DeepAgent
	deepAgent, err := NewDeepAgent(ctx, &DeepAgentConfig{
		Name:         "deep_agent",
		Description:  "一个能够处理复杂任务的深度智能体",
		ChatModel:    chatModel,
		Instruction:  "你是一个专业的任务处理助手，擅长将复杂任务分解为多个子任务并逐步完成。",
		MaxIteration: 50,
		// 使用默认配置：启用 write_todos 工具，启用通用子智能体
	})
	if err != nil {
		t.Fatalf("创建 DeepAgent 失败: %v", err)
	}

	fmt.Println("=== DeepAgent 基础使用示例 ===")
	fmt.Println("场景：处理复杂的多步骤任务")

	// 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           deepAgent,
	})

	query := `请帮我完成以下任务：
1. 分析一下 Go 语言的并发模型
2. 总结 goroutine 和 channel 的核心特点
3. 给出一个实际应用示例

请使用 write_todos 工具管理任务进度。`

	fmt.Println("\n用户输入:")
	fmt.Println(query)
	fmt.Println("\nDeepAgent 处理中...")

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行失败: %v", event.Err)
		}

		// 显示输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("\n[输出]")
				fmt.Println(msg.Content)
			}
		}
	}

	fmt.Println("\n任务完成！")
}

// TestDeepAgent_WithSubAgents 测试 DeepAgent 协调子智能体
// 场景：使用 DeepAgent 协调多个专业子智能体完成复杂任务
func TestDeepAgent_WithSubAgents(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	// 创建聊天模型
	chatModel, err := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})
	if err != nil {
		t.Fatalf("创建聊天模型失败: %v", err)
	}

	// 创建专业子智能体：数据分析师
	dataAnalystAgent, err := NewChatModelAgent(ctx, chatModel, &AgentConfig{
		Name:        "data_analyst",
		Description: "专业的数据分析师，擅长数据清洗、统计分析和可视化",
		Instruction: "你是一个专业的数据分析师，擅长处理数据、进行统计分析并给出洞察。",
	})
	if err != nil {
		t.Fatalf("创建数据分析师智能体失败: %v", err)
	}

	// 创建专业子智能体：报告撰写员
	reportWriterAgent, err := NewChatModelAgent(ctx, chatModel, &AgentConfig{
		Name:        "report_writer",
		Description: "专业的报告撰写员，擅长撰写结构化的分析报告",
		Instruction: "你是一个专业的报告撰写员，擅长将分析结果整理成清晰、专业的报告。",
	})
	if err != nil {
		t.Fatalf("创建报告撰写员智能体失败: %v", err)
	}

	// 创建 DeepAgent，配置子智能体
	deepAgent, err := NewDeepAgent(ctx, &DeepAgentConfig{
		Name:        "project_manager",
		Description: "项目经理，负责协调数据分析和报告撰写工作",
		ChatModel:   chatModel,
		Instruction: `你是一个项目经理，负责协调团队完成数据分析项目。
你的团队包括：
- data_analyst：数据分析师，负责数据处理和分析
- report_writer：报告撰写员，负责撰写分析报告

请合理分配任务给团队成员，并使用 write_todos 工具追踪进度。`,
		SubAgents: []adk.Agent{
			dataAnalystAgent,
			reportWriterAgent,
		},
		MaxIteration: 50,
	})
	if err != nil {
		t.Fatalf("创建 DeepAgent 失败: %v", err)
	}

	fmt.Println("=== DeepAgent 协调子智能体示例 ===")
	fmt.Println("场景：项目经理协调数据分析师和报告撰写员完成项目")

	// 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           deepAgent,
	})

	query := `我们有一个电商网站的用户行为数据，包括：
- 用户访问量：每日 10 万次
- 转化率：2.5%
- 平均订单金额：150 元
- 用户留存率：30 天留存 40%

请分析这些数据，并生成一份分析报告，包括：
1. 数据解读和关键指标分析
2. 与行业平均水平的对比
3. 改进建议

请合理分配任务给团队成员。`

	fmt.Println("\n用户输入:")
	fmt.Println(query)
	fmt.Println("\nDeepAgent 协调处理中...")

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行失败: %v", event.Err)
		}

		// 显示输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("\n[输出]")
				fmt.Println(msg.Content)
			}
		}
	}

	fmt.Println("\n任务完成！")
}

// TestDeepAgent_IterativeRefinement 测试 DeepAgent 迭代优化
// 场景：使用 DeepAgent 进行多轮迭代优化
func TestDeepAgent_IterativeRefinement(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	// 创建聊天模型
	chatModel, err := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})
	if err != nil {
		t.Fatalf("创建聊天模型失败: %v", err)
	}

	// 创建 DeepAgent
	deepAgent, err := NewDeepAgent(ctx, &DeepAgentConfig{
		Name:        "content_optimizer",
		Description: "内容优化专家，擅长迭代优化文案",
		ChatModel:   chatModel,
		Instruction: `你是一个内容优化专家，擅长通过多轮迭代优化文案。
你的工作流程：
1. 分析原始文案的问题
2. 提出改进方案
3. 生成优化后的文案
4. 评估优化效果
5. 如果需要，继续迭代优化

使用 write_todos 工具管理优化任务。`,
		MaxIteration: 50,
	})
	if err != nil {
		t.Fatalf("创建 DeepAgent 失败: %v", err)
	}

	fmt.Println("=== DeepAgent 迭代优化示例 ===")
	fmt.Println("场景：多轮迭代优化产品文案")

	// 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           deepAgent,
	})

	query := `请帮我优化以下产品文案：

"我们的产品很好用，功能很多，价格也不贵，欢迎购买。"

优化要求：
1. 更具体地描述产品特点
2. 突出核心卖点
3. 增加情感共鸣
4. 添加行动号召

请进行多轮迭代，直到文案达到专业水准。`

	fmt.Println("\n用户输入:")
	fmt.Println(query)
	fmt.Println("\nDeepAgent 迭代优化中...")

	iter := runner.Query(ctx, query)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行失败: %v", event.Err)
		}

		// 显示输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println("\n[输出]")
				fmt.Println(msg.Content)
			}
		}
	}

	fmt.Println("\n优化完成！")
}

// 使用建议：
//
// 1. DeepAgent 的核心特点：
//    - 自主任务分解：智能体自己决定如何分解任务
//    - 智能任务分配：自动选择合适的子智能体
//    - 任务进度追踪：使用 write_todos 工具管理待办事项
//    - 丰富的工具支持：文件系统、Shell 命令、自定义工具
//
// 2. 适用场景：
//    ✓ 复杂的多步骤任务，需要协调多个工具和子智能体
//    ✓ 需要文件操作或 Shell 命令执行的任务
//    ✓ 需要自主任务管理和进度追踪的场景
//    ✓ 需要迭代优化的任务
//    ✗ 简单的单步任务（使用 ChatModelAgent）
//    ✗ 固定流程的任务（使用 SequentialAgent）
//
// 3. 与其他智能体的对比：
//    - ChatModelAgent：基础对话，功能单一
//    - SupervisorAgent：中心化协调，需要明确指定任务分配
//    - PlanExecuteReplanAgent：规划-执行-重规划，适合有明确计划的任务
//    - DeepAgent：自主任务编排，智能体自己决定如何分解和分配任务
//
// 4. 配置建议：
//    - MaxIteration：建议设置为 50-100，防止无限循环
//    - SubAgents：配置专业领域的子智能体，每个子智能体职责明确
//    - Instruction：清晰说明智能体的角色和工作流程
//    - WithoutWriteTodos：通常保持 false，启用任务管理功能
//    - WithoutGeneralSubAgent：通常保持 false，启用通用子智能体
//
// 5. 内置工具：
//    - write_todos：管理任务列表（自动启用）
//    - read_file、write_file、edit_file：文件操作（需要配置 Backend）
//    - glob、grep：文件搜索（需要配置 Backend）
//    - execute：Shell 命令执行（需要配置 Shell）
//    - task：调用子智能体（自动创建）
//
// 6. 典型应用模式：
//    - 数据处理：DeepAgent + 数据清洗子智能体 + 数据分析子智能体
//    - 代码开发：DeepAgent + 代码生成子智能体 + 测试子智能体
//    - 文档处理：DeepAgent + 文档解析子智能体 + 内容生成子智能体
//    - 研究分析：DeepAgent + 搜索子智能体 + 分析子智能体
//    - Excel 处理：DeepAgent + 代码执行子智能体 + 文件操作工具
//
// 7. 最佳实践：
//    - 让 DeepAgent 使用 write_todos 工具管理任务进度
//    - 子智能体应该有明确的专业领域和职责
//    - 在 Instruction 中说明子智能体的能力和使用场景
//    - 合理设置 MaxIteration，避免无限循环
//    - 配置必要的工具（文件系统、Shell）以支持复杂操作
//
// 8. 调试技巧：
//    - 观察 write_todos 工具的调用，了解任务分解情况
//    - 追踪子智能体的调用，验证任务分配是否合理
//    - 检查迭代次数，判断是否需要调整 MaxIteration
//    - 分析任务完成情况，优化 Instruction 和子智能体配置
//
// 9. 性能考虑：
//    - DeepAgent 会进行多次 LLM 调用（任务分解、子智能体调用等）
//    - 总时间 = 任务分解时间 + 所有子智能体执行时间 + 结果整合时间
//    - 合理配置子智能体数量，避免过度复杂
//    - 监控迭代次数，防止无限循环
//
// 10. 常见问题：
//     - 任务分解不合理：优化 Instruction，明确任务分解策略
//     - 子智能体选择错误：完善子智能体的 Description
//     - 迭代次数过多：检查任务完成条件，调整 MaxIteration
//     - 工具调用失败：确保 Backend、Shell 配置正确
