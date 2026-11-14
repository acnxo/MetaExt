# 富途字段字典（FT Dict）

本目录按字段记录富途 / moomoo（FT）数据中各字段的取值字典，一个字段一个文件。文件名用大驼峰，
标题括号内与「说明」中的字段名沿用上游原始写法，如 `market_code`、`instrument_type_v2`。

## 字典一览

| 文件 | 字段 | 内容 | 状态 |
| --- | --- | --- | --- |
| [Region.md](Region.md) | `region` | 证券所属地区，25 个取值 | 现行 |
| [MarketCode.md](MarketCode.md) | `market_code` | 内部市场代码，54 个取值 | 现行 |
| [Exchange.md](Exchange.md) | `exchange` | 主要交易平台代码，31 个取值 | 现行 |
| [ListedExchange.md](ListedExchange.md) | `listed_exchange` | 具体上市交易所，49 个取值 | 现行 |
| [CsMarket.md](CsMarket.md) | `cs_market` | 市场分类代码，29 个取值 | 现行 |
| [CurrencyCode.md](CurrencyCode.md) | `currency_code` | 计价货币代码，20 个取值 | 现行 |
| [InstrumentType.md](InstrumentType.md) | `instrument_type` | 证券主类型，16 个取值 | 已过时，见 V2 |
| [InstrumentTypeV2.md](InstrumentTypeV2.md) | `instrument_type_v2` | 证券主类型 V2，20 个取值 | 现行 |
| [SubInstrumentType.md](SubInstrumentType.md) | `sub_instrument_type` | 证券子类型，65 个取值 | 已过时，见 V2 |
| [SubInstrumentTypeV2.md](SubInstrumentTypeV2.md) | `sub_instrument_type_v2` | 证券子类型 V2，91 个取值 | 现行 |

字段之间的关系：`region` 是国家/地区维度；`cs_market` 是市场分类，与 `exchange` 多对一；
`listed_exchange` 比 `exchange` 更细，用于区分同一市场内的子交易所；`market_code` 是内部编码，
按品种划分；`currency_code` 是计价货币；`instrument_type` 与 `sub_instrument_type` 是品种类型，
V2 版本重新归类并用 `主类型 * 1000 + 子类型编号` 统一编码。

## 统一结构

每份字典固定三段，顺序与写法一致：

| 小节 | 内容 |
| --- | --- |
| 一、字典表 | 键为上游原始取值，值为中文名，第三列按字段语义取交易所、地区、ISO 代码或对应编号 |
| 二、说明 | 解释键、值与第三列的口径，以及排序特征与易混点 |
| 三、维护约定 | 以上游为准、不自行裁剪，新增取值的插入规则 |

排序约定：数值型键按原值升序，代码型键按字母序，空值统一置末。

需要新增或修改字典时，按 [`Meta/README.md`](../README.md) 中「新增一个字段字典」一节的要求操作，
并同步更新本文件的「字典一览」。
