// Package thirdparty provides error codes for third-party services.
//
// This is an example of how to define error codes for external service integrations.
// Third-party service codes use range 90-99.
package thirdparty

import (
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"google.golang.org/grpc/codes"
)

// ============================================================================
// Third-party Service Codes
// ============================================================================

const (
	// ServiceThirdPartyPayment is for payment services.
	ServiceThirdPartyPayment = 90

	// ServiceThirdPartySMS is for SMS services.
	ServiceThirdPartySMS = 91

	// ServiceThirdPartyEmail is for email services.
	ServiceThirdPartyEmail = 92

	// ServiceThirdPartyStorage is for storage services.
	ServiceThirdPartyStorage = 93
)

func init() {
	errors.RegisterService(ServiceThirdPartyPayment, "payment-service")
	errors.RegisterService(ServiceThirdPartySMS, "sms-service")
	errors.RegisterService(ServiceThirdPartyEmail, "email-service")
	errors.RegisterService(ServiceThirdPartyStorage, "storage-service")
}

// ============================================================================
// Payment Service Errors (Service Code: 90)
// ============================================================================

var (
	// ErrPaymentFailed indicates payment failed.
	ErrPaymentFailed = errors.NewInternalErr(ServiceThirdPartyPayment, 1,
		"Payment failed", "支付失败")

	// ErrPaymentRefundFailed indicates refund failed.
	ErrPaymentRefundFailed = errors.NewInternalErr(ServiceThirdPartyPayment, 2,
		"Refund failed", "退款失败")

	// ErrPaymentInsufficientBalance indicates insufficient balance.
	ErrPaymentInsufficientBalance = errors.NewRequestErr(ServiceThirdPartyPayment, 1,
		"Insufficient balance", "余额不足")

	// ErrPaymentOrderNotFound indicates payment order not found.
	ErrPaymentOrderNotFound = errors.NewNotFoundErr(ServiceThirdPartyPayment, 1,
		"Payment order not found", "支付订单不存在")

	// ErrPaymentTimeout indicates payment timeout.
	ErrPaymentTimeout = errors.NewTimeoutErr(ServiceThirdPartyPayment, 1,
		"Payment timeout", "支付超时")

	// ErrPaymentChannelUnavailable indicates payment channel unavailable.
	ErrPaymentChannelUnavailable = errors.NewNetworkErr(ServiceThirdPartyPayment, 1,
		"Payment channel unavailable", "支付渠道不可用")
)

// ============================================================================
// SMS Service Errors (Service Code: 91)
// ============================================================================

var (
	// ErrSMSFailed indicates SMS send failed.
	ErrSMSFailed = errors.NewInternalErr(ServiceThirdPartySMS, 1,
		"SMS send failed", "短信发送失败")

	// ErrSMSLimitExceeded indicates SMS limit exceeded.
	ErrSMSLimitExceeded = errors.NewRateLimitErr(ServiceThirdPartySMS, 1,
		"SMS limit exceeded", "短信发送次数超限")

	// ErrSMSTimeout indicates SMS service timeout.
	ErrSMSTimeout = errors.NewTimeoutErr(ServiceThirdPartySMS, 1,
		"SMS service timeout", "短信服务超时")

	// ErrSMSInvalidPhone indicates invalid phone number for SMS.
	ErrSMSInvalidPhone = errors.NewRequestErr(ServiceThirdPartySMS, 1,
		"Invalid phone number", "手机号无效")

	// ErrSMSTemplateNotFound indicates SMS template not found.
	ErrSMSTemplateNotFound = errors.NewNotFoundErr(ServiceThirdPartySMS, 1,
		"SMS template not found", "短信模板不存在")

	// ErrSMSChannelUnavailable indicates SMS channel unavailable.
	ErrSMSChannelUnavailable = errors.NewNetworkErr(ServiceThirdPartySMS, 1,
		"SMS channel unavailable", "短信通道不可用")
)

// ============================================================================
// Email Service Errors (Service Code: 92)
// ============================================================================

var (
	// ErrEmailFailed indicates email send failed.
	ErrEmailFailed = errors.NewInternalErr(ServiceThirdPartyEmail, 1,
		"Email send failed", "邮件发送失败")

	// ErrEmailTimeout indicates email service timeout.
	ErrEmailTimeout = errors.NewTimeoutErr(ServiceThirdPartyEmail, 1,
		"Email service timeout", "邮件服务超时")

	// ErrEmailInvalidAddress indicates invalid email address.
	ErrEmailInvalidAddress = errors.NewRequestErr(ServiceThirdPartyEmail, 1,
		"Invalid email address", "邮箱地址无效")

	// ErrEmailTemplateNotFound indicates email template not found.
	ErrEmailTemplateNotFound = errors.NewNotFoundErr(ServiceThirdPartyEmail, 1,
		"Email template not found", "邮件模板不存在")

	// ErrEmailLimitExceeded indicates email limit exceeded.
	ErrEmailLimitExceeded = errors.NewRateLimitErr(ServiceThirdPartyEmail, 1,
		"Email limit exceeded", "邮件发送次数超限")

	// ErrEmailServerUnavailable indicates email server unavailable.
	ErrEmailServerUnavailable = errors.NewNetworkErr(ServiceThirdPartyEmail, 1,
		"Email server unavailable", "邮件服务器不可用")
)

// ============================================================================
// Storage Service Errors (Service Code: 93)
// ============================================================================

var (
	// ErrStorageUploadFailed indicates upload failed.
	ErrStorageUploadFailed = errors.NewInternalErr(ServiceThirdPartyStorage, 1,
		"Upload failed", "上传失败")

	// ErrStorageDownloadFailed indicates download failed.
	ErrStorageDownloadFailed = errors.NewInternalErr(ServiceThirdPartyStorage, 2,
		"Download failed", "下载失败")

	// ErrStorageDeleteFailed indicates delete failed.
	ErrStorageDeleteFailed = errors.NewInternalErr(ServiceThirdPartyStorage, 3,
		"Delete failed", "删除失败")

	// ErrStorageFileNotFound indicates file not found.
	ErrStorageFileNotFound = errors.NewNotFoundErr(ServiceThirdPartyStorage, 1,
		"File not found", "文件不存在")

	// ErrStorageFileTooLarge indicates file too large.
	ErrStorageFileTooLarge = errors.NewError(ServiceThirdPartyStorage, errors.CategoryRequest, 1,
		413, codes.InvalidArgument,
		"File too large", "文件过大")

	// ErrStorageInvalidFileType indicates invalid file type.
	ErrStorageInvalidFileType = errors.NewRequestErr(ServiceThirdPartyStorage, 2,
		"Invalid file type", "文件类型无效")

	// ErrStorageQuotaExceeded indicates storage quota exceeded.
	ErrStorageQuotaExceeded = errors.NewRateLimitErr(ServiceThirdPartyStorage, 1,
		"Storage quota exceeded", "存储配额已用尽")

	// ErrStorageTimeout indicates storage service timeout.
	ErrStorageTimeout = errors.NewTimeoutErr(ServiceThirdPartyStorage, 1,
		"Storage service timeout", "存储服务超时")

	// ErrStorageUnavailable indicates storage service unavailable.
	ErrStorageUnavailable = errors.NewNetworkErr(ServiceThirdPartyStorage, 1,
		"Storage service unavailable", "存储服务不可用")
)
