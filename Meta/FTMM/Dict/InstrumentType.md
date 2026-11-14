# 证券主类型字典（InstrumentType）

> 所属：MetaExt 元数据字典 · 存放：`Meta/FTMM/Dict/InstrumentType.md`\
> 用途：富途 / moomoo（FTMM）数据中 `instrument_type` 字段的取值字典——**类型代码为键，品种大类中文名为值**。\

## 一、字典表

| 键 | 值 |
|-----:|------|
| `1` | 港股权证/牛熊证（HK Warrant/CBBC） |
| `3` | 股票（Stock/Equity） |
| `4` | ETF / 基金（交易所交易基金） |
| `5` | 权证（Warrant）/ 牛熊证（CBBC） |
| `6` | 港股衍生品（HK Derivatives，涡轮/界内证等） |
| `7` | 板块/指数成分股分类（Sector/Constituent） |
| `9` | 沪深港通标的（Stock Connect） |
| `10` | 期货（Futures） |
| `11` | 外汇（Forex） |
| `12` | 加密货币（Cryptocurrency） |
| `13` | 结构化产品（Structured Products，FCN/雪球等） |
| `14` | 富途结构化产品（GT 系列） |
| `15` | 债券指数 / 固收（Bond Index） |
| `16` | 期货价差合约（Futures Spread） |
| `18` | 仿真/测试（Simulation/Test） |
| `19` | 其他/特殊 |

## 二、说明

- 键：`instrument_type` 字段的**原始取值**，十进制整数字符串，取值不连续（`1` 起，最大到 `19`）。
- 值：证券主类型的中文名称。
- 排序：按原值升序。
- 与 `instrument_type_v2` 是两套编码，同一品种在 V2 中可能换号，对照见
  [InstrumentTypeV2.md](InstrumentTypeV2.md)。

## 三、维护约定

- 本文件描述的是**富途 / moomoo 侧的原始取值**，取值集合由上游决定；上游字典变更后同步本文件，
  不在此处自行裁剪或新增条目。
- 新增取值时保持原值升序。
