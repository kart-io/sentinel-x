package response

import (
	"net/http"
	"time"

	"github.com/kart-io/sentinel-x/pkg/errors"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
	"github.com/kart-io/sentinel-x/pkg/validator"
)

// Writer provides convenient methods to write responses to transport.Context.
type Writer struct {
	ctx       transport.Context
	withTime  bool
	requestID string
	lang      string
}

// NewWriter creates a new response writer for the given context.
func NewWriter(ctx transport.Context) *Writer {
	return &Writer{ctx: ctx}
}

// WithTimestamp enables automatic timestamp in responses.
func (w *Writer) WithTimestamp() *Writer {
	w.withTime = true
	return w
}

// WithRequestID sets the request ID for responses.
func (w *Writer) WithRequestID(requestID string) *Writer {
	w.requestID = requestID
	return w
}

// WithLang sets the language for error messages.
func (w *Writer) WithLang(lang string) *Writer {
	w.lang = lang
	return w
}

// prepare adds optional fields to the response.
func (w *Writer) prepare(r *Response) *Response {
	if w.withTime {
		r.Timestamp = time.Now().UnixMilli()
	}
	if w.requestID != "" {
		r.RequestID = w.requestID
	}
	return r
}

// OK sends a successful response with data.
func (w *Writer) OK(data interface{}) {
	resp := w.prepare(Success(data))
	w.ctx.JSON(resp.HTTPStatus(), resp)
}

// OKWithMessage sends a successful response with custom message.
func (w *Writer) OKWithMessage(message string, data interface{}) {
	resp := w.prepare(SuccessWithMessage(message, data))
	w.ctx.JSON(resp.HTTPStatus(), resp)
}

// Fail sends an error response using Errno.
func (w *Writer) Fail(e *errors.Errno) {
	var resp *Response
	if w.lang != "" {
		resp = w.prepare(ErrWithLang(e, w.lang))
	} else {
		resp = w.prepare(Err(e))
	}
	w.ctx.JSON(e.HTTPStatus(), resp)
}

// FailWithLang sends an error response with language-specific message.
func (w *Writer) FailWithLang(e *errors.Errno, lang string) {
	resp := w.prepare(ErrWithLang(e, lang))
	w.ctx.JSON(e.HTTPStatus(), resp)
}

// FailWithCode sends an error response with code and message.
func (w *Writer) FailWithCode(code int, message string) {
	resp := w.prepare(ErrorWithCode(code, message))
	w.ctx.JSON(resp.HTTPStatus(), resp)
}

// FailWithError converts a standard error and sends it.
// If the error is an Errno, it uses it directly.
// Otherwise, it wraps it as ErrInternal.
func (w *Writer) FailWithError(err error) {
	e := errors.FromError(err)
	w.Fail(e)
}

// FailWithValidation sends a validation error response.
// It includes detailed validation error information in the response data.
func (w *Writer) FailWithValidation(verr *validator.ValidationErrors) {
	resp := w.prepare(&Response{
		Code:     errors.ErrValidationFailed.Code,
		HTTPCode: http.StatusBadRequest,
		Message:  verr.First(),
		Data:     verr.ToMap(),
	})
	w.ctx.JSON(http.StatusBadRequest, resp)
}

// FailWithBindOrValidation handles binding or validation errors appropriately.
// If err is a ValidationErrors, sends detailed validation error response.
// Otherwise, sends a generic invalid parameter error.
func (w *Writer) FailWithBindOrValidation(err error) {
	if verr, ok := err.(*validator.ValidationErrors); ok {
		w.FailWithValidation(verr)
		return
	}
	w.Fail(errors.ErrInvalidParam.WithMessage("invalid request body: " + err.Error()))
}

// PageOK sends a paginated response.
func (w *Writer) PageOK(list interface{}, total int64, page, pageSize int) {
	resp := w.prepare(Page(list, total, page, pageSize))
	w.ctx.JSON(resp.HTTPStatus(), resp)
}

// Send sends a custom response.
func (w *Writer) Send(r *Response) {
	resp := w.prepare(r)
	w.ctx.JSON(resp.HTTPStatus(), resp)
}

// ============================================================================
// Convenience functions that work directly with transport.Context
// ============================================================================

// OK sends a successful response.
func OK(ctx transport.Context, data interface{}) {
	NewWriter(ctx).OK(data)
}

// OKWithMessage sends a successful response with message.
func OKWithMessage(ctx transport.Context, message string, data interface{}) {
	NewWriter(ctx).OKWithMessage(message, data)
}

// Fail sends an error response using Errno.
func Fail(ctx transport.Context, e *errors.Errno) {
	NewWriter(ctx).Fail(e)
}

// FailWithLang sends an error response with language-specific message.
func FailWithLang(ctx transport.Context, e *errors.Errno, lang string) {
	NewWriter(ctx).FailWithLang(e, lang)
}

// FailWithCode sends an error response with code and message.
func FailWithCode(ctx transport.Context, code int, message string) {
	NewWriter(ctx).FailWithCode(code, message)
}

// FailWithError sends an error response from a standard error.
func FailWithError(ctx transport.Context, err error) {
	NewWriter(ctx).FailWithError(err)
}

// PageOK sends a paginated response.
func PageOK(ctx transport.Context, list interface{}, total int64, page, pageSize int) {
	NewWriter(ctx).PageOK(list, total, page, pageSize)
}

// FailWithValidation sends a validation error response.
func FailWithValidation(ctx transport.Context, verr *validator.ValidationErrors) {
	NewWriter(ctx).FailWithValidation(verr)
}

// FailWithBindOrValidation handles binding or validation errors.
// If err is a ValidationErrors, sends detailed validation error response.
// Otherwise, sends a generic invalid parameter error.
func FailWithBindOrValidation(ctx transport.Context, err error) {
	NewWriter(ctx).FailWithBindOrValidation(err)
}
