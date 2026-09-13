# MemGo 安全审计报告（2026-09-11）

## 结论

本次为当前工作树的静态安全审计，不包含对已部署实例的渗透测试。未发现已验证的 Critical 代码漏洞；发现 2 项高风险部署默认值、2 项中风险的可用性或防误配问题，以及 1 项低风险的 CORS 一致性问题。

最高优先级是将生产环境置于 HTTPS 反向代理之后，并取消 Postgres 对宿主机默认公开映射。当前 `deploy/docker-compose.yaml` 虽要求数据库密码和 JWT secret，但默认端口暴露与 HTTP Dashboard 配置不适合直接公网部署。

## 范围与方法

审计对象：Go server、Dashboard 身份会话边界、Docker Compose/Dockerfile、CI、Go/Node 生产依赖锁文件、受跟踪文件中的常见明文密钥模式。

- 使用 CodeGraph 审阅路由、鉴权解析、CORS、限流、JWT 和 Dashboard refresh-token 路径。
- 检查 `.env` 忽略规则与受跟踪文件；未读取本机未跟踪 `.env`，避免暴露实际凭据。
- 运行 `go test ./server/... ./core/config/...`：通过；`server/api`、`server/auth`、`server/middleware` 当前没有测试文件。
- 运行 `pnpm --dir dashboard audit --prod --json` 与 `pnpm --dir cli/node audit --prod --json`：未返回生产依赖 advisory。
- 本机无 `govulncheck`、`gitleaks`、`trivy`、`semgrep` 或 `osv-scanner`；未安装新工具，因此 Go CVE、镜像 CVE、完整 secret 扫描和 SAST 均未验证。

## 发现

### 高 H-01：默认编排公开 Postgres 端口

- 证据：`deploy/docker-compose.yaml:20-21` 默认发布 `${POSTGRES_HOST_PORT:-8432}:5432`，与 API、Dashboard 一同暴露到宿主机。
- 影响：生产主机若未另行配置网络策略，数据库可从外部网络访问；认证密码泄露、弱口令、Postgres 漏洞或错误配置会直接扩大为数据泄露或篡改。
- 建议：仅让 `memgo-server` 接入 compose 内部网络，删除 `postgres.ports`；开发调试另用显式 profile 或仅绑定 `127.0.0.1:8432:5432`。生产环境设置强随机 `POSTGRES_PASSWORD`，并在网络层拒绝 5432 入站。
- 验证：部署后从非容器网络确认 5432 不可达，服务仍可连接 `postgres:5432`。

### 高 H-02：生产部署无 TLS 边界，默认可形成明文会话

- 证据：`cmd/server/main.go:91-93` 使用 `http.ListenAndServe`；`deploy/docker-compose.yaml:35-36,62-63` 默认公开 API/Dashboard HTTP 端口；`DASHBOARD_URL` 默认 `http://localhost:3000`。Dashboard refresh cookie 仅当 `DASHBOARD_URL` 是 HTTPS 或 `NODE_ENV=production` 时设置 `Secure`，见 `dashboard/src/app/api/auth/refresh/route.ts:7-27`。
- 影响：若按 compose 默认值直接暴露到公网，密码、Bearer token、API key 和 refresh token 可被链路窃听或篡改；HTTP 下 cookie 不具备 `Secure` 标志。
- 建议：生产拓扑必须使用受控 TLS 反向代理或负载均衡器，仅发布 HTTPS Dashboard；设置 `DASHBOARD_URL=https://<域名>`、不直接公开 API，使用防火墙限制内部 API 端口。部署说明应将 compose 标为开发基线，明确禁止直接公网发布。
- 验证：浏览器响应的 `memgo_refresh_token` 具备 `HttpOnly; Secure; SameSite=Lax`，HTTP 请求被重定向或拒绝，API 不可从公网直连。

### 中 M-01：`AUTH_DISABLED` 缺少生产环境硬性拒绝

- 证据：`cmd/server/main.go:35-46` 只记录日志警告后接受 `AUTH_DISABLED=true`；`server/api/authn.go:91-93` 在无凭据时返回 disabled auth context，路由上的管理和记忆操作随之可访问。
- 影响：部署变量误配会完全移除认证，管理员端点亦受影响。
- 建议：生产环境拒绝启动 `AUTH_DISABLED=true`，或要求第二个明确的仅开发确认变量；健康检查、部署模板和运行手册均应检测此状态。
- 验证：生产模式传入 `AUTH_DISABLED=true` 时进程以非零码退出；开发模式仍需显式开启。

### 中 M-02：登录/刷新限流器的状态可无界增长且不感知反向代理

- 证据：`server/middleware/ratelimit.go:12-43` 以 `map[string]*rateBucket` 保存每个 `RemoteAddr`，无过期清理或最大容量；`RemoteIP` 仅读取连接对端地址（`46-53`）。
- 影响：大量源地址可增长进程内存；置于共享反向代理后，所有用户可能被视为同一代理 IP，造成集体限流拒绝服务。多副本实例间也不共享额度。
- 建议：采用受信任代理配置后才解析转发地址；为 bucket 设置过期回收和上限。需要横向扩展时改用共享限流存储。
- 验证：压测大量独立 IP 后内存稳定；配置代理后按真实客户端 IP 限流；多实例额度符合预期。

### 低 L-01：不受信任 Origin 的 OPTIONS 响应仍提供方法和任意请求头列表

- 证据：`server/middleware/cors.go:13-17` 仅允许指定 Origin 返回 `Access-Control-Allow-Origin`；但 `18-23` 对任意带 Origin 的 OPTIONS 都返回允许方法与 `Access-Control-Allow-Headers: *`。
- 影响：浏览器因缺少 `Access-Control-Allow-Origin` 仍会阻止跨域读取，未确认形成数据泄露；但行为不一致，增加后续修改时错误放宽 CORS 的风险。
- 建议：仅在 Origin 等于 `DASHBOARD_URL` 时处理 OPTIONS 并发送 CORS 头；其他 Origin 直接交由路由处理或返回拒绝。
- 验证：允许 Origin 的预检成功；任意其他 Origin 不返回任何 `Access-Control-Allow-*` 头。

## 已确认的正向控制

- `JWT_SECRET` 在认证开启时为必填项，`ADMIN_API_KEY` 比较使用恒时比较：`cmd/server/main.go:38-42`、`server/auth/apikey.go:31-37`。
- API key 使用 32 字节随机值并在校验阶段使用 bcrypt：`server/auth/apikey.go:14-27`、`server/api/authn.go:76-89`。
- 密码使用 bcrypt，refresh token 带 JTI 并由数据库条件消费，减少重放窗口：`server/auth/password.go:5-19`、`server/api/authhandlers.go:132-160`。
- refresh token 由 Dashboard 存入 `HttpOnly`、`SameSite=Lax` cookie；访问 token 只回传给客户端内存：`dashboard/src/app/api/auth/refresh/route.ts:19-27`、`dashboard/src/lib/auth.tsx:53-61`。
- 默认 CORS 实际响应只回显精确的 `DASHBOARD_URL`，不使用通配 Origin：`server/middleware/cors.go:10-17`。
- `.env`、`dashboard/.env`、`cli/python/.env` 被 Git 忽略；受跟踪文件的常见高熵赋值扫描只命中测试、示例、接口字段和文档路径，未确认提交真实凭据。

## 未覆盖范围与后续

- 未对运行中的容器、真实域名、网络策略、TLS 证书、数据库权限或日志平台执行验证。
- 未运行 Go/镜像/完整 secret CVE 扫描；CI 当前没有依赖漏洞、镜像或 secret 扫描作业。
- 未进行 DAST、认证绕过、CSRF、XSS、SSRF、SQL 注入、并发或资源耗尽压测。
- 当前工作树包含用户未提交的 Dashboard、Python CLI、Node CLI 和品牌资源改动。本报告描述审计时的工作树，不代表任何提交或线上版本。

建议顺序：先处理 H-01/H-02，再处理 M-01/M-02；之后把 `govulncheck`、依赖审计、secret 扫描和容器扫描加入 CI，并在隔离环境执行 HTTP/浏览器动态验证。
