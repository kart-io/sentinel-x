// Package main is the entry point for the Sentinel-X User Center Service.
//
//	@title						Sentinel-X User Center API
//	@version					1.0
//	@description				用户中心服务 API - 提供用户管理、认证、角色管理等功能
//	@termsOfService				https://github.com/kart-io/sentinel-x
//
//	@contact.name				Sentinel-X Team
//	@contact.url				https://github.com/kart-io/sentinel-x
//	@contact.email				support@sentinel-x.io
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:8081
//	@BasePath					/
//
//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Example: "Bearer {token}"
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	"github.com/kart-io/sentinel-x/cmd/user-center/app"
)

func main() {
	app.NewApp().Run()
}
