# key-bind

CLIProxyAPI 插件:为每个 proxy API Key 绑定**允许使用的供应商/账号**,未绑定的 Key 按平台原策略放行。带可视化配置页。

## 它解决什么

CLIProxyAPI 原生不支持「按 API Key 限制可用供应商」(顶层 `api-keys` 只是字符串列表,路由不按 key 过滤)。本插件在 `scheduler.pick` 阶段补上这层过滤:

- **Key 有绑定** → 只能用绑定的供应商/账号;允许集合内轮询负载均衡,无可用账号时返回 503(`auth_not_found`,**不回退**,保证隔离)。
- **Key 无绑定** → 插件返回 `Handled:false`,**直接跳过**,请求走平台原有调度策略(全部账号可用)。

Key 的鉴权仍完全由平台顶层 `api-keys` 负责,本插件只做转发收窄。

## 工作原理

1. 请求带 API Key 进入,先经平台 `config-api-key` 鉴权(不变)。
2. 到 `scheduler.pick`,插件从请求头取 Key,哈希后查绑定记录:
   - 查不到 / 已停用 → `Handled:false`(跳过,平台原策略)。
   - 查到 → 用绑定的 `allow` 列表过滤候选账号:`claude`/`openrouter` 等匹配 `candidate.Provider`(类型,覆盖该类型全部账号);`auth:<id>` 精确匹配 `candidate.ID`(具体账号)。
3. 过滤后非空 → 轮询选一个返回;为空 → 拒绝。

## 构建

需要 Go ≥ 1.26 与 Node.js(带 npm)。

```bash
make all        # 先构建前端(内联成单 index.html),再编译本机平台的插件二进制
# 产物:key-bind.so (Linux) / key-bind.dylib (macOS)
```

也可分步:`make web`(前端)→ `make plugin`(后端)。跨平台打包见 `make build-linux-amd64` / `build-linux-arm64`(arm64 需 C 交叉工具链)。

## 加载

把编译产物放进 CPA 的 `plugins.dir`(默认 `plugins/`),并在 `config.yaml` 启用:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    key-bind:
      enabled: true
      state_file: "key-bind-state.json"
```

> 首次放入新插件可热加载;但**替换一个已在运行的 `.so` 必须重启 CPA**(dlopen 运行时无法安全覆盖)。详见 CLIProxyAPI 文档。

## 使用配置页

1. 在 Management-Center「插件管理」里出现 `key-bind` 菜单,点开即配置页(作为同源 iframe,自动复用面板已登录的管理密钥,免登录;跨源时回退到登录页)。
2.「新建绑定」:
   - **名称**:备注,如「团队A」。
   - **API Key**:从平台 `api-keys` 下拉选。
   - **允许的供应商/账号**(多选):
     - *AI 供应商* 组:各 provider 类型(如 `claude`/`codex`/`gemini`),选中=允许该类型全部账号。
     - *认证文件* 组:具体 OAuth 凭证账号,选中(`auth:<id>`)= 精确到该账号。
3. 保存即可,绑定记录写入 `state_file`,热生效(`plugin.reconfigure`)。

## 约束

1. **单 scheduler 互斥**:CPA 同一时刻只启用一个 scheduler 插件。启用本插件后,不能再启用 `cpa-key-policy`、`codex-quota-scheduler` 等其它 scheduler 插件。
2. **Key 仍须在顶层 `api-keys`**:本插件不做鉴权;Key 必须先通过平台 `config-api-key`,请求才会到达本插件。
3. **`auth:<id>` 精确匹配依赖 ID 一致**:认证文件选项生成的 `auth:<id>` 需与 `scheduler` 候选的 `ID` 一致。若发现匹配不上,可改用「AI 供应商」类型粒度(覆盖全部账号)。

## 开发

- 改 Go 代码 → `make plugin`(已加载需重启 CPA)。
- 改前端 → `make web` 后重新 `make plugin`(embed 进二进制)。
- 改绑定记录(运行时)→ 配置页直接改,热加载,不重启。

## 目录结构

```
cpa-plugin-key-bind/
├── cmd/key-bind/            # C-ABI 入口(main.go cshared / main_stub.go)
├── internal/
│   ├── plugin/              # app/scheduler/management/types + web(embed)
│   │   └── web/dist/index.html   # 嵌入的配置页(构建前为 placeholder)
│   └── store/               # 绑定记录模型 + JSON 持久化
├── web/                     # React + Vite + TS 配置页(构建成单 index.html)
├── Makefile / go.mod / config.example.yaml
```
