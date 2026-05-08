package skills

import (
	"os"
	"path/filepath"
	"testing"
)

/*
Skill 中间件测试示例

本测试演示 Eino 的 Skill 概念和使用方法。

核心概念：
1. Skill - 可复用的知识/指令包，用 SKILL.md 描述
2. Skill Backend - 从文件系统加载技能的后端
3. Skill Middleware - 将技能注册到 Agent 的中间件

与 Tool 的区别：
- Tool：动作/能力（读文件、调用 API）
- Skill：知识/指令包（如何做某类事的说明书）

注意：本测试主要演示 Skill 的概念和目录结构，不依赖 eino-ext/adk。
如需完整的 Skill 中间件集成，请参考 eino-examples 项目。
*/

// TestSkillStructure 测试技能目录结构
func TestSkillStructure(t *testing.T) {
	// 创建测试用的技能目录
	skillsDir := setupTestSkills(t)
	defer os.RemoveAll(skillsDir)

	t.Logf("✓ 技能目录结构测试")
	t.Logf("  Skills 目录: %s", skillsDir)
	t.Logf("")

	// 验证每个技能都有 SKILL.md 文件
	skills := []struct {
		name        string
		description string
	}{
		{"code_review", "代码审查技能"},
		{"test_generator", "测试生成技能"},
		{"doc_writer", "文档编写技能"},
	}

	for _, skill := range skills {
		skillPath := filepath.Join(skillsDir, skill.name, "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			t.Errorf("技能 %s 缺少 SKILL.md 文件", skill.name)
		} else {
			t.Logf("  ✓ %s - %s", skill.name, skill.description)

			// 读取并验证文件内容
			content, err := os.ReadFile(skillPath)
			if err != nil {
				t.Errorf("读取 SKILL.md 失败: %v", err)
				continue
			}

			// 验证文件包含必要的元数据
			contentStr := string(content)
			if len(contentStr) == 0 {
				t.Errorf("SKILL.md 文件为空")
			}

			// 验证包含 frontmatter
			if contentStr[:3] != "---" {
				t.Errorf("SKILL.md 缺少 frontmatter")
			}
		}
	}
}

// TestSkillContent 测试技能内容的完整性
func TestSkillContent(t *testing.T) {
	skillsDir := setupTestSkills(t)
	defer os.RemoveAll(skillsDir)

	t.Logf("✓ 技能内容完整性测试")
	t.Logf("")

	// 测试 code_review 技能
	t.Run("code_review技能", func(t *testing.T) {
		skillPath := filepath.Join(skillsDir, "code_review", "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("读取技能文件失败: %v", err)
		}

		contentStr := string(content)

		// 验证包含关键章节
		requiredSections := []string{
			"# 代码审查技能",
			"## 能力",
			"## 使用方法",
			"代码质量",
			"安全性",
			"性能",
			"测试",
		}

		for _, section := range requiredSections {
			if !contains(contentStr, section) {
				t.Errorf("技能内容缺少必要章节: %s", section)
			}
		}

		t.Logf("  ✓ code_review 技能内容完整")
		t.Logf("    - 包含代码质量检查指南")
		t.Logf("    - 包含安全性检查清单")
		t.Logf("    - 包含性能优化建议")
		t.Logf("    - 包含测试覆盖率要求")
	})

	// 测试 test_generator 技能
	t.Run("test_generator技能", func(t *testing.T) {
		skillPath := filepath.Join(skillsDir, "test_generator", "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("读取技能文件失败: %v", err)
		}

		contentStr := string(content)

		// 验证包含关键内容
		requiredContent := []string{
			"单元测试",
			"集成测试",
			"AAA",
			"覆盖率",
		}

		for _, keyword := range requiredContent {
			if !contains(contentStr, keyword) {
				t.Errorf("技能内容缺少关键词: %s", keyword)
			}
		}

		t.Logf("  ✓ test_generator 技能内容完整")
		t.Logf("    - 包含单元测试模板")
		t.Logf("    - 包含测试命名规范")
		t.Logf("    - 包含 AAA 测试结构")
		t.Logf("    - 包含覆盖率目标")
	})

	// 测试 doc_writer 技能
	t.Run("doc_writer技能", func(t *testing.T) {
		skillPath := filepath.Join(skillsDir, "doc_writer", "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("读取技能文件失败: %v", err)
		}

		contentStr := string(content)

		// 验证包含关键内容
		requiredContent := []string{
			"README",
			"API 文档",
			"注释",
		}

		for _, keyword := range requiredContent {
			if !contains(contentStr, keyword) {
				t.Errorf("技能内容缺少关键词: %s", keyword)
			}
		}

		t.Logf("  ✓ doc_writer 技能内容完整")
		t.Logf("    - 包含 README 文档结构")
		t.Logf("    - 包含 API 文档规范")
		t.Logf("    - 包含代码注释原则")
	})
}

// TestSkillUsageExample 演示如何使用技能
func TestSkillUsageExample(t *testing.T) {
	skillsDir := setupTestSkills(t)
	defer os.RemoveAll(skillsDir)

	t.Logf("✓ Skill 使用示例")
	t.Logf("")
	t.Logf("在真实应用中，Skill 的使用流程如下：")
	t.Logf("")
	t.Logf("1. 准备技能目录")
	t.Logf("   export SKILLS_DIR=/path/to/skills")
	t.Logf("")
	t.Logf("2. 创建 Skill Backend（需要 eino-ext/adk）")
	t.Logf("   backend, _ := localbk.NewBackend(ctx, &localbk.Config{})")
	t.Logf("   skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{")
	t.Logf("       Backend: backend,")
	t.Logf("       BaseDir: skillsDir,")
	t.Logf("   })")
	t.Logf("")
	t.Logf("3. 创建 Skill 中间件")
	t.Logf("   skillMiddleware, _ := skill.NewMiddleware(ctx, &skill.Config{")
	t.Logf("       Backend: skillBackend,")
	t.Logf("   })")
	t.Logf("")
	t.Logf("4. 注册到 Agent")
	t.Logf("   agent, _ := deep.New(ctx, &deep.Config{")
	t.Logf("       Handlers: []adk.ChatModelAgentMiddleware{")
	t.Logf("           skillMiddleware,")
	t.Logf("       },")
	t.Logf("   })")
	t.Logf("")
	t.Logf("5. Agent 通过工具调用使用技能")
	t.Logf("   工具参数: {\"skill\": \"code_review\"}")
	t.Logf("")
	t.Logf("完整示例请参考:")
	t.Logf("  - eino-examples/quickstart/chatwitheino/cmd/ch09/main.go")
	t.Logf("  - eino-examples/adk/middlewares/skill/main.go")
}

// setupTestSkills 创建测试用的技能目录结构
func setupTestSkills(t *testing.T) string {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "eino-skills-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建三个示例技能
	skills := []struct {
		name        string
		description string
		content     string
	}{
		{
			name:        "code_review",
			description: "代码审查技能，帮助进行代码质量检查",
			content: `---
name: code_review
description: 代码审查技能，帮助进行代码质量检查和最佳实践建议
---

# 代码审查技能

本技能提供代码审查的最佳实践和检查清单。

## 能力

- 检查代码风格和格式
- 识别潜在的 bug 和安全问题
- 提供性能优化建议
- 验证测试覆盖率

## 使用方法

在需要代码审查时，可以参考以下检查清单：

### 1. 代码质量
- 函数是否过长（建议 < 50 行）
- 是否有重复代码
- 命名是否清晰易懂
- 是否遵循项目编码规范

### 2. 安全性
- 是否有 SQL 注入风险
- 是否有 XSS 漏洞
- 敏感信息是否硬编码
- 是否正确处理用户输入

### 3. 性能
- 是否有不必要的循环嵌套
- 数据库查询是否优化
- 是否有内存泄漏风险
- 是否使用了合适的数据结构

### 4. 测试
- 是否有单元测试
- 测试覆盖率是否达标（建议 > 80%）
- 是否有集成测试
- 边界条件是否测试

### 5. 错误处理
- 是否正确处理所有错误
- 错误信息是否清晰
- 是否有适当的日志记录
`,
		},
		{
			name:        "test_generator",
			description: "测试生成技能，帮助生成单元测试和集成测试",
			content: `---
name: test_generator
description: 测试生成技能，帮助生成高质量的单元测试和集成测试
---

# 测试生成技能

本技能提供测试用例生成的指导和模板。

## 能力

- 生成单元测试用例
- 生成集成测试用例
- 提供测试数据准备建议
- Mock 对象使用指导

## 使用方法

### 单元测试模板

对于每个函数，应该测试：

1. **正常情况**：输入有效数据，验证输出正确
2. **边界情况**：测试边界值（空值、最大值、最小值）
3. **异常情况**：测试错误处理逻辑

### 测试命名规范

使用 "Test + 函数名 + 场景" 的命名方式：
- TestCalculateSum_ValidInput
- TestCalculateSum_EmptyArray
- TestCalculateSum_NegativeNumbers

### 测试结构

每个测试应该遵循 AAA 模式：
- **Arrange**（准备）：设置测试数据和环境
- **Act**（执行）：调用被测试的函数
- **Assert**（断言）：验证结果是否符合预期

### 覆盖率目标

- 单元测试覆盖率：> 80%
- 关键业务逻辑：100%
- 错误处理路径：必须覆盖

### 测试数据准备

- 使用表驱动测试处理多个测试案例
- 准备典型数据、边界数据、异常数据
- 使用 Mock 隔离外部依赖

### 集成测试

- 测试组件间的交互
- 验证端到端的业务流程
- 使用测试数据库或容器
`,
		},
		{
			name:        "doc_writer",
			description: "文档编写技能，帮助编写清晰的技术文档",
			content: `---
name: doc_writer
description: 文档编写技能，帮助编写清晰、完整的技术文档
---

# 文档编写技能

本技能提供技术文档编写的最佳实践。

## 能力

- API 文档编写
- README 文档编写
- 架构设计文档编写
- 用户手册编写

## 使用方法

### README 文档结构

一个好的 README 应该包含：

1. **项目简介**：一句话说明项目是什么
2. **功能特性**：列出主要功能
3. **快速开始**：最简单的使用示例
4. **安装说明**：详细的安装步骤
5. **使用示例**：常见使用场景的代码示例
6. **API 文档**：主要接口说明
7. **贡献指南**：如何参与项目开发
8. **许可证**：开源协议说明

### API 文档规范

每个 API 应该包含：

- **功能描述**：简要说明 API 的作用
- **请求参数**：参数名、类型、是否必填、说明
- **返回值**：返回数据的结构和说明
- **错误码**：可能的错误情况和处理方式
- **示例代码**：完整的调用示例

### 代码注释原则

- 注释应该解释"为什么"，而不是"是什么"
- 复杂算法必须有注释说明
- 公共 API 必须有文档注释
- 避免无意义的注释（如：i++; // i 加 1）

### 架构文档

- 使用图表说明系统架构
- 说明关键设计决策和权衡
- 记录技术选型的理由
- 包含部署和运维指南
`,
		},
	}

	// 创建每个技能的目录和 SKILL.md 文件
	for _, skill := range skills {
		skillDir := filepath.Join(tmpDir, skill.name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("创建技能目录失败: %v", err)
		}

		skillFile := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillFile, []byte(skill.content), 0644); err != nil {
			t.Fatalf("创建 SKILL.md 文件失败: %v", err)
		}
	}

	return tmpDir
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
