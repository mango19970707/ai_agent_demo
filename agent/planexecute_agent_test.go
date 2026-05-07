package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestPlanExecuteReplanAgent 演示如何使用 Plan-Execute-Replan 智能体
//
// Plan-Execute-Replan 是一个强大的问题解决模式，通过三阶段循环实现复杂任务：
// 1. Planning（规划）：分析目标，生成结构化的执行计划
// 2. Execution（执行）：执行计划中的第一步
// 3. Replanning（重规划）：评估进度，决定继续或完成
//
// 使用场景：
// - 复杂任务分解（将大任务拆解为可执行步骤）
// - 动态策略调整（根据执行结果调整计划）
// - 迭代优化（多轮执行-评估-重规划循环）
// - 不确定性处理（初始信息不完整，边执行边调整）
//
// 典型应用：
// - 研究报告生成：规划大纲 → 收集资料 → 撰写内容 → 评估完整性 → 补充缺失部分
// - 软件开发：需求分析 → 设计方案 → 编码实现 → 测试验证 → 修复问题
// - 数据分析：确定目标 → 数据收集 → 统计分析 → 结果解读 → 深入挖掘
// - 旅行规划：确定目的地 → 查询交通 → 预订酒店 → 安排行程 → 优化路线
func TestPlanExecuteReplanAgent(t *testing.T) {
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

	// 1. 创建 Planner（规划智能体）
	// Planner 负责分析用户目标，生成结构化的执行计划
	plannerAgent, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: chatModel, // 使用工具调用模式生成结构化计划
		// 可选：自定义 ToolInfo 来定义计划的结构
		// ToolInfo: customPlanToolInfo,
		// 可选：自定义输入生成函数
		// GenInputFn: customGenPlannerInputFn,
	})
	if err != nil {
		t.Fatalf("创建 Planner 失败: %v", err)
	}

	// 2. 创建 Executor（执行智能体）
	// Executor 负责执行计划中的步骤
	executorAgent, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: chatModel,
		// 可选：配置工具，让 Executor 能够执行具体操作
		// ToolsConfig: adk.ToolsConfig{
		//     Tools: []tool.BaseTool{searchTool, calculatorTool},
		// },
		MaxIterations: 20, // 最大迭代次数（防止工具调用无限循环）
		// 可选：自定义输入生成函数
		// GenInputFn: customGenExecutorInputFn,
	})
	if err != nil {
		t.Fatalf("创建 Executor 失败: %v", err)
	}

	// 3. 创建 Replanner（重规划智能体）
	// Replanner 负责评估执行结果，决定继续或完成
	replannerAgent, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: chatModel, // 使用工具调用模式（plan 或 respond）
		// 可选：自定义工具定义
		// PlanTool: customPlanTool,
		// RespondTool: customRespondTool,
		// 可选：自定义输入生成函数
		// GenInputFn: customGenReplannerInputFn,
	})
	if err != nil {
		t.Fatalf("创建 Replanner 失败: %v", err)
	}

	// 4. 创建 Plan-Execute-Replan 智能体
	perAgent, err := NewPlanExecuteReplanAgent(ctx, &PlanExecuteReplanConfig{
		Planner:       plannerAgent,
		Executor:      executorAgent,
		Replanner:     replannerAgent,
		MaxIterations: 5, // 最多执行 5 轮 execute-replan 循环
	})
	if err != nil {
		t.Fatalf("创建 Plan-Execute-Replan 智能体失败: %v", err)
	}

	// 5. 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           perAgent,
	})

	// 6. 发送查询
	query := "请帮我写一篇关于'大语言模型的发展历程'的文章，要求包含关键里程碑和技术突破。"
	fmt.Println("用户:", query)
	fmt.Println("\n开始执行 Plan-Execute-Replan 流程...\n")

	iter := runner.Query(ctx, query)

	// 7. 处理响应
	phase := ""
	iteration := 0

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 追踪执行阶段
		currentPhase := ""
		switch event.AgentName {
		case "planner":
			currentPhase = "Planning（规划阶段）"
		case "executor":
			currentPhase = "Execution（执行阶段）"
			if phase != currentPhase {
				iteration++
			}
		case "replanner":
			currentPhase = "Replanning（重规划阶段）"
		}

		if currentPhase != phase {
			phase = currentPhase
			if event.AgentName == "executor" {
				fmt.Printf("\n=== 第 %d 轮迭代 ===\n", iteration)
			}
			fmt.Printf("\n--- %s ---\n", phase)
		}

		// 显示输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Println(msg.Content)
			}
		}

		// 检查是否提前退出
		if event.Action != nil && event.Action.BreakLoop != nil {
			fmt.Printf("\n✓ 任务完成，在第 %d 轮后退出\n", event.Action.BreakLoop.CurrentIterations+1)
		}
	}

	fmt.Println("\n\nPlan-Execute-Replan 流程完成！")
}

// TestPlanExecuteReplanAgentResearch 演示研究任务场景
//
// 使用 Plan-Execute-Replan 模式完成一个研究任务。
func TestPlanExecuteReplanAgentResearch(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 创建三个核心智能体
	plannerAgent, _ := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: chatModel,
	})

	executorAgent, _ := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:         chatModel,
		MaxIterations: 20,
		// 在实际应用中，这里可以配置搜索工具、数据库查询工具等
	})

	replannerAgent, _ := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: chatModel,
	})

	perAgent, _ := NewPlanExecuteReplanAgent(ctx, &PlanExecuteReplanConfig{
		Planner:       plannerAgent,
		Executor:      executorAgent,
		Replanner:     replannerAgent,
		MaxIterations: 8,
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: perAgent,
	})

	query := "研究并总结 Transformer 架构的核心创新点及其对 NLP 领域的影响。"
	fmt.Println("研究任务:", query)
	fmt.Println("\n开始研究...\n")

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
	}
}

// TestPlanExecuteReplanAgentWithCustomPlan 演示自定义计划结构
//
// 可以自定义计划的结构，以适应特定的业务需求。
func TestPlanExecuteReplanAgentWithCustomPlan(t *testing.T) {
	t.Skip("这是一个示例，展示如何自定义计划结构")

	// 自定义计划结构示例：
	//
	// type CustomPlan struct {
	//     Goal        string   `json:"goal"`         // 总体目标
	//     Steps       []Step   `json:"steps"`        // 执行步骤
	//     Resources   []string `json:"resources"`    // 所需资源
	//     Timeline    string   `json:"timeline"`     // 时间线
	//     Constraints []string `json:"constraints"`  // 约束条件
	// }
	//
	// type Step struct {
	//     ID          string   `json:"id"`
	//     Description string   `json:"description"`
	//     Dependencies []string `json:"dependencies"`
	//     EstimatedTime string `json:"estimated_time"`
	// }

	fmt.Println("自定义计划结构示例：")
	fmt.Println(`{
  "goal": "完成项目开发",
  "steps": [
    {
      "id": "step1",
      "description": "需求分析",
      "dependencies": [],
      "estimated_time": "2天"
    },
    {
      "id": "step2",
      "description": "系统设计",
      "dependencies": ["step1"],
      "estimated_time": "3天"
    }
  ],
  "resources": ["开发团队", "测试环境"],
  "timeline": "2周",
  "constraints": ["预算限制", "技术栈限制"]
}`)
}

// 使用建议：
//
// 1. 三个核心智能体的配置：
//
//    Planner（规划智能体）：
//    - 使用 ToolCallingChatModel 生成结构化计划
//    - 可以自定义 ToolInfo 定义计划结构
//    - 可以自定义 GenInputFn 控制输入格式
//    - 可以自定义 NewPlan 函数使用自定义计划类型
//
//    Executor（执行智能体）：
//    - 配置必要的工具（搜索、计算、API 调用等）
//    - 设置合理的 MaxIterations（防止工具调用循环）
//    - 可以自定义 GenInputFn 控制输入格式
//    - 执行结果会自动传递给 Replanner
//
//    Replanner（重规划智能体）：
//    - 使用 ToolCallingChatModel（plan 或 respond 工具）
//    - 可以自定义 PlanTool 和 RespondTool
//    - 可以自定义 GenInputFn 控制输入格式
//    - 决定是继续执行还是完成任务
//
// 2. 执行流程：
//    第一阶段 - Planning：
//    - Planner 分析用户目标
//    - 生成结构化的执行计划（JSON 格式）
//    - 计划包含多个清晰的步骤
//
//    第二阶段 - Execution：
//    - Executor 执行计划的第一步
//    - 可以使用工具完成具体操作
//    - 记录执行结果
//
//    第三阶段 - Replanning：
//    - Replanner 评估已完成的步骤
//    - 判断目标是否达成
//    - 如果未完成：生成修订后的计划（只包含剩余步骤）
//    - 如果已完成：调用 respond 工具生成最终响应
//
//    循环执行第二、三阶段，直到任务完成或达到 MaxIterations
//
// 3. 适用场景判断：
//    ✓ 适合使用 Plan-Execute-Replan：
//    - 任务复杂，需要分解为多个步骤
//    - 初始信息不完整，需要边执行边调整
//    - 需要根据执行结果动态调整策略
//    - 任务有明确的完成标准
//
//    ✗ 不适合使用 Plan-Execute-Replan：
//    - 简单的单步任务（使用 ChatModelAgent）
//    - 固定流程的任务（使用 SequentialAgent）
//    - 需要人工干预的任务（使用 Human-in-the-Loop）
//
// 4. 性能考虑：
//    - 每轮迭代包含 Executor + Replanner 两次 LLM 调用
//    - 加上初始的 Planner 调用
//    - 总调用次数 = 1 (Planner) + 2 × 实际迭代次数
//    - 合理设置 MaxIterations（通常 5-10 次）
//
// 5. 与其他模式的对比：
//    - Plan-Execute-Replan：动态规划，适应性强
//    - SequentialAgent：固定流程，执行确定
//    - LoopAgent：迭代优化，关注改进
//    - SupervisorAgent：任务分发，多专家协作
//
// 6. 调试技巧：
//    - 检查 Planner 生成的计划是否合理
//    - 追踪每步执行的结果
//    - 观察 Replanner 的决策（继续或完成）
//    - 记录实际迭代次数
//    - 分析是否过早或过晚完成
//
// 7. 常见问题：
//    - 计划过于笼统：优化 Planner 的 Instruction
//    - 执行不到位：为 Executor 配置必要的工具
//    - 无法判断完成：明确完成标准
//    - 迭代次数过多：检查 Replanner 的判断逻辑
//
// 8. 最佳实践：
//    - Planner 应该生成具体、可执行的步骤
//    - Executor 应该配置足够的工具来完成任务
//    - Replanner 应该有明确的完成标准
//    - 设置合理的 MaxIterations（5-10 次）
//    - 每个步骤应该是独立可执行的
//    - 步骤之间应该有逻辑关联
//
// 9. 高级用法：
//    - 自定义计划结构（实现 Plan 接口）
//    - 自定义输入生成函数（GenInputFn）
//    - 自定义工具定义（ToolInfo）
//    - 结合其他模式（如在 Supervisor 中使用）
//
// 10. 典型应用模式：
//     - 研究报告：规划大纲 → 收集资料 → 撰写 → 补充
//     - 软件开发：需求 → 设计 → 编码 → 测试 → 修复
//     - 数据分析：目标 → 收集 → 分析 → 解读 → 深入
//     - 项目管理：计划 → 执行 → 监控 → 调整 → 完成
//     - 问题求解：分析 → 尝试 → 验证 → 改进 → 解决
