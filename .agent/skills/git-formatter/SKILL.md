---
name: git-commit-formatter
description: 自动分析 Git 暂存区更改，并根据 Conventional Commits 规范（如 feat:, fix:）生成提交信息。
trigger:
  - type: message_contains
    pattern: "帮我提交代码"
  - type: intent
    pattern: "format git commit"
# command 定义了 Agent 如何运行此技能，可以使用环境变量和相对路径
command: python .agent/skills/git-formatter/script.py
---

## Git 提交信息格式化助手

本技能通过 Python 脚本调用 `git diff --cached` 获取更改内容，并利用 AI 建议符合规范的提交信息。

**支持的类型：**
- `feat`: 新功能
- `fix`: 修复问题
- `docs`: 文档变更
- `style`: 代码格式（不影响逻辑）
- `refactor`: 重构

**交互流程：**
1. 用户说：“帮我提交代码”。
2. Agent 执行 `script.py`。
3. 脚本输出格式化建议，Agent 将其呈现给用户确认。