# 上市交易所字典（listed_exchange）

> 所属：MetaExt 元数据字典 · 存放：`Meta/FT/Dict/ListedExchange.md`\
> 用途：富途 / moomoo（FTMM）数据中 `listed_exchange` 字段的取值字典——**上市交易所代码为键，交易所中文名为值**。\

## 一、字典表

| 键 | 值 | 地区 |
|------|------|------|
| `AMEX` | 美国证券交易所（NYSE American） | 美国 |
| `ARCA` | NYSE Arca（电子交易所，ETF 为主） | 美国 |
| `ASX` | 澳大利亚证券交易所 | 澳大利亚 |
| `BATS` | BATS 全球市场（Cboe BZX） | 美国 |
| `BFL` | Bitflyer 交易所 | 全球 |
| `BFX` | Bitfinex 交易所 | 全球 |
| `BGO` | Bitget 交易所 | 全球 |
| `BIN` | Binance（币安交易所） | 全球 |
| `BIU` | BitMart/BIU 交易所 | 全球 |
| `BMD` | 马来西亚衍生品交易所 | 马来西亚 |
| `BMS` | 债券市场系统 | — |
| `BSE` | 北京证券交易所（北交所，Beijing Stock Exchange） | 中国 |
| `BST` | Bitstamp 交易所 | 全球 |
| `BTK` | Bybit/BTK 交易所 | 全球 |
| `CBOE` | 芝加哥期权交易所 | 美国 |
| `CBOT` | 芝加哥期货交易所 | 美国 |
| `CME` | 芝加哥商业交易所 | 美国 |
| `COIN` | Coinbase 交易所 | 全球 |
| `COMEX` | 纽约商品交易所（贵金属期货） | 美国 |
| `COMPOSITE` | 加密货币综合 | 全球 |
| `CSE` | 加拿大证券交易所（Canadian Securities Exchange） | 加拿大 |
| `DDEX` | 去中心化交易所 | 全球 |
| `FIGR` | Figr/测试 | — |
| `FTCC` | 富途加密货币 | — |
| `GEM` | Gate.io 交易所 | 全球 |
| `HBI` | Huobi/HTX 交易所 | 全球 |
| `HKFE` | 香港期货交易所 | 香港 |
| `HPLD` | Hyperliquid 交易所 | — |
| `HSK` | HashKey 交易所 | 全球 |
| `KRX` | 韩国交易所（Korea Exchange） | 韩国 |
| `KUC` | KuCoin 交易所 | 全球 |
| `LBK` | LBank 交易所 | 全球 |
| `NASDAQ` | 纳斯达克交易所 | 美国 |
| `NEO` | NEO 交易所 | 加拿大 |
| `NONE` | 无上市交易所（结构性产品/基金内部编码） | — |
| `NYMEX` | 纽约商品交易所 | 美国 |
| `NYSE` | 纽约证券交易所 | 美国 |
| `OKX` | OKX 交易所 | 全球 |
| `OSE` | 大阪交易所 | 日本 |
| `OTC` | 场外交易市场（Over-The-Counter） | 美国 |
| `PT` | Poloniex 交易所 | 全球 |
| `SEHK` | 香港交易所 | 香港 |
| `SGX` | 新加坡交易所 | 新加坡 |
| `SSE` | 上海证券交易所 | 中国 |
| `SZSE` | 深圳证券交易所 | 中国 |
| `TSE` | 东京证券交易所（Tokyo Stock Exchange） | 日本 |
| `TSX` | 多伦多证券交易所（Toronto Stock Exchange） | 加拿大 |
| `TSXV` | TSX Venture 创业板 | 加拿大 |
| `—` | 未填写或不适用（美股主记录、指数、加密货币等） | — |

## 二、说明

- 键：`listed_exchange` 字段的**原始取值**，交易所代码（大写字母）；`—` 表示空值。
- 值：具体上市交易所或交易平台的名称。
- 地区：该交易所所属地区；`—` 表示不适用。
- 排序：按代码字母升序，空值置末。
- 比 `exchange` 更细：`exchange=US` 的证券在这里区分 NYSE / NASDAQ / ARCA / BATS / OTC / AMEX。
- 加密货币的 `listed_exchange` 标识具体平台（`BIN`=Binance、`OKX`、`HBI`=Huobi 等）。
- 空值包括美股主记录（已有 `exchange=US`）、指数、债券等无需细化的品种；`NONE` 表示无实际上市交易所
  （结构性产品、基金内部编码）。

## 三、维护约定

- 本文件描述的是**富途 / moomoo 侧的原始取值**，取值集合由上游决定；上游字典变更后同步本文件，
  不在此处自行裁剪或新增条目。
- 新增取值时按代码字母序插入，空值保持在末。
