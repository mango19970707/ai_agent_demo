# Eino Skill 中间件测试示例

## 概述

本示例演示了如何使用 Eino 的 Skill 中间件，让 Agent 能够发现并加载可复用的技能文档（SKILL.md）。

## 核心概念

### 1. Skill（技能）

Skill 是可复用的知识/指令包，用 markdown 格式（SKILL.md）描述"如何做某类事"。

**与 Tool 的区别：**
- **Tool**：动作/能力（读文件、调用 API、执行命令）
- **Skill**：知识/指令包（代码审查清单、测试编写指南、文档规范）

### 2. Skill Backend

从文件系统加载技能的后端，负责扫描和读取 SKILL.md 文件。

### 3. Skill Middleware

将技能注册到 Agent 的中间件，使 Agent 可以通过工具调用使用这些技能。

## 目录结构

```
skills/
├── skills_test.go          # 测试文件
├── README.md               # 本文档
└── test_skills/            # 测试时动态创建的技能目录
    ├── code_review/
    │   └── SKILL.md        # 代码审查技能
    ├── test_generator/
    │   └── SKILL.md        # 测试生成技能
    └── doc_writer/
        └── SKILL.md        # 文档编写技能
```

## 运行测试

```bash
cd ai_agent_demo/skills
go test -v
```

## 测试用例

### 1. TestSkillMiddlewareBasic

测试基本的 Skill 中间件功能，验证：
- 创建本地文件系统 backend
- 从文件系统创建 Skill Backend
- 创建 Skill 中间件
- 将中间件注册到 Agent

### 2. TestSkillDiscovery

测试技能发现功能，验证：
- Skill Backend 能够扫描技能目录
- 正确识别所有技能

### 3. TestSkillStructure

测试技能目录结构，验证：
- 每个技能都有 SKILL.md 文件
- 文件结构符合规范

### 4. TestSkillWithAgent

测试完整的 Agent + Skill 集成，验证：
- Agent 能够加载所有技能
- 技能可以通过工具调用使用

## 关键代码示例

### 创建 Skill Backend

```go
// 1. 创建本地文件系统 backend
backend, err := localbk.NewBackend(ctx, &localbk.Config{})

// 2. 从文件系统创建 Skill Backend
// BaseDir 指向包含所有技能的根目录
skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
    Backend: backend,
    BaseDir: skillsDir, // 例如: "./skills/eino-ext"
})
```

### 创建 Skill 中间件

```go
// 3. 创建 Skill 中间件
skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
    Backend: skillBackend,
})
```

### 注册到 Agent

```go
// 4. 创建 Agent 并注册 Skill 中间件
agent, err := deep.New(ctx, &deep.Config{
    ChatModel:       cm,
    Backend:         backend,
    StreamingShell:  backend,
    Handlers: []adk.ChatModelAgentMiddleware{
        skillMiddleware, // 注册 skill 中间件
        // ... 其他中间件
    },
})
```

## SKILL.md 文件格式

每个技能的 SKILL.md 文件应该包含：

```markdown
---
name: skill_name
description: 技能的简短描述
---

# 技能标题

技能的详细说明。

## 能力

列出技能提供的能力。

## 使用方法

详细的使用说明和示例。
```

## 示例技能

### 1. code_review（代码审查）

提供代码审查的最佳实践和检查清单：
- 代码质量检查
- 安全性检查
- 性能优化建议
- 测试覆盖率验证

### 2. test_generator（测试生成）

提供测试用例生成的指导和模板：
- 单元测试模板
- 测试命名规范
- AAA 测试结构
- 覆盖率目标

### 3. doc_writer（文档编写）

提供技术文档编写的最佳实践：
- README 文档结构
- API 文档规范
- 代码注释原则

## 使用场景

### 在 Agent 中使用 Skill

当 Agent 配置了 Skill 中间件后，可以通过工具调用使用技能：

```json
{
  "skill": "code_review"
}
```

Agent 会加载对应的 SKILL.md 文件，并根据其中的指导进行操作。

### 真实应用场景

1. **代码审查助手**
   - 加载 code_review 技能
   - 根据清单检查代码质量
   - 提供改进建议

2. **测试生成助手**
   - 加载 test_generator 技能
   - 根据模板生成测试用例
   - 确保测试覆盖率

3. **文档编写助手**
   - 加载 doc_writer 技能
   - 根据规范编写文档
   - 保持文档一致性

## 最佳实践

### 1. 技能组织

- **按领域分类**：将相关技能放在同一目录下
- **清晰命名**：使用描述性的技能名称
- **版本管理**：为技能添加版本信息

### 2. SKILL.md 编写

- **简洁明了**：避免冗长的描述
- **结构化**：使用标题和列表组织内容
- **示例丰富**：提供具体的使用示例
- **可操作性**：提供明确的操作步骤

### 3. 技能维护

- **定期更新**：根据最佳实践更新技能内容
- **测试验证**：确保技能指导的有效性
- **文档同步**：保持技能文档与实际使用一致

## 扩展思考

### 技能的复用性

Skill 的设计理念是"可复用的知识包"，可以：
- 在多个项目间共享
- 通过版本控制管理
- 根据团队规范定制

### 与其他中间件的配合

Skill 中间件可以与其他中间件配合使用：
- **approval**：需要人工确认时使用技能指导
- **retry**：失败重试时参考技能建议
- **safeTool**：安全检查时使用技能清单

### 技能的动态加载

在真实应用中，可以：
- 根据任务类型动态加载相关技能
- 支持技能的热更新
- 实现技能的权限控制

## 参考资料

- [Eino 官方文档](https://github.com/cloudwego/eino)
- [Skill 中间件文档](../../eino-examples/quickstart/chatwitheino/docs/ch09_skill.md)
- [Graph Tool 示例](../graph_tool/README.md)
- [Callback 示例](../callback/README.md)

## 注意事项

1. **技能目录路径**：必须使用绝对路径或正确的相对路径
2. **SKILL.md 格式**：必须包含 frontmatter（`---` 包围的元数据）
3. **文件权限**：确保 Agent 有读取技能文件的权限
4. **Mock 模型限制**：本示例使用 mock 模型，真实场景需要使用真实的 ChatModel
