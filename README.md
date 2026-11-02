# MetaExt

Meta External Impl

## 环境要求

- Go 1.25 及以上

## 构建与运行

```bash
# 直接运行
go run .

# 编译到当前目录
go build -o MetaExt .

# 注入版本号
go build -ldflags "-X main.version=v0.1.0" -o MetaExt .
```

## 许可

本项目采用 GPL-3.0 许可，详见 [LICENSE](LICENSE)。
