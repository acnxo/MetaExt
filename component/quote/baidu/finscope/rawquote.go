package finscope

import (
	"bytes"
	"encoding/json"
)

// RawQuoteResult 是「响应结构尚未验证」的接口的返回。
//
// 外汇分钟、期货分钟、期货五日三个接口在本机持续风控，文档也没能确认 Result 内部结构。
// 与其照着股票接口编造字段，不如把 Result 原样交给调用方：拿到真实样本后再补结构化解析，
// 在那之前调用方需要自己处理 json.RawMessage。
type RawQuoteResult struct {
	// QueryID 请求追踪 ID。
	QueryID string
	// ResultCode 网关层结果，0 表示请求被正常处理。
	ResultCode int
	// Result 原始业务数据，结构未确认。
	Result json.RawMessage
	// URL 本次请求地址，便于对照抓包记录。
	URL string
}

// parseRawQuote 解析「只确认了外层信封」的响应：识别风控与结构异常，Result 原样返回。
func parseRawQuote(path, targetURL string, body []byte) (*RawQuoteResult, error) {
	var envelope kLineEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, &APIError{
			Path: path, URL: targetURL, Body: truncate(string(body), errorBodyLimit),
			Message: "响应解析失败: " + err.Error(),
		}
	}
	resultRaw := bytes.TrimSpace(envelope.Result)
	if len(resultRaw) > 0 && resultRaw[0] == '{' {
		var risk riskControl
		if err := json.Unmarshal(resultRaw, &risk); err == nil && risk.Code != 0 {
			return nil, &APIError{
				Path: path, URL: targetURL, Code: risk.Code, Message: risk.Message,
				Body: truncate(string(body), errorBodyLimit),
			}
		}
	}
	return &RawQuoteResult{
		QueryID:    envelope.QueryID,
		ResultCode: envelope.ResultCode,
		Result:     json.RawMessage(resultRaw),
		URL:        targetURL,
	}, nil
}
