# MetaExt

Meta External Impl —— 外部数据接口的 Go 实现集合。

当前已实现 moomoo（富途）网页行情接口 `/quote-api/quote-v2/*` 的客户端封装：按网页端的
签名规则生成 `quote-token`，拉取 K 线数据并转换为统一结构。

## 环境要求

- Go 1.25 及以上

## 构建与运行

程序入口位于 `cmd/starter/`。

```bash
# 直接运行
go run ./cmd/starter

# 编译到当前目录
go build -o MetaExt ./cmd/starter

# 注入版本号
go build -ldflags "-X main.version=v0.1.0" -o MetaExt ./cmd/starter
```

## 目录结构

```
cmd/
└── starter/            程序入口
component/              按「品类 → 渠道」组织的接口实现
└── quote/
    └── moomoo/
        └── quoteapiv2/ moomoo quote-v2 客户端（含签名、K 线）
base/                   最底层公共基座，不依赖本项目其他包
model/                  跨层共享的数据模型
sdk/                    外部服务客户端封装
ext/                    对外扩展点的接口契约与注册机制
util/                   无状态通用工具函数
common/                 跨包共享的常量、枚举与轻量值对象
infra/                  配置、日志、存储等基础设施实现与装配
```

## 包职责

| 包 | 职责 |
| --- | --- |
| `component` | 具体外部接口的实现，按品类与渠道分层组织 |
| `base` | 最底层公共基座（时间口径、标识约定、错误包装），不依赖本项目其他包 |
| `model` | 纯数据结构模型，不含网络与存储行为 |
| `sdk` | 外部服务客户端：请求构造、签名、重试与响应解析 |
| `ext` | 扩展点契约与注册机制，运行期替换或增补行为 |
| `util` | 无状态的纯函数工具，不引用 model、sdk、infra |
| `common` | 公共常量、枚举与轻量值对象 |
| `infra` | 配置、日志、存储、缓存等实现，仅由入口层装配引用 |

## 已实现接口

### moomoo quote-v2 K 线

```go
client := quoteapiv2.New(quoteapiv2.Config{})
bars, err := client.Kline(ctx, quoteapiv2.KlineRequest{
    StockID:           "1234",
    MarketType:        "1",
    MarketCode:        "1",
    InstrumentType:    "3",
    SubInstrumentType: "0",
})
```

- 五个定位参数原样透传，不做语义换算
- 接口一次返回全部历史数据、无分页，结果按时间升序
- 价格保留原始字符串，避免浮点精度损失
- 错误按 HTTP 状态与业务码分类，可用 `IsTokenRejected` 判断签名被拒

## 许可

本项目采用 GPL-3.0 许可，详见 [LICENSE](LICENSE)。
