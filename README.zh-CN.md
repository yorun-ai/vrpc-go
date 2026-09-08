# vrpc-go

独立的 Go [vRPC](https://github.com/yorun-ai/vrpc-ts) 客户端，兼容 [Vine Portal](https://github.com/yorun-ai/vine)。

[English](README.md) | **简体中文**

- Module：`go.yorun.ai/vrpc`；包名：`vrpc`。
- 要求 Go 1.27 或更高版本。JSON 客户端仅依赖 Go 标准库。
- 可选 CBOR/Binary 支持：`go.yorun.ai/vrpc/codec/cbor`，依赖 fxamacker/cbor。
- 不依赖 Vine、应用生命周期、DI、Actor 注册表或框架日志。
- Apache License 2.0。当前为初始实现，公共 API 尚未发布版本。

## 试用

克隆 [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go)，运行 `go test ./...`。
导入路径确定为 `go.yorun.ai/vrpc`；在完成该路径的 vanity 映射并发布版本前，可以在调用方 module 中使用本地替换：

```sh
go mod edit -require=go.yorun.ai/vrpc@v0.0.0
go mod edit -replace=go.yorun.ai/vrpc=/absolute/path/to/vrpc-go
go mod tidy
```

```go
client, err := vrpc.NewClient(vrpc.Options{
    Endpoint: "https://api.example.com/invoke",
    Identity: vrpc.Identity{Name: "demo.client", Version: "1.0.0"},
    Authorization: func(ctx context.Context) (string, error) {
        return vrpc.EncodeCredentials(map[string]string{"key": token})
    },
})
if err != nil {
    return err
}

var result struct {
    Name string `json:"name"`
}
metadata, err := client.Call(ctx, "demo.UserService", "Get",
    struct { ID int64 `json:"id"` }{ID: 42}, &result)
```

导入 `context` 和 `go.yorun.ai/vrpc`；示例的 `ctx`、`token` 由调用方提供。
服务名、方法名和字段名应使用实际契约中的名称。`Endpoint` 包含调用前缀；客户端追加
`/<service>/<method>`，不会自动添加 `/invoke`。

仓库提供完整的命令行示例：

```sh
# 认证调用通过环境变量 PORTAL_AUTHORIZATION 提供完整凭据。
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -service demo.UserService -method Get -params '{"id":42}'
```

## 配置与行为

`Identity` 默认使用 `vrpc.client`、版本 `0.0.0` 和自动生成的 UUID。
名称使用小写字母及点分隔，版本使用完整语义版本。每个 client 保持固定实例 UUID；每次调用生成 trace/span。
`vrpc.WithTrace(parent)` 保留父 trace ID，并创建新的调用 span。

`Options.Timeout` 默认 30 秒，`vrpc.WithTimeout(duration)` 可覆盖单次调用；context 中更早的 deadline 优先。
剩余时间通过 `vrpc-options` 传递。当前 Portal 默认 30 秒，拒绝超过 120 秒的值；客户端不硬编码网关上限。

取消客户端请求只停止等待，Portal 可能继续执行直到自己的 deadline。客户端不增加调用重试、不跟随重定向；
自定义 transport 的行为由调用方负责。不添加幂等 header，也不构造框架 Actor、Initiator。

`Authorization` 每次调用执行一次，接收当前调用 context，可用于刷新凭据；返回空字符串时不发送认证 header。
`EncodeCredentials` 按字段名排序，生成 `field value, field value`；拒绝逗号、控制字符、空值和大小写不敏感的重名字段。
它不默认使用 Bearer，也不在本地执行认证。

`Headers` 接受额外的单值 HTTP header，构造 client 时复制；vRPC、编码、报文分帧和幂等 header 为保留项。
`Transport` 接受 `http.RoundTripper`，可定制 TLS 和网络行为，生命周期由调用方管理。
自定义 codec、transport 和凭据回调支持并发时，client 可并发调用。

## JSON 与 Binary

默认使用 JSON。带 `json` 标签的具体类型保留 Go 整数精度；`any`/map 遵循标准 JSON 解码规则，
需要完整整数精度时使用具体类型字段。`encoding/json/v2` 将 nil slice/map 编码为空集合，与当前 Vine 行为一致，无需注册 schema。

Binary 参数或结果可启用可选 codec：

```go
import "go.yorun.ai/vrpc/codec/cbor"

client, err := vrpc.NewClient(vrpc.Options{
    Endpoint: endpoint,
    Codec: cbor.Codec{},
})
```

启用后发送 CBOR，并接受 CBOR 或 JSON 响应，兼容 Portal 的 JSON 错误响应。
`[]byte` 使用 CBOR byte string，整数 map key 保持整数，nil 集合编码为空集合。
使用一致的 `json` 和 `cbor` 字段标签；fxamacker 也支持回退使用 `json` 标签。
JSON-only client 拒绝意外的 CBOR 响应。压缩由 transport 处理，默认 Go transport 支持协商和解码 gzip；本库不声明 zstd 支持。

## 错误与元数据

成功需要 HTTP 2xx 和 `vrpc-status: OK`，有效的 `content-type`、`vrpc-status`、`vrpc-server`，以及包含 result 且没有 error 的响应信封。
未知的大写状态码仍会保留，便于向前兼容。

- `*vrpc.InvocationError` 保存远端失败状态或非 2xx 响应、可选的 `ErrorPayload`，并在 `Cause` 中保留解码失败。
- `*vrpc.ProtocolError` 表示成功响应的协议格式错误。
- transport、认证回调和 context 错误保留原始原因；用 `errors.Is` 判断取消、超时，用 `errors.As` 读取结构化错误。

`Response` 提供 HTTP 状态、协议状态、server identity、调用 trace、响应 header 和 `portal-trace-id`。
远端结果以 `Response.Status` 为准，不依赖消息文本或错误体里的辅助 `Code`。
`MaxResponseBytes` 默认 16 MiB，限制 transport 解压后的响应体；协议错误文本不包含原始响应体。

## 兼容性与范围

独立集成测试通过 HTTP 调用 Vine v0.15.3 的真实 Portal RpcGW 和认证代码；后端使用进程内 fixture，发现和 schema 使用内存 fixture。
覆盖 JSON/CBOR、匿名和认证接口、认证失败、业务错误、追踪传递及网关超时限制。
不模拟完整 Hub/Link 部署或后端 mTLS。

首版支持手写请求、结果类型。现有 skelc Go service client 仍依赖 Vine，不能直接传给此客户端；
生成器适配和 Vine 对本 module 的依赖迁移属于后续工作。本客户端不实现 Portal `/inspect`。

## 开发

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
cd integration
GOWORK=off go test -race ./...
```

[独立集成测试 module](integration/README.md) 将 Vine 隔离在库的依赖图之外。CI 执行两套测试。
仓库边界见 [AGENTS.md](AGENTS.md)。
