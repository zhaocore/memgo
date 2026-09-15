# MemGo Skill for Claude

用 MemGo 自托管记忆层给 AI 应用添加持久化记忆:Go server 暴露 OSS HTTP API,CLI、MCP、
Go core 库一应俱全。

> **Skill 图谱:** 本 skill 是记忆层接入主入口。CLI 终端用法与具体接入安装细节分别见
> `memgo-cli`、`memgo-integrate`。

## 本 skill 能做什么

安装后,Claude 可以:

- **起 server**:`make up` + `make bootstrap` 或连生产实例,配 `MEMGO_API_KEY` / `MEMGO_BASE_URL`
- **用 HTTP 接入**:curl 或任意语言直连 `POST /memories`、`POST /search` 等 OSS 端点
- **用 Go 库接入**:在 Go 进程内嵌 `github.com/zhao-core/memgo/core/memory` 记忆引擎
- **接入编码 agent**:MCP server、Claude Code 插件、Codex hooks、DeepSeek 插件
- **生成可用代码**:基于真实端点与测试过的模式

## 安装

### CLI(Claude Code、OpenCode 或任意支持 skill 的工具)

把 `skills/memgo` 目录放到项目的 `.claude/skills/` 下即可。

### 前置

- 一个可访问的 MemGo server(自托管或 `https://memgo.wxget.com`)
- 环境变量:

  ```bash
  export MEMGO_API_KEY="..."
  export MEMGO_BASE_URL="https://memgo.wxget.com"   # 默认已指向该地址
  ```

## 快速开始

装好后直接问 Claude:

- "用 memgo 给我的聊天机器人加记忆"
- "帮我给 memgo server 写入一条记忆"
- "写个 Go 例子,搜索某用户的记忆"
- "怎么把 memgo 接到 Claude Code / Codex"

## 目录结构

```text
skills/memgo/
├── SKILL.md                    # skill 定义与入口
├── README.md                   # 本文件
├── LICENSE                     # Apache-2.0
└── references/                 # 按需加载的文档
    ├── quickstart.md           # 快速开始(起 server + curl + Go)
    ├── sdk-guide.md            # Go 客户端指南(core/memory + HTTP)
    ├── api-reference.md        # OSS HTTP 端点 / filters / 记忆对象
    ├── architecture.md         # core 分层 / 双库拓扑 / 流水线
    ├── features.md             # 同步提取 / 混合评分 / 实体 / 过滤
    ├── integration-patterns.md # MCP / CLI / 编码 agent hooks / HTTP 直连
    └── use-cases.md            # Go / HTTP 用例
```

## License

Apache-2.0
