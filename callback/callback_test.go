/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package callback

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ============================================================================
// 最佳实践总结
// ============================================================================

/*
Callback 最佳实践：

1. OnStart (组件开始执行前)
   - 记录入参：保存请求参数用于调试和审计
   - 鉴权：验证用户权限，拦截未授权请求
   - 限流：实现 QPS 限制，防止滥用
   - 敏感词过滤：拦截包含敏感内容的输入

2. OnEnd (组件执行成功结束)
   - 记录出参：保存响应结果用于分析
   - 耗时统计：监控性能，识别慢查询
   - Token 统计：用于成本核算和计费
   - 会话历史：保存对话记录到数据库

3. OnError (组件执行报错)
   - 错误日志：记录详细的错误信息
   - 告警通知：集成钉钉、企业微信等告警系统
   - 异常兜底：提供降级方案，保证服务可用性
   - 故障上报：上报到监控系统（如 Prometheus）

4. OnLLMStart (LLM 开始调用)
   - 记录 Prompt：保存完整的输入消息
   - 拦截敏感输入：过滤违规内容
   - 参数校验：验证模型参数是否合法

5. OnLLMEnd (LLM 响应结束)
   - 统计 Token：记录输入/输出 Token 数量
   - 记录完整问答：保存 Q&A 用于训练和分析
   - 计费：根据 Token 使用量计算成本

6. OnLLMStream (流式每分片输出)
   - 实时流式日志：记录每个分片的内容
   - 实时内容过滤：检测并过滤敏感内容
   - 性能监控：统计流式输出的延迟

7. OnToolStart (工具开始调用)
   - 校验工具参数：验证参数格式和取值范围
   - 权限校验：检查用户是否有权限调用该工具
   - 审计：记录工具调用日志用于审计

8. OnToolEnd (工具执行完成)
   - 记录工具返回结果：保存工具输出
   - 耗时统计：监控工具执行性能
   - 审计留痕：记录完整的调用链路

9. OnRetrieverStart (检索开始)
   - 记录检索 Query：保存用户的检索请求
   - 拦截非法检索：过滤恶意查询
   - 参数校验：验证 TopK 等参数

10. OnRetrieverEnd (检索结束)
    - 记录检索召回文档：保存召回的文档列表
    - RAG 链路分析：分析召回质量和相关性
    - 性能监控：统计检索耗时

集成建议：
- 使用 CozeLoop 实现链路追踪和可观测性
- 使用 OpenTelemetry 对接标准可观测性协议
- 使用 Prometheus 收集指标数据
- 使用 ELK/Loki 收集和分析日志
*/

// ============================================================================
// 1. OnStart Callback - 组件开始执行前
// 使用场景：记录入参、鉴权、限流、打印开始日志
// ============================================================================

// OnStartHandler 在组件开始执行前触发，用于记录入参、鉴权、限流等
// 注意：不需要嵌入 HandlerHelper，直接使用 NewHandlerBuilder 构建

func NewOnStartHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info == nil {
				return ctx
			}

			// 记录开始时间，用于后续计算耗时
			ctx = context.WithValue(ctx, "start_time", time.Now())

			switch info.Component {
			case components.ComponentOfChatModel:
				// LLM 开始调用：记录 Prompt
				mci := model.ConvCallbackInput(input)
				fmt.Printf("[OnStart] ChatModel 开始调用 | 模型: %s | 消息数: %d\n",
					info.Type, len(mci.Messages))

				// 可以在这里实现敏感词过滤、权限校验等
				for _, msg := range mci.Messages {
					if strings.Contains(msg.Content, "敏感词") {
						fmt.Printf("[OnStart] ⚠️  检测到敏感输入，已拦截\n")
					}
				}

			case components.ComponentOfTool:
				// 工具开始调用：校验工具参数、权限校验
				tci := tool.ConvCallbackInput(input)
				fmt.Printf("[OnStart] Tool 开始调用 | 工具名: %s | 参数: %s\n",
					info.Name, tci.ArgumentsInJSON)

				// 可以在这里实现工具权限校验、参数验证等
				// 例如：检查用户是否有权限调用该工具

			case components.ComponentOfRetriever:
				// 检索开始：记录检索 Query、拦截非法检索
				rci := retriever.ConvCallbackInput(input)
				fmt.Printf("[OnStart] Retriever 开始检索 | Query: %s | TopK: %d\n",
					rci.Query, rci.TopK)

				// 可以在这里实现检索权限控制、非法查询拦截等
				if len(rci.Query) > 1000 {
					fmt.Printf("[OnStart] ⚠️  检索 Query 过长，已拦截\n")
				}

			default:
				fmt.Printf("[OnStart] %s/%s 开始执行\n", info.Component, info.Name)
			}

			return ctx
		}).
		Build()
}

// TestOnStartCallback 测试 OnStart 回调
func TestOnStartCallback(t *testing.T) {
	// 注册全局回调
	callbacks.AppendGlobalHandlers(NewOnStartHandler())

	fmt.Println("\n=== 测试 OnStart Callback ===")
	fmt.Println("场景：组件开始执行前记录入参、鉴权、限流")

	// 模拟调用会触发 OnStart 回调
	// 实际使用中，当调用 ChatModel、Tool、Retriever 等组件时会自动触发
}

// ============================================================================
// 2. OnEnd Callback - 组件执行成功结束
// 使用场景：记录出参、耗时统计、保存会话历史
// ============================================================================

// OnEndHandler 在组件执行成功结束时触发，用于记录出参、耗时统计等
// 注意：不需要嵌入 HandlerHelper，直接使用 NewHandlerBuilder 构建

func NewOnEndHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info == nil {
				return ctx
			}

			// 计算耗时
			startTime, ok := ctx.Value("start_time").(time.Time)
			var duration time.Duration
			if ok {
				duration = time.Since(startTime)
			}

			switch info.Component {
			case components.ComponentOfChatModel:
				// LLM 响应结束：统计 Token、记录完整问答、计费
				mco := model.ConvCallbackOutput(output)
				fmt.Printf("[OnEnd] ChatModel 调用完成 | 耗时: %v | 响应: %s\n",
					duration, truncateString(mco.Message.Content, 50))

				// 统计 Token 使用情况（用于计费）
				if mco.TokenUsage != nil {
					fmt.Printf("[OnEnd] Token 统计 | 输入: %d | 输出: %d | 总计: %d\n",
						mco.TokenUsage.PromptTokens,
						mco.TokenUsage.CompletionTokens,
						mco.TokenUsage.TotalTokens)
				}

				// 可以在这里保存会话历史到数据库

			case components.ComponentOfTool:
				// 工具执行完成：记录工具返回结果、耗时、审计留痕
				tco := tool.ConvCallbackOutput(output)
				fmt.Printf("[OnEnd] Tool 执行完成 | 工具名: %s | 耗时: %v | 结果: %s\n",
					info.Name, duration, truncateString(tco.Response, 50))

				// 可以在这里记录审计日志

			case components.ComponentOfRetriever:
				// 检索结束：记录检索召回文档、RAG 链路分析
				_ = retriever.ConvCallbackOutput(output)
				fmt.Printf("[OnEnd] Retriever 检索完成 | 耗时: %v\n", duration)

				// 可以在这里记录召回文档信息用于 RAG 链路分析
				// 注意：具体的文档结构取决于 retriever 的实现

			default:
				fmt.Printf("[OnEnd] %s/%s 执行完成 | 耗时: %v\n",
					info.Component, info.Name, duration)
			}

			return ctx
		}).
		Build()
}

// TestOnEndCallback 测试 OnEnd 回调
func TestOnEndCallback(t *testing.T) {
	// 注册全局回调
	callbacks.AppendGlobalHandlers(NewOnEndHandler())

	fmt.Println("\n=== 测试 OnEnd Callback ===")
	fmt.Println("场景：组件执行成功结束后记录出参、耗时统计、保存会话历史")
}

// ============================================================================
// 3. OnError Callback - 组件执行报错
// 使用场景：错误日志、告警、异常兜底、故障上报
// ============================================================================

// OnErrorHandler 在组件执行报错时触发，用于错误日志、告警、故障上报等
// 注意：不需要嵌入 HandlerHelper，直接使用 NewHandlerBuilder 构建

func NewOnErrorHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if info == nil {
				fmt.Printf("[OnError] ❌ 发生错误: %v\n", err)
				return ctx
			}

			// 计算耗时
			startTime, ok := ctx.Value("start_time").(time.Time)
			var duration time.Duration
			if ok {
				duration = time.Since(startTime)
			}

			// 记录错误日志
			fmt.Printf("[OnError] ❌ %s/%s 执行失败 | 耗时: %v | 错误: %v\n",
				info.Component, info.Name, duration, err)

			// 根据组件类型进行不同的错误处理
			switch info.Component {
			case components.ComponentOfChatModel:
				fmt.Printf("[OnError] LLM 调用失败，可能原因：API 限流、网络超时、余额不足\n")
				// 可以在这里实现：
				// 1. 发送告警通知
				// 2. 切换到备用模型
				// 3. 记录到监控系统

			case components.ComponentOfTool:
				fmt.Printf("[OnError] 工具执行失败，工具名: %s\n", info.Name)
				// 可以在这里实现：
				// 1. 返回兜底结果
				// 2. 重试机制
				// 3. 故障上报

			case components.ComponentOfRetriever:
				fmt.Printf("[OnError] 检索失败，可能原因：向量库连接失败、索引不存在\n")
				// 可以在这里实现：
				// 1. 降级到关键词检索
				// 2. 返回缓存结果
			}

			// 可以在这里集成告警系统（如钉钉、企业微信、PagerDuty）
			// sendAlert(info.Component, info.Name, err)

			return ctx
		}).
		Build()
}

// TestOnErrorCallback 测试 OnError 回调
func TestOnErrorCallback(t *testing.T) {
	// 注册全局回调
	callbacks.AppendGlobalHandlers(NewOnErrorHandler())

	fmt.Println("\n=== 测试 OnError Callback ===")
	fmt.Println("场景：组件执行报错时记录错误日志、告警、异常兜底")
}

// ============================================================================
// 4. OnEndWithStreamOutput Callback - 流式输出
// 使用场景：实时流式日志、实时内容过滤
// ============================================================================

// OnStreamHandler 在流式输出时触发，用于实时日志、实时内容过滤等
// 注意：不需要嵌入 HandlerHelper，直接使用 NewHandlerBuilder 构建

func NewOnStreamHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			if info == nil {
				output.Close()
				return ctx
			}

			fmt.Printf("[OnStream] %s/%s 开始流式输出\n", info.Component, info.Name)

			// 创建一个新的 StreamReader 用于实际消费
			go func() {
				defer output.Close()

				chunkCount := 0
				var fullContent strings.Builder

				for {
					chunk, err := output.Recv()
					if err != nil {
						if err.Error() != "EOF" {
							fmt.Printf("[OnStream] ❌ 流式读取错误: %v\n", err)
						}
						break
					}

					chunkCount++

					// 根据组件类型处理流式输出
					switch info.Component {
					case components.ComponentOfChatModel:
						// LLM 流式输出：实时内容过滤
						mco := model.ConvCallbackOutput(chunk)
						content := mco.Message.Content
						fullContent.WriteString(content)

						// 实时内容过滤（敏感词检测）
						if strings.Contains(content, "敏感词") {
							fmt.Printf("[OnStream] ⚠️  检测到敏感内容，已过滤\n")
							continue
						}

						fmt.Printf("[OnStream] Chunk %d: %s\n", chunkCount, content)

					default:
						fmt.Printf("[OnStream] Chunk %d 已接收\n", chunkCount)
					}
				}

				fmt.Printf("[OnStream] 流式输出完成 | 总分片数: %d | 总内容长度: %d\n",
					chunkCount, fullContent.Len())
			}()

			return ctx
		}).
		Build()
}

// TestOnStreamCallback 测试流式输出回调
func TestOnStreamCallback(t *testing.T) {
	// 注册全局回调
	callbacks.AppendGlobalHandlers(NewOnStreamHandler())

	fmt.Println("\n=== 测试 OnStream Callback ===")
	fmt.Println("场景：流式输出时实时记录日志、实时内容过滤")
}

// ============================================================================
// 5. 综合示例：组合多个 Callback
// 使用场景：完整的可观测性方案
// ============================================================================

// ComprehensiveHandler 综合回调处理器，实现完整的可观测性
// 注意：不需要嵌入 HandlerHelper，直接使用 NewHandlerBuilder 构建

func NewComprehensiveHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		// OnStart: 记录入参、鉴权、限流
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info == nil {
				return ctx
			}

			ctx = context.WithValue(ctx, "start_time", time.Now())
			ctx = context.WithValue(ctx, "trace_id", generateTraceID())

			traceID := ctx.Value("trace_id").(string)

			switch info.Component {
			case components.ComponentOfChatModel:
				mci := model.ConvCallbackInput(input)
				fmt.Printf("[%s][OnLLMStart] 模型: %s | 消息数: %d\n",
					traceID, info.Type, len(mci.Messages))

			case components.ComponentOfTool:
				tci := tool.ConvCallbackInput(input)
				fmt.Printf("[%s][OnToolStart] 工具: %s | 参数: %s\n",
					traceID, info.Name, tci.ArgumentsInJSON)

			case components.ComponentOfRetriever:
				rci := retriever.ConvCallbackInput(input)
				fmt.Printf("[%s][OnRetrieverStart] Query: %s | TopK: %d\n",
					traceID, rci.Query, rci.TopK)
			}

			return ctx
		}).
		// OnEnd: 记录出参、耗时统计
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info == nil {
				return ctx
			}

			startTime, _ := ctx.Value("start_time").(time.Time)
			traceID, _ := ctx.Value("trace_id").(string)
			duration := time.Since(startTime)

			switch info.Component {
			case components.ComponentOfChatModel:
				mco := model.ConvCallbackOutput(output)
				tokenCount := 0
				if mco.TokenUsage != nil {
					tokenCount = mco.TokenUsage.TotalTokens
				}
				fmt.Printf("[%s][OnLLMEnd] 耗时: %v | Token: %d\n",
					traceID, duration, tokenCount)

			case components.ComponentOfTool:
				tco := tool.ConvCallbackOutput(output)
				fmt.Printf("[%s][OnToolEnd] 耗时: %v | 结果长度: %d\n",
					traceID, duration, len(tco.Response))

			case components.ComponentOfRetriever:
				_ = retriever.ConvCallbackOutput(output)
				fmt.Printf("[%s][OnRetrieverEnd] 耗时: %v\n",
					traceID, duration)
			}

			return ctx
		}).
		// OnError: 错误日志、告警
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			traceID, _ := ctx.Value("trace_id").(string)

			if info == nil {
				fmt.Printf("[%s][OnError] ❌ 错误: %v\n", traceID, err)
			} else {
				fmt.Printf("[%s][OnError] ❌ %s/%s 失败: %v\n",
					traceID, info.Component, info.Name, err)
			}

			return ctx
		}).
		// OnEndWithStreamOutput: 流式输出处理
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			traceID, _ := ctx.Value("trace_id").(string)

			if info == nil {
				output.Close()
				return ctx
			}

			fmt.Printf("[%s][OnLLMStream] 开始流式输出\n", traceID)

			go func() {
				defer output.Close()
				chunkCount := 0

				for {
					_, err := output.Recv()
					if err != nil {
						break
					}
					chunkCount++
				}

				fmt.Printf("[%s][OnLLMStream] 流式输出完成 | 分片数: %d\n",
					traceID, chunkCount)
			}()

			return ctx
		}).
		Build()
}

// TestComprehensiveCallback 测试综合回调
func TestComprehensiveCallback(t *testing.T) {
	// 注册全局回调
	callbacks.AppendGlobalHandlers(NewComprehensiveHandler())

	fmt.Println("\n=== 测试综合 Callback ===")
	fmt.Println("场景：完整的可观测性方案，包含所有回调类型")
	fmt.Println("功能：")
	fmt.Println("  1. OnLLMStart - 记录 Prompt、拦截敏感输入")
	fmt.Println("  2. OnLLMEnd - 统计 Token、记录完整问答、计费")
	fmt.Println("  3. OnLLMStream - 实时流式日志、实时内容过滤")
	fmt.Println("  4. OnToolStart - 校验工具参数、权限校验、审计")
	fmt.Println("  5. OnToolEnd - 记录工具返回结果、耗时、审计留痕")
	fmt.Println("  6. OnRetrieverStart - 记录检索 Query、拦截非法检索")
	fmt.Println("  7. OnRetrieverEnd - 记录检索召回文档、RAG 链路分析")
	fmt.Println("  8. OnError - 错误日志、告警、异常兜底、故障上报")
}

// ============================================================================
// 工具函数
// ============================================================================

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateTraceID 生成追踪 ID
func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

// getTotalTokens 获取总 Token 数（已废弃，直接使用 TokenUsage.TotalTokens）
// func getTotalTokens(usage *model.TokenUsage) int {
// 	if usage == nil {
// 		return 0
// 	}
// 	return usage.TotalTokens
// }
