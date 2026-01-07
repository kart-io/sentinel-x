// Package httputils provides HTTP utility functions.
package httputils

import (
	"github.com/gin-gonic/gin"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
	"github.com/kart-io/sentinel-x/pkg/utils/response"
)

// WriteResponse writes the response to the client.
// It handles both success and error cases, ensuring consistent response format.
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		var resp *response.Response
		if errno, ok := err.(*errors.Errno); ok {
			resp = response.Err(errno)
		} else {
			resp = response.Err(errors.ErrInternal.WithMessage(err.Error()))
		}
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	// data can be *response.Response (e.g. from response.Page) or raw data
	if resp, ok := data.(*response.Response); ok {
		defer response.Release(resp)
		c.JSON(resp.HTTPStatus(), resp)
		return
	}

	resp := response.Success(data)
	defer response.Release(resp)
	c.JSON(resp.HTTPStatus(), resp)
}
