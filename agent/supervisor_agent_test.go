package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestSupervisorAgent 演示如何使用 SupervisorAgent（层级协调智能体）
//
// SupervisorAgent 实现了中心化的多智能体协调机制，由一个 Supervisor 智能体
// 协调多个专业子智能体，形成星型拓扑结构。
//
// 使用场景：
// - 多专家协作系统（需要中心协调者）
// - 任务动态分发（根据任务类型选择合适的专家）
// - 层级管理架构（构建多层级的智能体系统）
// - 集中控制决策（统一的决策点）
//
// 典型应用：
// - 客服系统：主管协调账户查询、订单处理、技术支持等专员
// - 金融顾问：协调市场分析师、风险评估师、投资顾问
// - 项目管理：协调需求分析、开发、测试、部署团队
// - 医疗诊断：协调症状分析、检查建议、治疗方案等专科医生
func TestSupervisorAgent(t *testing.T) {
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

	// 1. 创建子智能体：账户查询专员
	accountAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "account_agent",
		Description: "负责处理账户相关查询，如余额查询、账户信息等",
		Instruction: `你是一个账户查询专员。
负责处理账户相关的查询，包括：
- 账户余额查询
- 账户信息查询
- 交易历史查询

处理完成后，将结果报告给主管。
只回答账户相关的问题，其他问题请告知用户联系相应部门。

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建账户智能体失败: %v", err)
	}

	// 2. 创建子智能体：订单处理专员
	orderAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "order_agent",
		Description: "负责处理订单相关事务，如订单查询、订单修改、退款等",
		Instruction: `你是一个订单处理专员。
负责处理订单相关的事务，包括：
- 订单状态查询
- 订单修改
- 退款处理
- 物流跟踪

处理完成后，将结果报告给主管。
只处理订单相关的问题，其他问题请告知用户联系相应部门。

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建订单智能体失败: %v", err)
	}

	// 3. 创建子智能体：技术支持专员
	techSupportAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "tech_support_agent",
		Description: "负责处理技术问题，如产品使用、故障排查、功能咨询等",
		Instruction: `你是一个技术支持专员。
负责处理技术相关的问题，包括：
- 产品使用指导
- 故障排查
- 功能咨询
- 技术问题解答

处理完成后，将结果报告给主管。
只处理技术相关的问题，其他问题请告知用户联系相应部门。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建技术支持智能体失败: %v", err)
	}

	// 4. 创建 Supervisor 智能体（主管）
	supervisorAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "customer_service_supervisor",
		Description: "客服主管，负责协调各个专员处理用户请求",
		Instruction: `你是一个客服主管，管理三个专员：
- account_agent：处理账户相关问题（余额、账户信息等）
- order_agent：处理订单相关问题（订单查询、退款等）
- tech_support_agent：处理技术问题（产品使用、故障排查等）

你的职责：
1. 分析用户的请求
2. 判断应该由哪个专员处理
3. 将任务委派给相应的专员
4. 收集专员的处理结果
5. 向用户提供最终答复

重要规则：
- 一次只委派给一个专员
- 不要自己处理具体问题，要委派给专员
- 如果需要多个专员，按顺序逐个委派
- 收到专员的回复后，整理并回复用户
- 完成所有任务后，使用 exit 工具结束对话`,
		Model: chatModel,
		Exit:  &adk.ExitTool{}, // 重要：Supervisor 需要 Exit 工具来结束对话
	})
	if err != nil {
		t.Fatalf("创建主管智能体失败: %v", err)
	}

	// 5. 创建 SupervisorAgent
	multiAgent, err := NewSupervisorAgent(ctx, &SupervisorAgentConfig{
		Supervisor: supervisorAgent,
		SubAgents: []adk.Agent{
			accountAgent,
			orderAgent,
			techSupportAgent,
		},
	})
	if err != nil {
		t.Fatalf("创建 Supervisor 智能体失败: %v", err)
	}

	// 6. 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           multiAgent,
	})

	// 7. 发送查询
	query := "我想查询一下我的账户余额，另外我的订单什么时候能到？"
	fmt.Println("用户:", query)
	fmt.Println("\n开始处理...\n")

	iter := runner.Query(ctx, query)

	// 8. 处理响应
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 显示智能体切换
		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("\n→ 主管将任务委派给: %s\n\n", event.Action.TransferToAgent.DestAgentName)
		}

		// 显示输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Printf("[%s] %s\n", event.AgentName, msg.Content)
			}
		}

		// 检查是否退出
		if event.Action != nil && event.Action.Exit {
			fmt.Println("\n✓ 主管结束对话")
		}
	}

	fmt.Println("\n任务处理完成！")
}

// TestSupervisorAgentFinancialAdvisor 演示金融顾问场景
//
// 一个金融顾问主管协调多个专业分析师，为用户提供投资建议。
func TestSupervisorAgentFinancialAdvisor(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 市场分析师
	marketAnalystAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "market_analyst",
		Description: "市场分析师，分析市场趋势和行业动态",
		Instruction: `你是一个市场分析师。
分析市场趋势、行业动态、宏观经济环境。
提供专业的市场分析报告。
完成后向主管汇报。

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})

	// 风险评估师
	riskAnalystAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "risk_analyst",
		Description: "风险评估师，评估投资风险",
		Instruction: `你是一个风险评估师。
评估投资风险，包括：
- 市场风险
- 信用风险
- 流动性风险
- 操作风险

提供风险评估报告和风险控制建议。
完成后向主管汇报。

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})

	// 投资顾问
	investmentAdvisorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "investment_advisor",
		Description: "投资顾问，提供具体的投资建议",
		Instruction: `你是一个投资顾问。
根据市场分析和风险评估，提供具体的投资建议：
- 资产配置建议
- 投资产品推荐
- 投资时机建议
- 预期收益分析

完成后向主管汇报。

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})

	// 金融顾问主管
	supervisorAgent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "financial_supervisor",
		Description: "金融顾问主管，协调各专业分析师",
		Instruction: `你是一个金融顾问主管，管理三个专业分析师：
- market_analyst：市场分析师
- risk_analyst：风险评估师
- investment_advisor：投资顾问

处理流程：
1. 分析用户的投资需求
2. 先委派市场分析师分析市场
3. 再委派风险评估师评估风险
4. 最后委派投资顾问给出建议
5. 整合所有信息，给用户综合建议

注意：按顺序委派，一次一个。`,
		Model: chatModel,
		Exit:  &adk.ExitTool{},
	})

	multiAgent, _ := NewSupervisorAgent(ctx, &SupervisorAgentConfig{
		Supervisor: supervisorAgent,
		SubAgents: []adk.Agent{
			marketAnalystAgent,
			riskAnalystAgent,
			investmentAdvisorAgent,
		},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: multiAgent,
	})

	query := "我有 10 万元想投资，风险承受能力中等，请给我一些投资建议。"
	fmt.Println("用户:", query)
	fmt.Println("\n开始咨询...\n")

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

		if event.Action != nil && event.Action.TransferToAgent != nil {
			fmt.Printf("\n→ 委派给: %s\n", event.Action.TransferToAgent.DestAgentName)
		}

		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				fmt.Printf("\n[%s]\n%s\n", event.AgentName, msg.Content)
			}
		}
	}
}

// TestSupervisorAgentHierarchical 演示层级 Supervisor（Supervisor 嵌套）
//
// 可以构建多层级的 Supervisor 结构，实现更复杂的组织架构。
func TestSupervisorAgentHierarchical(t *testing.T) {
	t.Skip("这是一个示例，展示如何构建层级 Supervisor")

	// 层级结构示例：
	//
	// 总经理 (Top Supervisor)
	//   ├─ 销售部主管 (Sales Supervisor)
	//   │   ├─ 国内销售
	//   │   └─ 国际销售
	//   ├─ 技术部主管 (Tech Supervisor)
	//   │   ├─ 前端开发
	//   │   └─ 后端开发
	//   └─ 客服部主管 (Support Supervisor)
	//       ├─ 售前咨询
	//       └─ 售后服务

	fmt.Println("层级 Supervisor 结构：")
	fmt.Println("总经理")
	fmt.Println("  ├─ 销售部主管")
	fmt.Println("  │   ├─ 国内销售")
	fmt.Println("  │   └─ 国际销售")
	fmt.Println("  ├─ 技术部主管")
	fmt.Println("  │   ├─ 前端开发")
	fmt.Println("  │   └─ 后端开发")
	fmt.Println("  └─ 客服部主管")
	fmt.Println("      ├─ 售前咨询")
	fmt.Println("      └─ 售后服务")
}

// 使用建议：
//
// 1. Supervisor 设计：
//    - Supervisor 应该是决策者，不是执行者
//    - 明确 Supervisor 的职责：分析、分发、整合
//    - 必须配置 Exit 工具，用于结束对话
//    - Instruction 要清楚说明何时委派、委派给谁
//
// 2. 子智能体设计：
//    - 每个子智能体应该有明确的专业领域
//    - 子智能体之间职责不重叠
//    - 子智能体完成任务后应该"报告"给 Supervisor
//    - 子智能体不需要 Exit 工具（自动返回 Supervisor）
//
// 3. 任务委派策略：
//    - 一次委派一个子智能体（避免混乱）
//    - 按逻辑顺序委派（如果有依赖关系）
//    - Supervisor 收集所有结果后再回复用户
//    - 可以根据子智能体的回复决定下一步
//
// 4. 通信模式：
//    - 星型拓扑：所有通信都通过 Supervisor
//    - 子智能体之间不直接通信
//    - Supervisor 负责信息整合
//    - 用户只与 Supervisor 交互
//
// 5. 典型应用场景：
//    - 客服系统：主管 + 多个专业客服
//    - 咨询服务：总顾问 + 多个领域专家
//    - 项目管理：项目经理 + 各职能团队
//    - 医疗诊断：主治医生 + 各科室专家
//
// 6. 与其他模式的对比：
//    - Supervisor：中心化协调，动态任务分发
//    - Sequential：固定顺序执行，无动态决策
//    - Parallel：并行执行，无协调机制
//    - Loop：迭代优化，无任务分发
//
// 7. 性能考虑：
//    - 每次委派都需要一次 LLM 调用
//    - 总时间 = Supervisor 决策时间 + 所有子智能体执行时间
//    - 如果子智能体独立，考虑使用 Parallel
//    - 监控委派次数，避免过度委派
//
// 8. 调试技巧：
//    - 追踪 TransferToAgent 事件，了解委派流程
//    - 记录每个智能体的输入和输出
//    - 检查 Supervisor 的决策是否合理
//    - 验证子智能体是否正确返回 Supervisor
//
// 9. 常见问题：
//    - Supervisor 自己执行任务：明确 Instruction 要求委派
//    - 无限委派循环：设置明确的完成条件
//    - 子智能体不返回：确保子智能体配置正确
//    - 忘记 Exit：Supervisor 必须配置 Exit 工具
//
// 10. 最佳实践：
//     - Supervisor 的 Instruction 要详细说明委派规则
//     - 子智能体的 Description 要准确描述职责
//     - 使用清晰的命名（如 xxx_agent）
//     - 测试每个子智能体的独立功能
//     - 逐步增加子智能体数量（先 2-3 个）
//     - 考虑构建层级结构（Supervisor 嵌套）
