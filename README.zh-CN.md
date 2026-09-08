# vrpc-go

独立的 Go [vRPC](https://github.com/yorun-ai/vrpc-ts) 客户端，兼容 [Vine Portal](https://github.com/yorun-ai/vine)。

[English](README.md) | **简体中文**

- Module：`go.yorun.ai/vrpc`；包名：`vrpc`。
- 要求 Go 1.27 或更高版本，内置 JSON 和 CBOR/Binary 支持（依赖 fxamacker/cbor）。
- 不依赖 Vine、应用生命周期、DI、Actor 注册表或框架日志。
- Apache License 2.0。当前为初始实现，公共 API 尚未发布版本。

## 试用

克隆 [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go)，运行 `go test ./...`。
导入路径 `go.yorun.ai/vrpc` 已映射到本仓库；首个版本发布前，可以在调用方 module 中使用本地替换：

```sh
go mod edit -require=go.yorun.ai/vrpc@v0.0.0
go mod edit -replace=go.yorun.ai/vrpc=/absolute/path/to/vrpc-go
go mod tidy
```

```go
client, err := vrpc.NewClient(vrpc.Option{
    Endpoint: "https://api.example.com/invoke",
    Identity: vrpc.Identity{
        Name: "demo.client",
        Version: "1.0.0",
        InstanceID: "123e4567-e89b-12d3-a456-426614174001",
    },
    Authorization: map[string]string{
        "key": token,
    },
})
if err != nil {
    return err
}

registry := vrpc.NewRegistry()
registry.Register(&vrpc.ServiceSpec{
    SkelName: "demo.UserService",
    Methods: []vrpc.MethodSpec{
        {
            SkelName: "Get",
        },
    },
})
methodInfo, ok := registry.GetMethodInfo("demo.UserService", "Get")
if !ok {
    return fmt.Errorf("method not registered")
}

type User struct {
    Name string `json:"name"`
}
result, metadata, err := client.Invoke[User](ctx, methodInfo,
    struct { ID int64 `json:"id"` }{ID: 42})
```

导入 `context`、`fmt` 和 `go.yorun.ai/vrpc`；示例的 `ctx`、`token` 由调用方提供。
服务名、方法名和字段名应使用实际契约中的名称。`Endpoint` 包含调用前缀；客户端追加
`/<service>/<method>`，不会自动添加 `/invoke`。

仓库提供完整的命令行示例：

```sh
# 认证调用通过环境变量 PORTAL_KEY 提供 key 凭据。
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -client-name demo.client -client-version 1.0.0 \
  -client-instance-id 123e4567-e89b-12d3-a456-426614174001 \
  -service demo.UserService -method Get -params '{"id":42}'
```

`client.Invoke[T]` 返回业务结果、响应元数据和错误；无返回值的方法使用 `client.Invoke[struct{}]`。

`client.InvokeRaw(ctx, service, method, params, options...)` 无需注册方法，使用 JSON
调用并返回 `(any, *ResponseMetadata, error)`。JSON 对象解码为 `map[string]any`，
数组为 `[]any`，数字为 `float64`。

失败时 `Invoke[T]` 返回 `T` 的零值，`InvokeRaw` 返回 nil；收到 HTTP 响应时仍保留元数据。

## 配置与行为

`Identity.Name`、`Identity.Version` 和 `Identity.InstanceID` 必须显式提供。
任一字段缺失或格式无效，创建客户端时直接返回错误。
名称使用小写字母及点分隔，版本使用完整语义版本。每个 client 保持固定实例 UUID；每次调用生成 trace/span。
`vrpc.WithTrace(parent)` 保留父 trace ID，并创建新的调用 span。

`Option.Timeout` 默认 30 秒，`vrpc.WithTimeout(duration)` 可覆盖单次调用；context 中更早的 deadline 优先。
剩余时间通过 `vrpc-options` 传递。当前 Portal 默认 30 秒，拒绝超过 120 秒的值；客户端不硬编码网关上限。

取消客户端请求只停止等待，Portal 可能继续执行直到自己的 deadline。客户端不增加调用重试、不跟随重定向；
自定义 transport 的行为由调用方负责。不添加幂等 header，也不构造框架 Actor、Initiator。

固定凭据通过 `Option.Authorization` 传入，创建客户端时编码一次，之后修改 map 不影响客户端。
非 nil map 覆盖 `Headers["Authorization"]`；空 map 删除该头，nil 则保留自定义头。
`EncodeCredentials` 按字段名排序，生成 `field value, field value`；拒绝逗号、控制字符、空值和大小写不敏感的重名字段。
它不默认使用 Bearer，也不在本地执行认证。

`Headers` 接受额外的 HTTP header 并保留多个值，构造 client 时复制；客户端生成的 header 会覆盖同名自定义值。
`Transport` 接受 `http.RoundTripper`，可定制 TLS 和网络行为，生命周期由调用方管理。
自定义 transport 支持并发时，client 可并发调用。

## 客户端 registry

`Registry` 只保存客户端服务和方法契约，不包含服务端注册、执行器、DI 或服务发现。
注册会复制方法描述、拒绝重复注册，并原子地发布服务；
支持并发调用和注册。

为生成客户端提供的注册接口如下：

```go
// 位于生成的客户端包中，导入 go.yorun.ai/vrpc。
func init() {
    vrpc.Register(&vrpc.ServiceSpec{
        SkelName: "demo.Files",
        Methods: []vrpc.MethodSpec{
            {SkelName: "upload", ArgumentsContainsBinaryType: true},
            {SkelName: "download", ResultContainsBinaryType: true},
        },
    })
}
```

通过 `GetMethodInfo` 获取一次方法描述，检查返回的布尔值，再直接调用
`client.Invoke[T](ctx, methodInfo, params, options...)`。生成客户端可以在包初始化后保存它。
描述包含服务名、方法名、请求路径和 binary 标志；调用时不再查询 registry，空描述会在发送前被拒绝。

请求与响应独立选择编码：参数含 binary 时使用 CBOR，返回值含 binary 时通过 `Accept`
接受 CBOR 和 JSON；其他情况使用 JSON。两种编码均已内置，无需导入或配置 codec。

需要隔离时使用 `NewRegistry`，默认 registry 则使用包级 `Register`/`GetMethodInfo`。
客户端可以直接使用任一 registry 返回的描述，不需要 registry 配置。

`ServiceSpec.Name` 和 `MethodSpec.Name` 保存 Go 名称，通过
`MethodInfo.ServiceName()` 和 `Name()` 读取；路由仍使用 `SkelName`。

`MethodSpec.ArgumentsSensitive` 和 `ResultSensitive` 标记参数及结果是否敏感，
日志和诊断功能可通过 `MethodInfo` 的同名方法读取；这些标记不改变编码，也不会自动脱敏。

`Register` 无返回值，对无效或重复注册直接 panic。

本库已实现 registry 和调用支持；skelc 现有 Go 生成器仍面向 Vine，生成上述注册代码需另行适配。

## JSON 与 Binary

默认使用 JSON。带 `json` 标签的具体类型保留 Go 整数精度；`any`/map 遵循标准 JSON 解码规则，
需要完整整数精度时使用具体类型字段。`encoding/json/v2` 将 nil slice/map 编码为空集合，与当前 Vine 行为一致。手写调用也需要注册方法契约并取得描述。

Binary 参数或结果按前述示例注册方法的 binary 标志即可。`[]byte` 使用 CBOR byte string，
整数 map key 保持整数，nil 集合编码为空集合。使用一致的 `json` 和 `cbor` 字段名；
fxamacker 也会回退到 `json` tag。返回值含 binary 的方法仍接受 JSON 响应，包括 Portal 错误。
仅声明接受 JSON 的调用会拒绝 CBOR 响应。压缩由 transport 处理，默认 Go transport 支持
协商和解码 gzip；本库不声明 zstd 支持。

## 错误与元数据

成功需要 HTTP 2xx 和 `vrpc-status: OK`，有效的 `content-type`、`vrpc-status`、`vrpc-server`，以及包含 result 且没有 error 的响应信封。
未知的大写状态码仍会保留，便于向前兼容。

- `*vrpc.InvocationError` 保存远端失败状态或非 2xx 响应、可选的 `ErrorPayload`，并在 `Cause` 中保留解码失败。
- `*vrpc.ProtocolError` 表示成功响应的协议格式错误。
- transport 和 context 错误保留原始原因；用 `errors.Is` 判断取消、超时，用 `errors.As` 读取结构化错误。

`ResponseMetadata` 提供 HTTP 状态、协议状态、server identity 和响应 header。
需要时可从 `ResponseMetadata.Header` 读取 `portal-trace-id`。
远端结果以 `ResponseMetadata.Status` 为准，不依赖消息文本或错误体里的辅助 `Code`。
共享 transport 将解压后的响应体限制为 128 MiB；协议错误文本不包含原始响应体。

## 共享传输层

`transport/http` 提供协议常量、header 校验、超时处理、JSON/CBOR 原始信封、
响应体限制和 HTTP 往返生命周期。JSON 和 CBOR 信封由同一个包提供。
适配层提供已编码的参数和结果、传输配置，以及应用相关的元数据和错误处理。
共享信封原样保留已编码的载荷，包括集合编码配置。

## 兼容性与范围

独立集成测试通过 HTTP 调用 Vine v0.15.3 的真实 Portal RpcGW 和认证代码；后端使用进程内 fixture，发现和 schema 使用内存 fixture。
覆盖 JSON/CBOR、匿名和认证接口、认证失败、业务错误、追踪传递及网关超时限制。
不模拟完整 Hub/Link 部署或后端 mTLS。

首版支持手写请求、结果类型。现有 skelc Go service client 仍依赖 Vine，不能直接传给此客户端；
生成器适配仍属于后续工作；Vine 通过框架适配层使用共享 transport，不依赖独立客户端 API。本客户端不实现 Portal `/inspect`。

## 包结构

根目录的 `api.go` 通过类型别名和普通函数转发提供客户端 API。客户端、registry、编码、
凭据和协议元数据的实现位于 `internal`，调用方仍导入 `go.yorun.ai/vrpc`。
`transport/http` 保持公开，供 Vine 跨模块复用。

## 开发

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
cd test/integration
GOWORK=off go test -race ./...
```

[独立集成测试 module](test/integration/README.md) 将 Vine 隔离在库的依赖图之外。CI 执行两套测试。
仓库边界见 [AGENTS.md](AGENTS.md)。
