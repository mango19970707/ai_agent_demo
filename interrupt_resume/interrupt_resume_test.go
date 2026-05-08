package interrupt_resume

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

/*
===========================================
Eino 中断/恢复（Interrupt/Resume）测试示例
场景：写小说 Agent
===========================================

## 核心概念

1. **Interrupt（中断）**：在执行敏感操作前暂停，等待用户确认
2. **Resume（恢复）**：用户确认后继续执行，或拒绝后返回错误信息
3. **CheckPointStore**：保存中断状态，支持跨进程恢复
4. **ApprovalMiddleware**：拦截特定Tool调用，实现审批流程

## 写小说场景中的中断点

1. **保存章节**：写入文件前需要用户确认内容
2. **删除章节**：删除操作不可逆，需要用户确认
3. **发布小说**：发布前需要用户最终确认

## 使用方法

运行测试：
	go test -v

测试会模拟以下流程：
1. Agent调用敏感Tool（如保存章节）
2. Tool触发中断，返回审批信息
3. 用户查看信息并做出决定（批准/拒绝）
4. Resume恢复执行，根据用户决定执行或拒绝操作

## 执行流程

1. Agent 决定调用敏感Tool（如保存章节）
2. Tool 检测到首次调用，触发中断
3. 返回 InterruptInfo 给调用方
4. 用户查看操作详情，决定批准或拒绝
5. 调用 Resume 恢复执行
6. Tool 根据用户决定执行或拒绝操作

## 真实场景集成

在真实的Eino应用中，需要：
1. 使用 adk.Runner 配置 CheckPointStore
2. 实现 ApprovalMiddleware 拦截Tool调用
3. 使用 tool.StatefulInterrupt 触发中断
4. 使用 tool.GetInterruptState 和 tool.GetResumeContext 处理恢复
5. 参考 eino-examples/quickstart/chatwitheino/cmd/ch07/main.go
*/

// ==================== 数据结构 ====================

// NovelChapter 小说章节
type NovelChapter struct {
	ChapterID int    `json:"chapter_id"` // 章节ID
	Title     string `json:"title"`      // 章节标题
	Content   string `json:"content"`    // 章节内容
	WordCount int    `json:"word_count"` // 字数
}

// ApprovalInfo 审批信息（中断时携带的信息）
type ApprovalInfo struct {
	ToolName        string `json:"tool_name"`         // Tool名称
	Operation       string `json:"operation"`         // 操作类型（save/delete/publish）
	ArgumentsInJSON string `json:"arguments_in_json"` // 参数JSON
	Description     string `json:"description"`       // 操作描述
}

// ApprovalResult 审批结果（恢复时用户提供的结果）
type ApprovalResult struct {
	Approved         bool    `json:"approved"`          // 是否批准
	DisapproveReason *string `json:"disapprove_reason"` // 拒绝原因（可选）
}

// ==================== 简化的Tool实现（用于演示） ====================

// SaveChapterArgs 保存章节的参数
type SaveChapterArgs struct {
	ChapterID int    `json:"chapter_id"` // 章节ID
	Title     string `json:"title"`      // 章节标题
	Content   string `json:"content"`    // 章节内容
}

// saveChapterTool 保存章节Tool（简化版，用于演示）
func saveChapterTool(ctx context.Context, args string, isResume bool, approved bool) (string, *ApprovalInfo, error) {
	var chapterArgs SaveChapterArgs
	if err := json.Unmarshal([]byte(args), &chapterArgs); err != nil {
		return "", nil, fmt.Errorf("参数解析失败: %w", err)
	}

	if !isResume {
		// 首次调用：触发中断
		info := &ApprovalInfo{
			ToolName:        "save_chapter",
			Operation:       "save",
			ArgumentsInJSON: args,
			Description:     fmt.Sprintf("保存章节 %d: %s (共%d字)", chapterArgs.ChapterID, chapterArgs.Title, len(chapterArgs.Content)),
		}
		return "", info, nil
	}

	// Resume调用：根据审批结果执行
	if !approved {
		return "✗ 保存操作被用户拒绝", nil, nil
	}

	// 执行保存操作
	result := fmt.Sprintf("✓ 章节已保存: %s (ID=%d, 字数=%d)",
		chapterArgs.Title, chapterArgs.ChapterID, len(chapterArgs.Content))
	return result, nil, nil
}

// DeleteChapterArgs 删除章节的参数
type DeleteChapterArgs struct {
	ChapterID int `json:"chapter_id"` // 章节ID
}

// deleteChapterTool 删除章节Tool（简化版，用于演示）
func deleteChapterTool(ctx context.Context, args string, isResume bool, approved bool) (string, *ApprovalInfo, error) {
	var deleteArgs DeleteChapterArgs
	if err := json.Unmarshal([]byte(args), &deleteArgs); err != nil {
		return "", nil, fmt.Errorf("参数解析失败: %w", err)
	}

	if !isResume {
		// 首次调用：触发中断
		info := &ApprovalInfo{
			ToolName:        "delete_chapter",
			Operation:       "delete",
			ArgumentsInJSON: args,
			Description:     fmt.Sprintf("⚠️ 删除章节 %d（不可恢复）", deleteArgs.ChapterID),
		}
		return "", info, nil
	}

	// Resume调用：根据审批结果执行
	if !approved {
		return "✗ 删除操作被用户拒绝", nil, nil
	}

	// 执行删除操作
	result := fmt.Sprintf("✓ 章节 %d 已删除", deleteArgs.ChapterID)
	return result, nil, nil
}

// ViewOutlineArgs 查看大纲的参数
type ViewOutlineArgs struct {
	NovelID int `json:"novel_id"` // 小说ID
}

// viewOutlineTool 查看小说大纲Tool（无需审批）
func viewOutlineTool(ctx context.Context, args string) (string, error) {
	var outlineArgs ViewOutlineArgs
	if err := json.Unmarshal([]byte(args), &outlineArgs); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	// 模拟返回大纲
	outline := fmt.Sprintf(`小说大纲 (ID=%d):
第一章: 开端 - 主角登场
第二章: 发展 - 遇到挑战
第三章: 高潮 - 决战时刻
第四章: 结局 - 尘埃落定`, outlineArgs.NovelID)

	return outline, nil
}

// ==================== Mock Runner（用于测试） ====================

// mockRunner 模拟Runner，用于测试中断/恢复流程
type mockRunner struct {
	checkpointStore  map[string]*mockCheckpoint
	lastInterruptCtx *interruptContext
}

type mockCheckpoint struct {
	toolName string
	args     string
	info     *ApprovalInfo
}

type interruptContext struct {
	id       string
	toolName string
	info     *ApprovalInfo
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		checkpointStore: make(map[string]*mockCheckpoint),
	}
}

// Run 模拟执行Tool（第一次调用，可能触发中断）
func (r *mockRunner) Run(ctx context.Context, toolName string, args string) (string, *ApprovalInfo, error) {
	var result string
	var info *ApprovalInfo
	var err error

	switch toolName {
	case "save_chapter":
		result, info, err = saveChapterTool(ctx, args, false, false)
	case "delete_chapter":
		result, info, err = deleteChapterTool(ctx, args, false, false)
	case "view_outline":
		result, err = viewOutlineTool(ctx, args)
		return result, nil, err
	default:
		return "", nil, fmt.Errorf("Tool不存在: %s", toolName)
	}

	if err != nil {
		return "", nil, err
	}

	if info != nil {
		// 触发了中断，保存checkpoint
		checkpointID := fmt.Sprintf("cp_%s", toolName)
		r.checkpointStore[checkpointID] = &mockCheckpoint{
			toolName: toolName,
			args:     args,
			info:     info,
		}
		r.lastInterruptCtx = &interruptContext{
			id:       checkpointID,
			toolName: toolName,
			info:     info,
		}
		return "", info, nil
	}

	return result, nil, nil
}

// Resume 模拟恢复执行（用户审批后）
func (r *mockRunner) Resume(ctx context.Context, checkpointID string, approvalResult *ApprovalResult) (string, error) {
	checkpoint, ok := r.checkpointStore[checkpointID]
	if !ok {
		return "", fmt.Errorf("checkpoint不存在: %s", checkpointID)
	}

	var result string
	var err error

	switch checkpoint.toolName {
	case "save_chapter":
		result, _, err = saveChapterTool(ctx, checkpoint.args, true, approvalResult.Approved)
	case "delete_chapter":
		result, _, err = deleteChapterTool(ctx, checkpoint.args, true, approvalResult.Approved)
	default:
		return "", fmt.Errorf("Tool不存在: %s", checkpoint.toolName)
	}

	if err != nil {
		return "", err
	}

	return result, nil
}

// ==================== 测试用例 ====================

// TestNovelWritingWithInterrupt 测试写小说场景的中断/恢复
func TestNovelWritingWithInterrupt(t *testing.T) {
	ctx := context.Background()
	runner := newMockRunner()

	t.Run("保存章节-用户批准", func(t *testing.T) {
		// 1. 第一次调用：触发中断
		args := `{"chapter_id": 1, "title": "第一章：开端", "content": "很久很久以前..."}`
		result, interruptInfo, err := runner.Run(ctx, "save_chapter", args)

		if err != nil {
			t.Fatalf("Run失败: %v", err)
		}

		if interruptInfo == nil {
			t.Fatal("期望触发中断，但没有返回InterruptInfo")
		}

		t.Logf("✓ 触发中断")
		t.Logf("  Tool: %s", interruptInfo.ToolName)
		t.Logf("  操作: %s", interruptInfo.Operation)
		t.Logf("  描述: %s", interruptInfo.Description)

		// 2. 用户审批：批准
		approvalResult := &ApprovalResult{
			Approved: true,
		}

		// 3. Resume：恢复执行
		result, err = runner.Resume(ctx, runner.lastInterruptCtx.id, approvalResult)
		if err != nil {
			t.Fatalf("Resume失败: %v", err)
		}

		t.Logf("✓ Resume成功")
		t.Logf("  结果: %s", result)

		if !strings.Contains(result, "章节已保存") {
			t.Errorf("期望包含'章节已保存'，实际: %s", result)
		}
	})

	t.Run("删除章节-用户拒绝", func(t *testing.T) {
		// 1. 第一次调用：触发中断
		args := `{"chapter_id": 2}`
		result, interruptInfo, err := runner.Run(ctx, "delete_chapter", args)

		if err != nil {
			t.Fatalf("Run失败: %v", err)
		}

		if interruptInfo == nil {
			t.Fatal("期望触发中断，但没有返回InterruptInfo")
		}

		t.Logf("✓ 触发中断")
		t.Logf("  Tool: %s", interruptInfo.ToolName)
		t.Logf("  操作: %s", interruptInfo.Operation)
		t.Logf("  描述: %s", interruptInfo.Description)

		// 2. 用户审批：拒绝
		reason := "章节内容还需要修改"
		approvalResult := &ApprovalResult{
			Approved:         false,
			DisapproveReason: &reason,
		}

		// 3. Resume：恢复执行
		result, err = runner.Resume(ctx, runner.lastInterruptCtx.id, approvalResult)
		if err != nil {
			t.Fatalf("Resume失败: %v", err)
		}

		t.Logf("✓ Resume成功")
		t.Logf("  结果: %s", result)

		if !strings.Contains(result, "被用户拒绝") {
			t.Errorf("期望包含'被用户拒绝'，实际: %s", result)
		}
	})

	t.Run("查看大纲-无需审批", func(t *testing.T) {
		// 查看大纲不需要审批，直接执行
		args := `{"novel_id": 1}`
		result, interruptInfo, err := runner.Run(ctx, "view_outline", args)

		if err != nil {
			t.Fatalf("Run失败: %v", err)
		}

		if interruptInfo != nil {
			t.Fatal("view_outline不应该触发中断")
		}

		t.Logf("✓ 直接执行成功（无需审批）")
		t.Logf("  结果:\n%s", result)

		if !strings.Contains(result, "小说大纲") {
			t.Errorf("期望包含'小说大纲'，实际: %s", result)
		}
	})
}

// TestInterruptResumeFlow 测试完整的中断/恢复流程
func TestInterruptResumeFlow(t *testing.T) {
	ctx := context.Background()
	runner := newMockRunner()

	t.Log("=== 场景：Agent写小说，保存章节前需要用户确认 ===")

	// 步骤1：Agent决定保存章节
	t.Log("\n步骤1: Agent调用save_chapter")
	chapterArgs := SaveChapterArgs{
		ChapterID: 1,
		Title:     "第一章：命运的邂逅",
		Content:   "在一个月黑风高的夜晚，主角踏上了冒险的旅程...",
	}
	argsJSON, _ := json.Marshal(chapterArgs)

	result, interruptInfo, err := runner.Run(ctx, "save_chapter", string(argsJSON))
	if err != nil {
		t.Fatalf("Run失败: %v", err)
	}

	// 步骤2：触发中断，展示审批信息
	if interruptInfo == nil {
		t.Fatal("期望触发中断")
	}

	t.Log("\n步骤2: 触发中断，等待用户审批")
	t.Logf("  ⚠️  需要审批")
	t.Logf("  Tool名称: %s", interruptInfo.ToolName)
	t.Logf("  操作类型: %s", interruptInfo.Operation)
	t.Logf("  操作描述: %s", interruptInfo.Description)
	t.Logf("  参数详情: %s", interruptInfo.ArgumentsInJSON)

	// 步骤3：用户审批
	t.Log("\n步骤3: 用户审批")
	t.Log("  用户输入: y (批准)")

	approvalResult := &ApprovalResult{
		Approved: true,
	}

	// 步骤4：Resume恢复执行
	t.Log("\n步骤4: Resume恢复执行")
	result, err = runner.Resume(ctx, runner.lastInterruptCtx.id, approvalResult)
	if err != nil {
		t.Fatalf("Resume失败: %v", err)
	}

	t.Logf("  执行结果: %s", result)

	// 验证结果
	if !strings.Contains(result, "章节已保存") {
		t.Errorf("期望保存成功，实际: %s", result)
	}

	t.Log("\n✓ 完整流程测试通过")
}

// TestMultipleInterrupts 测试多次中断场景
func TestMultipleInterrupts(t *testing.T) {
	ctx := context.Background()
	runner := newMockRunner()

	t.Log("=== 场景：连续保存多个章节，每次都需要确认 ===")

	chapters := []SaveChapterArgs{
		{ChapterID: 1, Title: "第一章", Content: "内容1..."},
		{ChapterID: 2, Title: "第二章", Content: "内容2..."},
		{ChapterID: 3, Title: "第三章", Content: "内容3..."},
	}

	for i, chapter := range chapters {
		t.Logf("\n--- 保存章节 %d ---", i+1)

		argsJSON, _ := json.Marshal(chapter)

		// 触发中断
		_, interruptInfo, err := runner.Run(ctx, "save_chapter", string(argsJSON))
		if err != nil {
			t.Fatalf("Run失败: %v", err)
		}

		if interruptInfo == nil {
			t.Fatal("期望触发中断")
		}

		t.Logf("触发中断: %s", interruptInfo.Description)

		// 用户批准
		approvalResult := &ApprovalResult{Approved: true}
		result, err := runner.Resume(ctx, runner.lastInterruptCtx.id, approvalResult)
		if err != nil {
			t.Fatalf("Resume失败: %v", err)
		}

		t.Logf("执行结果: %s", result)
	}

	t.Log("\n✓ 多次中断测试通过")
}

// ==================== 最佳实践总结 ====================

/*
## 最佳实践

### 1. 何时使用 Interrupt

✓ 适合使用的场景：
- 不可逆操作（删除、发布）
- 敏感操作（修改配置、执行命令）
- 高成本操作（调用付费API、发送邮件）
- 需要人工决策的操作

✗ 不适合使用的场景：
- 只读操作（查询、查看）
- 低风险操作（记录日志）
- 高频操作（会影响用户体验）

### 2. 中断信息设计

好的中断信息应该包含：
- 操作名称：让用户知道要做什么
- 操作参数：让用户知道具体内容
- 风险提示：让用户了解潜在影响
- 操作描述：用人类语言解释

示例：
```go
&ApprovalInfo{
    ToolName:    "delete_chapter",
    Operation:   "delete",
    Description: "⚠️ 删除章节 5（不可恢复）",
    ArgumentsInJSON: `{"chapter_id": 5}`,
}
```

### 3. 审批策略

**白名单策略**（推荐）：
- 只对敏感操作要求审批
- 其他操作自动执行
- 用户体验好，安全性高

**黑名单策略**：
- 默认所有操作都需要审批
- 只有安全操作自动执行
- 安全性最高，但用户体验差

**动态策略**：
- 根据参数内容决定是否审批
- 例如：删除重要章节需要审批，删除草稿不需要
- 灵活性高，需要更复杂的逻辑

### 4. CheckPointStore 选择

**内存存储**（开发/测试）：
```go
CheckPointStore: adkstore.NewInMemoryStore()
```
- 优点：简单快速
- 缺点：进程重启后丢失

**持久化存储**（生产环境）：
```go
CheckPointStore: adkstore.NewRedisStore(...)
```
- 优点：支持跨进程恢复
- 缺点：需要额外的存储服务

### 5. 用户体验优化

- 提供清晰的操作描述
- 显示操作的具体参数
- 标注风险等级（⚠️ 高风险）
- 支持批量审批（一次审批多个操作）
- 提供撤销机制

### 6. 安全考虑

- 敏感参数脱敏（密码、Token等）
- 审批超时机制（避免长时间挂起）
- 审批日志记录（审计追踪）
- 权限校验（不同用户不同审批权限）

## 真实场景集成指南

参考 eino-examples/quickstart/chatwitheino/cmd/ch07/main.go：

1. 配置Runner使用CheckPointStore
2. 实现ApprovalMiddleware拦截Tool调用
3. 在Tool中使用tool.StatefulInterrupt触发中断
4. 使用tool.GetInterruptState和tool.GetResumeContext处理恢复
5. 处理InterruptInfo事件并调用runner.ResumeWithParams

详细文档：eino-examples/quickstart/chatwitheino/docs/ch07_interrupt_resume.md
*/
