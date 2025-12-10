// Package helloservice provides the HelloService implementation.
package helloservice

import (
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// Service code for HelloService.
// Using 80 as it's in the internal service range (80-89).
const ServiceHello = 80

func init() {
	errors.RegisterService(ServiceHello, "hello-service")
}

// HelloService specific errors.
var (
	// ErrEmptyName indicates name is empty.
	ErrEmptyName = errors.NewRequestErr(ServiceHello, 1,
		"Name cannot be empty", "名称不能为空")

	// ErrNameTooLong indicates name is too long.
	ErrNameTooLong = errors.NewRequestErr(ServiceHello, 2,
		"Name is too long", "名称过长")

	// ErrGreetingFailed indicates greeting generation failed.
	ErrGreetingFailed = errors.NewInternalErr(ServiceHello, 1,
		"Failed to generate greeting", "生成问候语失败")
)
