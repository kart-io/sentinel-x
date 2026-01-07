package middleware

import (
	"github.com/kart-io/sentinel-x/pkg/options"
	"github.com/spf13/pflag"
)

func init() {
	Register(MiddlewareSecurityHeaders, func() MiddlewareConfig {
		return NewSecurityHeadersOptions()
	})
}

// 确保 SecurityHeadersOptions 实现 MiddlewareConfig 接口。
var _ MiddlewareConfig = (*SecurityHeadersOptions)(nil)

// SecurityHeadersOptions 定义安全头中间件的配置选项。
type SecurityHeadersOptions struct {
	// EnableHSTS 启用 Strict-Transport-Security 头。
	EnableHSTS bool `json:"enable-hsts" mapstructure:"enable-hsts"`
	// HSTSMaxAge 是 HSTS max-age（秒）。
	HSTSMaxAge int `json:"hsts-max-age" mapstructure:"hsts-max-age"`
	// HSTSIncludeSubdomains 在 HSTS 中包含子域。
	HSTSIncludeSubdomains bool `json:"hsts-include-subdomains" mapstructure:"hsts-include-subdomains"`
	// HSTSPreload 启用 HSTS 预加载。
	HSTSPreload bool `json:"hsts-preload" mapstructure:"hsts-preload"`

	// EnableFrameOptions 启用 X-Frame-Options 头。
	EnableFrameOptions bool `json:"enable-frame-options" mapstructure:"enable-frame-options"`
	// FrameOptionsValue 是 X-Frame-Options 的值（DENY, SAMEORIGIN）。
	FrameOptionsValue string `json:"frame-options-value" mapstructure:"frame-options-value"`

	// EnableContentTypeOptions 启用 X-Content-Type-Options 头。
	EnableContentTypeOptions bool `json:"enable-content-type-options" mapstructure:"enable-content-type-options"`

	// EnableXSSProtection 启用 X-XSS-Protection 头。
	EnableXSSProtection bool `json:"enable-xss-protection" mapstructure:"enable-xss-protection"`
	// XSSProtectionValue 是 X-XSS-Protection 的值。
	XSSProtectionValue string `json:"xss-protection-value" mapstructure:"xss-protection-value"`

	// ContentSecurityPolicy 是 Content-Security-Policy 头的值。
	ContentSecurityPolicy string `json:"content-security-policy" mapstructure:"content-security-policy"`
	// ReferrerPolicy 是 Referrer-Policy 头的值。
	ReferrerPolicy string `json:"referrer-policy" mapstructure:"referrer-policy"`
}

// NewSecurityHeadersOptions 创建默认的安全头选项。
func NewSecurityHeadersOptions() *SecurityHeadersOptions {
	return &SecurityHeadersOptions{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,

		EnableFrameOptions: true,
		FrameOptionsValue:  "DENY",

		EnableContentTypeOptions: true,

		EnableXSSProtection: true,
		XSSProtectionValue:  "1; mode=block",

		ContentSecurityPolicy: "", // 默认为空（用户应配置）
		ReferrerPolicy:        "no-referrer",
	}
}

// AddFlags 为安全头选项添加标志到指定的 FlagSet。
func (o *SecurityHeadersOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := options.Join(prefixes...) + "middleware.security-headers."

	fs.BoolVar(&o.EnableHSTS, prefix+"enable-hsts", o.EnableHSTS, "Enable Strict-Transport-Security header.")
	fs.IntVar(&o.HSTSMaxAge, prefix+"hsts-max-age", o.HSTSMaxAge, "HSTS max-age in seconds.")
	fs.BoolVar(&o.HSTSIncludeSubdomains, prefix+"hsts-include-subdomains", o.HSTSIncludeSubdomains, "Include subdomains in HSTS.")
	fs.BoolVar(&o.HSTSPreload, prefix+"hsts-preload", o.HSTSPreload, "Enable HSTS preload.")

	fs.BoolVar(&o.EnableFrameOptions, prefix+"enable-frame-options", o.EnableFrameOptions, "Enable X-Frame-Options header.")
	fs.StringVar(&o.FrameOptionsValue, prefix+"frame-options-value", o.FrameOptionsValue, "X-Frame-Options header value (DENY, SAMEORIGIN).")

	fs.BoolVar(&o.EnableContentTypeOptions, prefix+"enable-content-type-options", o.EnableContentTypeOptions, "Enable X-Content-Type-Options header.")

	fs.BoolVar(&o.EnableXSSProtection, prefix+"enable-xss-protection", o.EnableXSSProtection, "Enable X-XSS-Protection header.")
	fs.StringVar(&o.XSSProtectionValue, prefix+"xss-protection-value", o.XSSProtectionValue, "X-XSS-Protection header value.")

	fs.StringVar(&o.ContentSecurityPolicy, prefix+"content-security-policy", o.ContentSecurityPolicy, "Content-Security-Policy header value.")
	fs.StringVar(&o.ReferrerPolicy, prefix+"referrer-policy", o.ReferrerPolicy, "Referrer-Policy header value.")
}

// Validate 验证安全头选项。
func (o *SecurityHeadersOptions) Validate() []error {
	if o == nil {
		return nil
	}
	// 安全头配置无强制要求，所有选项都是可选的
	return nil
}

// Complete 完成安全头选项的默认值设置。
func (o *SecurityHeadersOptions) Complete() error {
	return nil
}
