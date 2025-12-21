// Package main is the entry point for the Sentinel-X API server.
//
//	@title						Sentinel-X API
//	@version					1.0
//	@description				Sentinel-X 平台 API 服务
//	@termsOfService				https://github.com/kart-io/sentinel-x
//
//	@contact.name				Sentinel-X Team
//	@contact.url				https://github.com/kart-io/sentinel-x
//	@contact.email				support@sentinel-x.io
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:8100
//	@BasePath					/api/v1
//
//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Example: "Bearer {token}"
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	api "github.com/kart-io/sentinel-x/internal/api"
)

func main() {
	api.NewApp().Run()
}
