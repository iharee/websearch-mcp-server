# Searcher 包结构重构

## 目标

纯结构整理，删除死代码，扁平化子包，不改任何行为。所有引擎实现保留硬编码 stub。

## 约束

- **对上层透明**：`main.go` 无需任何改动
- **引擎可切换**：优先级为 参数 `engine` > 环境变量 `SEARCH_ENGINE` > 默认 `duckduckgo`
- **结果格式统一**：所有引擎返回 `[]model.SearchResult`

## 最终文件结构

```
internal/searcher/
  provider.go          - Provider 接口 + Parser 接口
  tool.go              - MCP tool 定义 + handler + resolveProvider
  mock.go              - MockProvider（实现 Provider 接口）
  duckduckgo/
    duckduckgo.go       - Provider + Parser 合一，硬编码 stub
  tavily/
    tavily.go           - Provider + Parser 合一，硬编码 stub
```

## 各文件详情

### `provider.go` — 接口定义

合并现有 `provider.go` 和 `parser.go`。定义两个接口：

- `Provider`：`Search(ctx, query) ([]model.SearchResult, error)` — 引擎统一入口
- `Parser`：`Parse(data []byte) ([]model.SearchResult, error)` — Provider 内部使用，供后续 HTTP 响应解析

### `tool.go` — 注册入口

不变。唯一修改：import 路径适配新文件名，构造函数改名为 `duckduckgo.NewProvider()` / `tavily.NewProvider()`。

`resolveProvider()` 优先级逻辑不变：参数 > 环境变量 > 默认。

签名不变：`ToolDefinition() mcp.Tool` / `Handler() mcp.ToolHandler`。

### `mock.go` — 测试替身

`MockProvider` 不变，实现 `Provider` 接口。

### `duckduckgo/duckduckgo.go` — DuckDuckGo 引擎

合并 `duckduckgo_provider.go` + `duckduckgo_parser.go`：
- `Provider` struct 持有 `*Parser`
- `NewProvider() *Provider`
- `Parser` struct 的 `Parse` 返回硬编码 stub

### `tavily/tavily.go` — Tavily 引擎

同上模式，包名 `tavily`。

## 删除清单

| 文件 | 原因 |
|------|------|
| `internal/searcher/parser.go` | 接口合并到 `provider.go` |
| `internal/searcher/duckduckgo/duckduckgo_parser.go` | 合并到 `duckduckgo.go` |
| `internal/searcher/duckduckgo/duckduckgo_provider.go` | 合并到 `duckduckgo.go` |
| `internal/searcher/tavily/tavily_parser.go` | 合并到 `tavily.go` |
| `internal/searcher/tavily/tavily_provider.go` | 合并到 `tavily.go` |

## 不变项

- `internal/model/model.go` — `SearchResult` 不动
- `internal/config/config.go` — `SearchEngine()` / `TavilyAPIKey()` 不动
- `internal/mcp/` — 整个包不动
- `main.go` — 零改动
