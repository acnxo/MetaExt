// Package indexapi 封装万得官网的指数日线接口。
//
// 当前实现指数日线（`/indexofficialwebsite/Kline`）：按官网内部编码 `indexId` 返回
// 最近约 60 个交易日的开高低收、成交量、成交额与涨跌幅。
//
// 与 Wind 终端的 WindPy 不同，这是**公开网页 API**：不需要安装终端、不需要登录，
// 因此可以直接在 CI 里运行。
//
// 协议要点：
//
//   - 只返回最近约 60 个交易日，增量更新时必须与本地数据按日期合并去重；
//   - `indexId` 是官网内部编码（Base64 形态），与 `windCode` 不是一套，本包内置映射；
//   - 最高价字段名拼写为 **`hight`**（不是 `high`）；
//   - SSL 证书是 `*.wind.com.cn`，与 `index_api.wind.com.cn` 不匹配：
//     本包关闭主机名校验，但仍通过 VerifyConnection 做证书链校验。
package indexapi

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acnxo/MetaExt/component/quote/internal/fetch"
)

// 协议默认值。
const (
	// DefaultBaseURL 站点地址。
	DefaultBaseURL = "https://index_api.wind.com.cn"
	// DefaultIndexKLinePath 指数日线路径。
	DefaultIndexKLinePath = "/indexofficialwebsite/Kline"
	// DefaultLanguage 语言参数，固定 cn。
	DefaultLanguage = "cn"
)

// DefaultIndexIDs 是 windCode → 官网内部编码的映射。
//
// indexId 是官网前端的内部标识（Base64 形态），无法从 windCode 推导，只能内置。
var DefaultIndexIDs = map[string]string{
	"868008.WI":  "tShUKnLO4cLQ8v+TnUd2uQ==",
	"8841431.WI": "2O5u8vTCa4KiLTLQ7DT/9w==",
}

// Config 是客户端配置；零值代表使用协议默认值。
type Config struct {
	// BaseURL 站点地址，默认 DefaultBaseURL。
	BaseURL string
	// IndexKLinePath 指数日线路径，默认 DefaultIndexKLinePath。
	IndexKLinePath string
	// IndexIDs windCode → indexId 映射，为空时取 DefaultIndexIDs。
	IndexIDs map[string]string
	// Timeout 单次请求超时，默认 20 秒。
	Timeout time.Duration
	// Retries 请求尝试次数，默认 4。
	Retries int
	// HTTPClient 可注入自定义客户端；注入时不再使用本包的 TLS 配置。
	HTTPClient *http.Client
	// Sleep 可注入等待函数，测试时用来跳过退避等待。
	Sleep func(time.Duration)
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	if strings.TrimSpace(c.IndexKLinePath) == "" {
		c.IndexKLinePath = DefaultIndexKLinePath
	}
	if !strings.HasPrefix(c.IndexKLinePath, "/") {
		c.IndexKLinePath = "/" + c.IndexKLinePath
	}
	if len(c.IndexIDs) == 0 {
		c.IndexIDs = DefaultIndexIDs
	}
	return c
}

// Client 是万得官网指数客户端。
//
// Client 并发安全：创建后仅读取配置，内部不保存可变状态。
type Client struct {
	cfg Config
	req *fetch.Fetcher
}

// New 创建客户端，零值配置按协议默认值补齐。
func New(cfg Config) *Client {
	cfg = cfg.withDefaults()
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout:   cfg.Timeout,
			Transport: &http.Transport{TLSClientConfig: newTLSConfig()},
		}
	}
	return &Client{
		cfg: cfg,
		req: fetch.New(fetch.Config{
			Timeout:    cfg.Timeout,
			Retries:    cfg.Retries,
			HTTPClient: client,
			Sleep:      cfg.Sleep,
		}),
	}
}

// newTLSConfig 返回「只跳过主机名校验、保留证书链校验」的 TLS 配置。
//
// 站点证书为 *.wind.com.cn，与 index_api.wind.com.cn 不匹配。直接
// InsecureSkipVerify 会把证书链校验也跳过，因此这里改用 VerifyConnection 手工校验链。
func newTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // 仅关闭默认的「链 + 主机名」校验，下面补回链校验
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("证书链为空")
			}
			options := x509.VerifyOptions{
				Roots:         nil, // 使用系统根证书
				Intermediates: x509.NewCertPool(),
				// DNSName 留空即跳过主机名校验
			}
			for _, cert := range state.PeerCertificates[1:] {
				options.Intermediates.AddCert(cert)
			}
			if _, err := state.PeerCertificates[0].Verify(options); err != nil {
				return fmt.Errorf("证书链校验失败: %w", err)
			}
			return nil
		},
	}
}

// Config 返回生效配置（已补齐默认值）。
func (c *Client) Config() Config { return c.cfg }

// APIError 表示接口没有返回可用数据。
type APIError struct {
	// Path 接口路径。
	Path string
	// URL 完整请求地址。
	URL string
	// HTTPStatus HTTP 状态码，请求未到达服务端时为 0。
	HTTPStatus int
	// Message 错误说明。
	Message string
	// Body 响应体，已截断。
	Body string
	// Err 被包装的底层错误，可能为 nil。
	Err error
}

func (e *APIError) Error() string {
	switch {
	case e.HTTPStatus != 0:
		return fmt.Sprintf("万得官网 %s 请求失败：HTTP %d %s", e.Path, e.HTTPStatus, e.Message)
	case e.Message != "":
		return fmt.Sprintf("万得官网 %s 请求失败：%s", e.Path, e.Message)
	default:
		return "万得官网 " + e.Path + " 请求失败"
	}
}

// Unwrap 返回被包装的底层错误。
func (e *APIError) Unwrap() error { return e.Err }
