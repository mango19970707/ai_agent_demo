package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
)

// TestParallelAgent 演示如何使用 ParallelAgent（并行并发智能体）
//
// ParallelAgent 同时执行多个子智能体，所有子智能体并发运行，互不依赖。
//
// 使用场景：
// - 多源数据收集（同时从多个 API 获取数据）
// - 多角度分析（同时进行技术分析、市场分析、风险分析）
// - 并行处理（同时处理多个独立任务）
// - 多专家评审（多个专家同时评审同一内容）
//
// 典型应用：
// - 市场研究：同时收集股票数据、新闻资讯、社交媒体情绪
// - 内容审核：同时进行敏感词检测、情感分析、事实核查
// - 多语言翻译：同时翻译成多种语言
// - 竞品分析：同时分析多个竞争对手的产品
func TestParallelAgent(t *testing.T) {
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

	// 1. 创建子智能体：股票数据收集器
	stockAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "stock_collector",
		Description: "收集股票市场数据",
		Instruction: `你是一个股票数据分析师。
根据用户提供的公司名称，提供以下信息：
- 当前股价趋势
- 市盈率（P/E）
- 市值
- 近期重要财报数据

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建股票智能体失败: %v", err)
	}

	// 2. 创建子智能体：新闻资讯收集器
	newsAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "news_collector",
		Description: "收集相关新闻资讯",
		Instruction: `你是一个新闻分析师。
根据用户提供的公司名称，总结：
- 近期重要新闻（3-5条）
- 行业动态
- 公司战略变化
- 市场反应

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建新闻智能体失败: %v", err)
	}

	// 3. 创建子智能体：社交媒体情绪分析器
	sentimentAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "sentiment_analyzer",
		Description: "分析社交媒体情绪",
		Instruction: `你是一个社交媒体情绪分析师。
根据用户提供的公司名称，分析：
- 社交媒体整体情绪（正面/中性/负面）
- 主要讨论话题
- 用户关注点
- 情绪变化趋势

注意：这是模拟数据，仅用于演示。`,
		Model: chatModel,
	})
	if err != nil {
		t.Fatalf("创建情绪分析智能体失败: %v", err)
	}

	// 4. 创建 ParallelAgent，三个子智能体并行执行
	parallelAgent, err := NewParallelAgent(ctx, &ParallelAgentConfig{
		Name:        "market_research",
		Description: "市场研究：并行收集股票数据、新闻资讯、社交媒体情绪",
		SubAgents: []adk.Agent{
			stockAgent,     // 并行任务 1：股票数据
			newsAgent,      // 并行任务 2：新闻资讯
			sentimentAgent, // 并行任务 3：情绪分析
		},
	})
	if err != nil {
		t.Fatalf("创建并行智能体失败: %v", err)
	}

	// 5. 创建 Runner 并执行
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           parallelAgent,
	})

	// 6. 发送查询
	query := "请帮我分析一下特斯拉（Tesla）公司的市场情况"
	fmt.Println("用户:", query)
	fmt.Println("\n开始并行收集数据...\n")

	iter := runner.Query(ctx, query)

	// 7. 处理响应
	// 注意：并行执行的结果可能以任意顺序返回
	results := make(map[string]string)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			t.Fatalf("执行出错: %v", event.Err)
		}

		// 收集每个子智能体的输出
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				results[event.AgentName] = msg.Content
				fmt.Printf("✓ %s 完成\n", event.AgentName)
			}
		}
	}

	// 8. 显示所有结果
	fmt.Println("\n=== 收集到的数据 ===\n")
	for agentName, result := range results {
		fmt.Printf("[%s]\n%s\n\n", agentName, result)
	}

	fmt.Println("并行数据收集完成！")
}

// TestParallelAgentMultiLanguageTranslation 演示多语言并行翻译
//
// 这是 ParallelAgent 的典型应用场景：同时将内容翻译成多种语言。
func TestParallelAgentMultiLanguageTranslation(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	ctx := context.Background()

	chatModel, _ := openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: "your-api-key-here",
	})

	// 创建多个翻译智能体
	englishTranslator, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "english_translator",
		Description: "翻译成英文",
		Instruction: "将提供的中文内容翻译成英文，保持原意，使用地道的表达。",
		Model:       chatModel,
	})

	japaneseTranslator, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "japanese_translator",
		Description: "翻译成日文",
		Instruction: "将提供的中文内容翻译成日文，保持原意，使用地道的表达。",
		Model:       chatModel,
	})

	koreanTranslator, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "korean_translator",
		Description: "翻译成韩文",
		Instruction: "将提供的中文内容翻译成韩文，保持原意，使用地道的表达。",
		Model:       chatModel,
	})

	frenchTranslator, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "french_translator",
		Description: "翻译成法文",
		Instruction: "将提供的中文内容翻译成法文，保持原意，使用地道的表达。",
		Model:       chatModel,
	})

	// 创建并行翻译智能体
	parallelAgent, _ := NewParallelAgent(ctx, &ParallelAgentConfig{
		Name:        "multi_language_translator",
		Description: "多语言并行翻译器",
		SubAgents: []adk.Agent{
			englishTranslator,
			japaneseTranslator,
			koreanTranslator,
			frenchTranslator,
		},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: parallelAgent,
	})

	// 要翻译的中文内容
	chineseText := "人工智能正在改变我们的生活方式，从智能手机到自动驾驶汽车，AI 技术无处不在。"

	fmt.Println("原文（中文）:", chineseText)
	fmt.Println("\n开始并行翻译...\n")

	iter := runner.Query(ctx, chineseText)

	translations := make(map[string]string)

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
				translations[event.AgentName] = msg.Content
			}
		}
	}

	// 显示翻译结果
	fmt.Println("=== 翻译结果 ===\n")
	languageMap := map[string]string{
		"english_translator":  "英文",
		"japanese_translator": "日文",
		"korean_translator":   "韩文",
		"french_translator":   "法文",
	}

	for agentName, translation := range translations {
		fmt.Printf("%s: %s\n\n", languageMap[agentName], translation)
	}
}

// TestParallelAgentPerformanceComparison 演示并行执行的性能优势
//
// 对比串行执行和并行执行的时间差异。
func TestParallelAgentPerformanceComparison(t *testing.T) {
	t.Skip("这是一个示例，需要配置 API Key 才能运行")

	// 这个测试展示了并行执行的性能优势
	// 假设每个子智能体需要 2 秒执行：
	//
	// 串行执行（SequentialAgent）：
	// - 总时间 = 2s + 2s + 2s = 6s
	//
	// 并行执行（ParallelAgent）：
	// - 总时间 ≈ 2s（所有任务同时执行）
	//
	// 性能提升：3倍

	fmt.Println("并行执行的性能优势：")
	fmt.Println("- 3个任务，每个需要2秒")
	fmt.Println("- 串行执行：6秒")
	fmt.Println("- 并行执行：2秒")
	fmt.Println("- 性能提升：3倍")
}

// 使用建议：
//
// 1. 适用场景判断：
//    - ✓ 子任务之间完全独立，没有依赖关系
//    - ✓ 需要从多个数据源收集信息
//    - ✓ 需要多个专家同时评审
//    - ✗ 子任务之间有依赖关系（应使用 SequentialAgent）
//    - ✗ 需要根据前一个任务的结果决定下一步（应使用 SequentialAgent 或 LoopAgent）
//
// 2. 性能优化：
//    - 并行执行可以显著减少总执行时间
//    - 总时间约等于最慢的子智能体的执行时间
//    - 适合 I/O 密集型任务（如 API 调用、数据库查询）
//    - 注意并发数量，避免超过 API 限流
//
// 3. 结果处理：
//    - 并行执行的结果可能以任意顺序返回
//    - 使用 event.AgentName 识别结果来源
//    - 可以使用 map 收集所有结果后再处理
//    - 考虑部分失败的情况（某些子智能体可能失败）
//
// 4. 错误处理：
//    - 如果某个子智能体失败，不会影响其他子智能体
//    - 需要检查每个子智能体的执行结果
//    - 考虑设置超时时间，避免某个子智能体卡住
//
// 5. 资源管理：
//    - 并行执行会同时占用多个 API 配额
//    - 注意 API 的并发限制和速率限制
//    - 考虑使用连接池管理资源
//    - 监控内存使用情况
//
// 6. 典型应用模式：
//    - 多源数据聚合：同时从多个 API 获取数据
//    - 多角度分析：技术、市场、风险等多维度分析
//    - 多专家评审：多个专家同时评审同一内容
//    - 批量处理：同时处理多个独立的任务
//    - A/B 测试：同时运行多个版本的智能体进行对比
//
// 7. 与 SequentialAgent 的对比：
//    - ParallelAgent：任务独立，追求速度
//    - SequentialAgent：任务有依赖，追求逻辑
//    - 可以组合使用：先并行收集数据，再串行处理
//
// 8. 调试技巧：
//    - 先单独测试每个子智能体
//    - 使用日志记录每个子智能体的开始和结束时间
//    - 监控并发执行的资源使用情况
//    - 考虑添加超时保护机制
