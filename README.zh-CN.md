# vrpc-go

[![License](https://img.shields.io/github/license/yorun-ai/vrpc-go)](LICENSE)
[![Version](https://img.shields.io/github/v/release/yorun-ai/vrpc-go?label=version&cacheSeconds=300)](https://github.com/yorun-ai/vrpc-go/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/yorun-ai/vrpc-go)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/go.yorun.ai/vrpc.svg)](https://pkg.go.dev/go.yorun.ai/vrpc)
[![CI](https://github.com/yorun-ai/vrpc-go/actions/workflows/ci.yml/badge.svg)](https://github.com/yorun-ai/vrpc-go/actions/workflows/ci.yml)

独立的 Go [vRPC](https://github.com/yorun-ai/vrpc-ts) 客户端，兼容 [Vine Portal](https://github.com/yorun-ai/vine)。

[English](README.md) | **简体中文**

- 模块路径 `go.yorun.ai/vrpc`，包名 `vrpc`。
- 要求 Go 1.27 或更高版本。内置 JSON 与 CBOR/Binary 编码，其中 CBOR 使用 fxamacker/cbor。
- 客户端不依赖 Vine、应用生命周期、依赖注入、Actor 注册表或框架日志。
- Apache License 2.0。公共 API 在 v1.0.0 之前仍处于稳定化阶段。

## 开始使用

模块路径 `go.yorun.ai/vrpc` 映射到本仓库。安装命令如下：

```sh
go get go.yorun.ai/vrpc@v0.18.0
```

可从 [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go) 克隆本仓库，并运行 `go test ./...` 验证。
如需使用尚未发布的源码，可在调用方模块中添加本地替换：

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
        InstanceID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
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
result, metadata, err := client.InvokeAs[User](ctx, methodInfo,
    struct { ID int64 `json:"id"` }{ID: 42})
```

`client.InvokeAs[T]` 返回解码后的结果、响应元数据和错误。无返回值的方法使用
`client.InvokeAs[struct{}]`。

`client.InvokeRaw(ctx, service, method, params, options...)` 以 JSON 调用未注册的方法，
返回 `(any, *ResponseMetadata, error)`。JSON 对象解码为 `map[string]any`，数组解码为
`[]any`，数字解码为 `float64`。

调用失败时，`InvokeAs[T]` 返回 `T` 的零值，`InvokeRaw` 返回 nil；若已收到 HTTP 响应，
响应元数据仍然可用。

示例假定已导入 `context`、`fmt`、`uuid` 和 `go.yorun.ai/vrpc`，且 `ctx` 与 `token`
由调用方定义。服务名、方法名和字段名必须与服务端契约一致。`Endpoint` 是调用地址前缀，
路由包含 `/invoke` 时也应包含在内；客户端会追加 `/<service>/<method>`。

仓库还提供一个命令行示例：

```sh
# 认证调用需要设置环境变量 PORTAL_KEY。
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -client-name demo.client -client-version 1.0.0 \
  -client-instance-id 123e4567-e89b-12d3-a456-426614174001 \
  -service demo.UserService -method Get -params '{"id":42}'
```

## 配置与行为

`Identity.Name` 要求使用以点分隔的小写字母，`Identity.Version` 要求完整的语义版本。每个客户端
应使用固定的 `Identity.InstanceID`。

每次调用都会创建 trace 和 span。`vrpc.WithTrace(parent)` 保留父 trace ID，并创建子 span。

`Option.Timeout` 默认为 30 秒。`vrpc.WithTimeout(duration)` 覆盖单次调用的超时，context
中更早的 deadline 优先。剩余时长通过 `vrpc-options` 传递。Portal 当前默认 30 秒，并拒绝
超过 120 秒的值；本客户端不硬编码该网关上限。

取消调用后，客户端会停止等待，但 Portal 仍可能继续执行至其自身的 deadline。客户端不重试
调用、不跟随重定向、不添加幂等 header，也不构造框架的 Actor 与 Initiator 值。自定义
transport 的行为由调用方负责。

固定凭据通过 `Option.Authorization` 设置，并在创建客户端时编码一次；此后修改 map 不会
影响该客户端。非 nil 的 map 覆盖 `Headers["Authorization"]`；空 map 删除该 header，nil
则保留自定义 header。`EncodeCredentials` 按字段名排序，生成
`field value, field value`，并拒绝逗号、控制字符、空值以及大小写不敏感的重复字段名。
该函数不假定采用 Bearer 认证，也不在本地执行认证。

`Headers` 接受额外的 HTTP header，并保留多个值；该字段在创建客户端时被复制。客户端生成的
同名 header 会覆盖自定义值。`Transport` 接受 `http.RoundTripper`，可用于自定义 TLS 或
网络行为；其生命周期由调用方管理。当自定义 transport 可安全用于并发时，客户端也支持并发
调用。

## 客户端注册表

`Registry` 保存客户端的服务与方法契约。它不提供服务端注册、执行器、依赖注入或服务发现。
注册时会复制方法描述符、拒绝重复项，并以原子方式发布服务；注册与查询均支持并发。

以下注册接口供生成的客户端包和手写调用方使用：

```go
// 位于导入 go.yorun.ai/vrpc 的生成客户端包中。
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

通过 `GetMethodInfo` 获取一次 `MethodInfo`，检查返回的布尔值，然后将描述符直接传给
`client.InvokeAs[T](ctx, methodInfo, params, options...)`。生成的客户端可以在包初始化后
保留该描述符。描述符包含服务名、方法名、调用路径和 binary 标志。调用时不会查询注册表，
空描述符会在发送请求前被拒绝。

若方法缺失应视为初始化错误，可改用
`methodInfo := vrpc.MustGetMethodInfo("demo.Files", "upload")`；`Registry` 实例也提供
`registry.MustGetMethodInfo`。契约缺失时，两者都会 panic，并在 panic 信息中包含服务名和
方法名。

请求编码与响应编码独立选择。参数含 binary 类型时使用 CBOR；结果含 binary 类型时，在
`Accept` 中声明 CBOR 和 JSON；其他参数与结果使用 JSON。两种编解码器均已内置，无需额外
导入或配置。

`ServiceSpec.Name` 与 `MethodSpec.Name` 保存 Go 名称，可通过 `MethodInfo.ServiceName()`
和 `Name()` 读取。路由仍使用 `SkelName`。

`MethodSpec.ArgumentsSensitive` 和 `ResultSensitive` 标记敏感载荷，对应的 `MethodInfo`
访问器向日志和诊断功能提供这些标志。这些标志不改变编码方式，也不会自动脱敏。

需要相互隔离的契约集合时，可使用 `NewRegistry`；默认注册表使用包级 `Register` 与
`GetMethodInfo`。客户端可以使用任一注册表返回的描述符，无需额外配置。`Register` 没有
返回值，对无效或重复注册会 panic。

skelc 的 Go API 客户端生成器面向本独立运行时；生成的后端包使用 Vine。

## JSON 与 Binary

默认编码为 JSON。带 `json` 标签的结构体保留 Go 整数精度；`any` 或 map 结果遵循标准
JSON 解码规则，需要整数精度时应使用具体类型的字段。在 `encoding/json/v2` 下，nil slice
与 map 编码为空集合，与当前 Vine 行为一致。

参数或结果含 binary 时，按上述示例注册方法的 binary 标志。`[]byte` 使用 CBOR byte
string，整数 map key 保持整数，nil 集合编码为空集合。`json` 与 `cbor` 字段名应保持一致；
没有 `cbor` 标签时，fxamacker/cbor 会回退到 `json` 标签。结果含 binary 的方法仍接受
JSON 响应，包括 Portal 错误；在 `Accept` 中只声明 JSON 的调用会拒绝 CBOR 响应。压缩由
传输层处理，默认 Go transport 会协商并解码 gzip。本包不声明支持 zstd。

## 错误与元数据

调用成功的条件为：HTTP 状态为 2xx，必需的 header（`content-type`、`vrpc-status` 和
`vrpc-server`）均有效，`vrpc-status` 为 `OK`，且响应信封包含 result 而不包含 error。
无法识别的大写状态码仍会保留在元数据中，以支持向前兼容。

- `*vrpc.InvocationError` 保留远端失败状态或非 2xx HTTP 响应、可选的 `ErrorPayload`，以及
  `Cause` 中的解码失败。
- `*vrpc.ProtocolError` 表示成功响应存在格式错误。

传输层与 context 错误会包装原始原因。使用 `errors.Is` 判断取消和超时错误，使用
`errors.As` 读取结构化错误。

`vrpc.ErrorPayload` 是共享结构 `transport/http.ErrorPayload` 的别名，包含 `Code`、
`Message`、`Reason` 和 `Detail`。

`ResponseMetadata` 提供 HTTP 状态、协议状态、服务端标识和响应 header。需要时可从
`ResponseMetadata.Header` 读取 `portal-trace-id`；判断远端结果时应以
`ResponseMetadata.Status` 为准，而不是消息文本或辅助的 `Code` 字段。共享传输层将解压后的
响应体限制为 128 MiB。协议错误信息不包含响应体。

## 共享传输层

`transport/http` 提供协议常量、header 校验、超时处理、JSON/CBOR 请求与响应编解码、
消息体限制和 HTTP 交互生命周期。适配层将 Go 值传给 `EncodeRequest` 与 `EncodeResponse`，
错误以 `*ErrorPayload` 表示。参数通过 `DecodeRequest` 解码；结果通过 `DecodeResponse`
取得，再用 `ResponsePayload.Unmarshal` 解码，错误则用 `ResponsePayload.DecodeError`
解码。这些操作遵循固定的协议规则：nil 集合编码为空集合，解码拒绝重复键。适配层负责传输
配置、应用元数据和错误策略。原始信封辅助函数保留已编码的载荷；
`ResponsePayload.EncodeWithError` 替换错误时，无需先解码再重新编码结果。

`ReadRequestBody` 按 `Content-Type` 限制编码后的完整请求信封：JSON 为 32 MiB，CBOR 为
128 MiB。它同时检查声明的长度和实际读取的字节数，未声明长度的请求也受限制。响应体上限
仍为 128 MiB。`RequestBodyLimit` 返回指定编码的上限。`MaxRequestBodyBytes` 已删除；原先
使用它的调用方应改用 `MaxJSONRequestBodyBytes` 或 `MaxCBORRequestBodyBytes`。这些是
传输层的固定限制，不是客户端选项。接收端必须使用更新后的 `transport/http`，才能接受
大于 32 MiB 的 CBOR 请求。

## 兼容性与范围

测试覆盖客户端调用、header、错误、取消以及共享 JSON/CBOR 协议实现。Portal 集成与网关
策略测试由 Vine 负责。

本客户端支持手写请求与结果类型，以及 skelc 生成的 Go API 客户端。生成的后端服务客户端
使用 Vine。Vine 通过其框架适配层使用共享传输层。本客户端不实现 Portal `/inspect`。

## 包结构

根目录的 `api.go` 通过类型别名和函数包装对外提供客户端 API。客户端、注册表、凭据与协议
元数据的实现位于 `internal`，调用方仍导入 `go.yorun.ai/vrpc`。`transport/http` 保持公开，
供 Vine 跨模块复用。

## 开发

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

CI 运行客户端与共享传输层测试，不依赖 Vine。
仓库边界见 [AGENTS.md](AGENTS.md)。

## Skel 标量类型

`go.yorun.ai/skel/types`（导入别名 `skeltype`）提供 Decimal、Binary、Timestamp、
Duration、LocalDate、LocalTime、LocalDateTime 和 JSON 类型及其构造函数。vRPC 复用这些
类型及对应的 JSON/CBOR 编解码，不再提供独立的 `go.yorun.ai/vrpc/skel` 包。这些类型的表示形式
与 Vine 契约一致，包括小数精度和基于字符串的日期时间值。生成的 API 客户端与手写调用方
使用相同类型，无需依赖 Vine。

共享的 `go.yorun.ai/skel/types/codec` 在 JSON 和 CBOR 中将 UUID 值和 map key 编码为
字符串，支持 `uuid.UUID` 和 `skeltype.UUID`。
