package quoteapiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// GetFxOverviewRequest 汇率概览请求参数（对应 get-fx-overview 接口）。
//
// 该接口只接受一个可选的毫秒时间戳：省略时请求不带查询串，签名输入为空对象 "{}"。
type GetFxOverviewRequest struct {
	// TimestampMillis 毫秒时间戳（查询串 _ 参数），为空则省略该参数。
	TimestampMillis string
}

// FxOverviewQuote 单个货币对的最新报价。
type FxOverviewQuote struct {
	// Time 报价时间，毫秒 Unix 时间戳。
	Time int64 `json:"time"`
	// PriceNominal 最新价，小数位随货币对精度变化。
	PriceNominal string `json:"priceNominal"`
	// Price 最新价；已观测样本中与 PriceNominal 始终相同，用途差异未验证。
	Price string `json:"price"`
}

// GetFxOverview 一次拉取 9 种货币两两组合的 72 个货币对最新价（对应 get-fx-overview 接口）。
//
// 返回值以货币对代码为键（如 USDCNH、JPYUSD）。键**没有顺序语义**，不要依赖迭代顺序；
// 货币对清单由服务端固定（不含贵金属与美元指数），无法按币种筛选。
//
// 省略时间戳时请求不带查询串，此时签名输入是空对象 "{}"，而不是空串——
// Sign 对空串有「降级为字面量 quote」的分支，走那条分支会得到不匹配的 token，
// 因此这里显式按 "{}" 计算。
func (c *Client) GetFxOverview(ctx context.Context, req GetFxOverviewRequest) (map[string]FxOverviewQuote, error) {
	params := make([]Param, 0, 1)
	if timestamp := strings.TrimSpace(req.TimestampMillis); timestamp != "" {
		params = append(params, Param{Name: "_", Value: timestamp})
	}
	signInput := SignInput(params)
	if signInput == "" {
		signInput = "{}"
	}

	path := c.cfg.FxOverviewPath
	body, err := c.do(ctx, http.MethodGet, path, params, nil, Sign(signInput), "")
	if err != nil {
		return nil, err
	}
	var payload fxOverviewPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &APIError{
			Path: path, URL: c.url(path, params), Body: truncate(string(body), 500),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	if err := decode(body, path, c.url(path, params), payload.Code, payload.Message); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

// fxOverviewPayload 汇率概览响应外层包裹；data 是以货币对代码为键的对象。
type fxOverviewPayload struct {
	Code    int                        `json:"code"`
	Message string                     `json:"message"`
	Data    map[string]FxOverviewQuote `json:"data"`
}
