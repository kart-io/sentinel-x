// Package helloservice provides the HelloService implementation.
package helloservice

import (
	"github.com/kart-io/sentinel-x/pkg/errors"
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
	ErrEmptyName = errors.NewRequestError(ServiceHello, 1).
			Message("Name cannot be empty", "名称不能为空").
			MustBuild()

	// ErrNameTooLong indicates name is too long.
	ErrNameTooLong = errors.NewRequestError(ServiceHello, 2).
			Message("Name is too long", "名称过长").
			MustBuild()

	// ErrGreetingFailed indicates greeting generation failed.
	ErrGreetingFailed = errors.NewInternalError(ServiceHello, 1).
				Message("Failed to generate greeting", "生成问候语失败").
				MustBuild()
)
