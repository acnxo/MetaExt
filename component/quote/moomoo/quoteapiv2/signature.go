package quoteapiv2

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"strings"
)

// 本文件实现请求签名：把有序查询参数拼成紧凑 JSON，再经 HMAC-SHA512 与 SHA-256
// 两段截断得到 quote-token。签名对「键序 + 值字符串形式」逐字节敏感，改动前务必
// 先确认协议，否则会出现 token 被服务端拒绝、但参数看起来完全正确的情况。

// 签名相关常量。
const (
	// signKey HMAC 密钥，取自网页前端产物，不是用户凭据。
	signKey = "quote_web"
	// hmacTruncLen HMAC-SHA512 十六进制结果截取长度。
	hmacTruncLen = 10
	// tokenLen 最终 quote-token 长度。
	tokenLen = 10
	// emptyInputFallback 空输入时的降级字面量，与前端行为一致。
	emptyInputFallback = "quote"
)

// Param 有序查询参数；值为已字符串化的结果。
type Param struct {
	Name  string
	Value string
}

// Params 按「名称, 值, 名称, 值...」的顺序构造参数列表。
func Params(pairs ...string) []Param {
	params := make([]Param, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		params = append(params, Param{Name: pairs[i], Value: pairs[i+1]})
	}
	return params
}

// SignInput 把有序参数拼成紧凑 JSON 字符串作为签名输入。
//
// 这里不用 encoding/json：对象键序、紧凑格式与转义方式都是签名契约的一部分，
// 标准库会对键排序或改写格式，直接导致 token 与服务端不一致。
func SignInput(params []Param) string {
	if len(params) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(params) * 24)
	b.WriteByte('{')
	for i, p := range params {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(escapeJSONString(p.Name))
		b.WriteString(`":"`)
		b.WriteString(escapeJSONString(p.Value))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

// Sign 计算 quote-token：HMAC-SHA512 → 取前 10 位十六进制 → SHA-256 → 取前 10 位。
//
// 空输入按协议降级为字面量 "quote"。
func Sign(input string) string {
	if len(input) == 0 {
		input = emptyInputFallback
	}
	mac := hmac.New(sha512.New, []byte(signKey))
	mac.Write([]byte(input))
	first := hex.EncodeToString(mac.Sum(nil))[:hmacTruncLen]

	sum := sha256.Sum256([]byte(first))
	return hex.EncodeToString(sum[:])[:tokenLen]
}

// SignParams 计算有序参数对应的 quote-token。
func SignParams(params []Param) string { return Sign(SignInput(params)) }

// SignedQuery 按参数原顺序拼查询串。
//
// 不能用 url.Values.Encode：它会按键名排序，破坏签名键序。
func SignedQuery(params []Param) string {
	var b strings.Builder
	for i, p := range params {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(p.Name))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(p.Value))
	}
	return b.String()
}

// escapeJSONString 做 JSON 最小转义，非 ASCII 字符保留原始 UTF-8 形式。
func escapeJSONString(s string) string {
	if !strings.ContainsAny(s, `"\\`+"\n\r\t") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
