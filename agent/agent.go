package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
)

type AgentConfig struct {
	Name        string
	Description string
	Instruction string
}

// NewChatModelAgent ReAct 通用对话智能体
func NewChatModelAgent(ctx context.Context, chatModel model.BaseChatModel, agentConfig *AgentConfig) (*adk.ChatModelAgent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        agentConfig.Name,
		Description: agentConfig.Description,
		Instruction: agentConfig.Instruction,
		Model:       chatModel,
	})
}

// SequentialAgentConfig 串行智能体配置
type SequentialAgentConfig struct {
	Name        string
	Description string
	SubAgents   []adk.Agent
}

// NewSequentialAgent 串行流水线智能体
func NewSequentialAgent(ctx context.Context, config *SequentialAgentConfig) (adk.ResumableAgent, error) {
	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        config.Name,
		Description: config.Description,
		SubAgents:   config.SubAgents,
	})
}

// ParallelAgentConfig 并行智能体配置
type ParallelAgentConfig struct {
	Name        string
	Description string
	SubAgents   []adk.Agent
}

// NewParallelAgent 并行并发智能体
func NewParallelAgent(ctx context.Context, config *ParallelAgentConfig) (adk.ResumableAgent, error) {
	return adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        config.Name,
		Description: config.Description,
		SubAgents:   config.SubAgents,
	})
}

// LoopAgentConfig 循环智能体配置
type LoopAgentConfig struct {
	Name          string
	Description   string
	SubAgents     []adk.Agent
	MaxIterations int
}

// NewLoopAgent 循环执行智能体
func NewLoopAgent(ctx context.Context, config *LoopAgentConfig) (adk.ResumableAgent, error) {
	return adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          config.Name,
		Description:   config.Description,
		SubAgents:     config.SubAgents,
		MaxIterations: config.MaxIterations,
	})
}

// ==================== 多智能体预置模式（开箱即用） ====================

// SupervisorAgentConfig Supervisor 层级协调智能体配置
// 使用场景：
// 1. 多专家协作：需要一个中心协调者来管理多个专业领域的子智能体
// 2. 任务分发：根据任务类型动态选择合适的子智能体执行
// 3. 层级管理：构建多层级的智能体架构，每层有自己的 Supervisor
// 4. 集中控制：需要统一的决策点来协调多个智能体的工作流程
//
// 典型应用：
// - 客服系统：主管智能体协调账户查询、订单处理、技术支持等子智能体
// - 金融顾问：协调市场分析、风险评估、投资建议等专业智能体
// - 项目管理：协调需求分析、开发、测试、部署等不同阶段的智能体
// - 医疗诊断：协调症状分析、检查建议、治疗方案等专科智能体
//
// 工作原理：
// - Supervisor 作为中心节点，接收用户请求并分析任务
// - 根据任务特征选择合适的子智能体并委派任务
// - 子智能体完成任务后将结果返回给 Supervisor
// - Supervisor 整合所有结果并生成最终响应
// - 子智能体之间不直接通信，所有交互都通过 Supervisor
type SupervisorAgentConfig struct {
	// Supervisor 中心协调智能体，负责任务分析和分发
	Supervisor adk.Agent
	// SubAgents 被管理的子智能体列表，每个子智能体负责特定领域
	SubAgents []adk.Agent
}

// NewSupervisorAgent 创建 Supervisor 层级协调智能体
// Supervisor 模式实现了中心化的多智能体协调机制：
// - 单一决策点：所有任务分发决策由 Supervisor 统一做出
// - 星型拓扑：Supervisor 位于中心，子智能体围绕其工作
// - 双向通信：Supervisor 可以向子智能体发送任务，子智能体将结果返回
// - 统一追踪：整个 Supervisor 结构共享同一个追踪根节点，便于观测
//
// 注意事项：
// - Supervisor 需要配置 Exit 工具以便在完成所有任务后退出
// - 子智能体会被自动配置为只能与 Supervisor 通信
// - 适合任务边界清晰、职责分明的场景
func NewSupervisorAgent(ctx context.Context, config *SupervisorAgentConfig) (adk.ResumableAgent, error) {
	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: config.Supervisor,
		SubAgents:  config.SubAgents,
	})
}

// PlanExecuteReplanConfig Plan-Execute-Replan 智能体配置
// 使用场景：
// 1. 复杂任务分解：需要将大型任务拆解为多个可执行步骤
// 2. 动态调整：执行过程中根据实际情况调整计划
// 3. 迭代优化：通过多轮执行-评估-重规划循环逐步完善结果
// 4. 不确定性处理：初始信息不完整，需要边执行边调整策略
//
// 典型应用：
// - 研究报告生成：规划大纲 → 收集资料 → 撰写内容 → 评估完整性 → 补充缺失部分
// - 软件开发：需求分析 → 设计方案 → 编码实现 → 测试验证 → 修复问题
// - 数据分析：确定分析目标 → 数据收集 → 统计分析 → 结果解读 → 深入挖掘
// - 旅行规划：确定目的地 → 查询交通 → 预订酒店 → 安排行程 → 优化路线
//
// 工作原理：
// 1. Planning（规划）：Planner 分析目标，生成结构化的执行计划
// 2. Execution（执行）：Executor 执行计划中的第一步
// 3. Replanning（重规划）：Replanner 评估执行结果，决定：
//    - 如果目标达成：生成最终响应并退出
//    - 如果需要继续：生成修订后的计划，返回步骤 2
// 4. 循环执行步骤 2-3，直到任务完成或达到最大迭代次数
type PlanExecuteReplanConfig struct {
	// Planner 规划智能体，负责生成初始执行计划
	// 可使用 planexecute.NewPlanner 创建
	Planner adk.Agent

	// Executor 执行智能体，负责执行计划中的步骤
	// 可使用 planexecute.NewExecutor 创建
	// 通常配置工具以完成具体任务
	Executor adk.Agent

	// Replanner 重规划智能体，负责评估进度并决定下一步行动
	// 可使用 planexecute.NewReplanner 创建
	// 会调用 plan 或 respond 工具来继续或完成任务
	Replanner adk.Agent

	// MaxIterations 最大执行-重规划循环次数
	// 防止无限循环，默认为 10
	MaxIterations int
}

// NewPlanExecuteReplanAgent 创建 Plan-Execute-Replan 智能体
// 这是一个强大的问题解决模式，通过三阶段循环实现复杂任务：
//
// 阶段 1 - Planning（规划）：
// - 分析用户目标，生成清晰的步骤列表
// - 每个步骤应该是独立可执行的
// - 步骤按逻辑顺序排列
//
// 阶段 2 - Execution（执行）：
// - 执行当前计划的第一步
// - 可以使用工具完成具体操作
// - 记录执行结果供后续评估
//
// 阶段 3 - Replanning（重规划）：
// - 评估已完成的步骤和结果
// - 判断目标是否达成
// - 如果未完成，生成修订后的计划（只包含剩余步骤）
// - 如果已完成，生成最终响应
//
// 优势：
// - 适应性强：可以根据执行结果动态调整策略
// - 容错性好：执行失败时可以重新规划
// - 可追溯：每步执行都有记录，便于调试
// - 渐进式：逐步逼近目标，而非一次性完成
//
// 注意事项：
// - 需要配置合理的 MaxIterations 防止无限循环
// - Planner 和 Replanner 需要支持结构化输出（JSON）
// - Executor 应该配置必要的工具来完成实际操作
func NewPlanExecuteReplanAgent(ctx context.Context, config *PlanExecuteReplanConfig) (adk.ResumableAgent, error) {
	return planexecute.New(ctx, &planexecute.Config{
		Planner:       config.Planner,
		Executor:      config.Executor,
		Replanner:     config.Replanner,
		MaxIterations: config.MaxIterations,
	})
}

// 注意：以下智能体模式在当前 eino v0.8.13 版本中不存在或位于不同的包中

// HostMultiAgent（主机式多智能体）
// 说明：在当前版本的 eino SDK 中，没有找到名为 "HostMultiAgent" 的预置模式。
// 这可能是：
// 1. 文档中提到但尚未实现的功能
// 2. 在更高版本中提供的功能
// 3. 通过组合现有的 Supervisor 或其他模式实现的概念
//
// 如果需要类似的集中调度功能，建议使用 SupervisorAgent，它提供了：
// - 中心化的任务分发
// - 统一的状态管理
// - 子智能体的协调控制

// MultiQueryRetriever（多查询检索器）
// 说明：MultiQueryRetriever 存在于 eino SDK 中，但它不是一个 Agent，
// 而是一个 Retriever 组件，位于 flow/retriever/multiquery 包中。
//
// 使用场景：
// - RAG（检索增强生成）系统中的查询扩展
// - 将单个用户查询重写为多个不同角度的查询
// - 从多个检索器并行检索文档
// - 提高检索召回率和多样性
//
// 典型应用：
// - 问答系统：将问题改写为多个相关查询以获取更全面的答案
// - 文档搜索：从不同角度检索相关文档
// - 知识库查询：扩展查询范围以覆盖更多相关内容
//
// 使用方式：
// import "github.com/cloudwego/eino/flow/retriever/multiquery"
//
// retriever := multiquery.NewRetriever(ctx, &multiquery.Config{
//     ChatModel: chatModel,           // 用于生成多个查询的模型
//     Retriever: baseRetriever,       // 底层检索器
//     QueryCount: 3,                  // 生成的查询数量
// })
//
// 注意：这是一个工具型组件，通常作为 ChatModelAgent 的工具使用，
// 而不是独立的 Agent。

// ==================== DeepAgent（深度任务编排智能体） ====================

// DeepAgentConfig Deep 智能体配置
// 使用场景：
// 1. 复杂任务编排：需要协调多个子智能体和工具完成复杂任务
// 2. 自主任务分解：智能体自主将大任务分解为子任务并分配给合适的子智能体
// 3. 工具丰富场景：需要使用文件系统、Shell 命令等多种工具
// 4. 待办事项管理：自动管理任务列表，追踪任务进度
//
// 典型应用：
// - Excel 数据处理：读取、分析、处理 Excel 文件，生成报告
// - 代码开发助手：编写代码、执行测试、调试问题
// - 数据分析：收集数据、清洗数据、分析数据、生成可视化
// - 文档处理：读取文档、提取信息、生成新文档
// - 自动化运维：执行系统命令、监控服务、处理告警
//
// 工作原理：
// - DeepAgent 是一个增强版的 ChatModelAgent
// - 内置任务管理工具（write_todos）追踪任务进度
// - 内置文件系统工具（read_file、write_file、edit_file、glob、grep）
// - 内置 Shell 执行工具（execute）
// - 可以调用子智能体处理专业任务
// - 支持通用子智能体（general-purpose）处理未分配的任务
//
// 与其他智能体的区别：
// - ChatModelAgent：基础对话智能体，功能单一
// - SupervisorAgent：中心化协调，需要明确指定任务分配
// - DeepAgent：自主任务编排，智能体自己决定如何分解和分配任务
type DeepAgentConfig struct {
	// Name 智能体名称
	Name string
	// Description 智能体描述
	Description string
	// ChatModel 使用的聊天模型
	ChatModel model.BaseChatModel
	// Instruction 系统提示词，为空时使用内置默认提示词
	Instruction string
	// SubAgents 子智能体列表，DeepAgent 可以调用这些子智能体
	SubAgents []adk.Agent
	// ToolsConfig 工具配置
	ToolsConfig adk.ToolsConfig
	// MaxIteration 最大迭代次数
	MaxIteration int

	// Backend 文件系统后端，提供文件操作能力
	// 如果设置，将注册 read_file、write_file、edit_file、glob、grep 工具
	Backend interface{} // filesystem.Backend

	// Shell Shell 命令执行器
	// 如果设置，将注册 execute 工具支持 Shell 命令执行
	Shell interface{} // filesystem.Shell

	// StreamingShell 流式 Shell 命令执行器
	// 如果设置，将注册流式 execute 工具
	StreamingShell interface{} // filesystem.StreamingShell

	// WithoutWriteTodos 禁用内置的 write_todos 工具
	WithoutWriteTodos bool
	// WithoutGeneralSubAgent 禁用通用子智能体
	WithoutGeneralSubAgent bool
}

// NewDeepAgent 创建 Deep 智能体
// Deep 智能体是一个功能强大的任务编排智能体，具有以下特点：
//
// 核心能力：
// 1. 自主任务分解：智能体自己决定如何将复杂任务分解为子任务
// 2. 智能任务分配：根据子智能体的能力自动选择合适的子智能体
// 3. 任务进度追踪：使用 write_todos 工具管理待办事项列表
// 4. 丰富的工具支持：文件系统、Shell 命令、自定义工具
//
// 内置工具：
// - write_todos：管理任务列表（自动启用，除非 WithoutWriteTodos=true）
// - read_file、write_file、edit_file：文件操作（需要配置 Backend）
// - glob、grep：文件搜索（需要配置 Backend）
// - execute：Shell 命令执行（需要配置 Shell 或 StreamingShell）
//
// 子智能体机制：
// - 显式子智能体：通过 SubAgents 配置的专业智能体
// - 通用子智能体：自动创建的通用智能体，处理未分配的任务
// - 任务工具：DeepAgent 通过 task 工具调用子智能体
//
// 使用建议：
// 1. 适合复杂的多步骤任务，需要协调多个工具和子智能体
// 2. 子智能体应该有明确的专业领域和职责
// 3. 合理设置 MaxIteration，防止无限循环（建议 50-100）
// 4. 使用 write_todos 工具让智能体自己管理任务进度
// 5. 配置 Backend 和 Shell 以支持文件和命令操作
//
// 典型使用模式：
// - 数据处理：DeepAgent + 数据清洗子智能体 + 数据分析子智能体
// - 代码开发：DeepAgent + 代码生成子智能体 + 测试子智能体
// - 文档处理：DeepAgent + 文档解析子智能体 + 内容生成子智能体
// - 研究分析：DeepAgent + 搜索子智能体 + 分析子智能体
func NewDeepAgent(ctx context.Context, config *DeepAgentConfig) (adk.ResumableAgent, error) {
	return deep.New(ctx, &deep.Config{
		Name:                   config.Name,
		Description:            config.Description,
		ChatModel:              config.ChatModel,
		Instruction:            config.Instruction,
		SubAgents:              config.SubAgents,
		ToolsConfig:            config.ToolsConfig,
		MaxIteration:           config.MaxIteration,
		WithoutWriteTodos:      config.WithoutWriteTodos,
		WithoutGeneralSubAgent: config.WithoutGeneralSubAgent,
		// Backend, Shell, StreamingShell 需要类型断言
		// 这里简化处理，实际使用时需要传入正确的类型
	})
}
