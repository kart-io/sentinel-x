// Package main is the entry point for the Sentinel-X RAG Service.
//
//	@title						Sentinel-X RAG API
//	@version					1.0
//	@description				RAG 知识库服务 - 基于 Milvus 向量数据库和 Ollama LLM
//	@termsOfService				https://github.com/kart-io/sentinel-x
//
//	@contact.name				Sentinel-X Team
//	@contact.url				https://github.com/kart-io/sentinel-x
//	@contact.email				support@sentinel-x.io
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:8082
//	@BasePath					/
//
//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Example: "Bearer {token}"
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	rag "github.com/kart-io/sentinel-x/internal/rag"
)

func main() {
	rag.NewApp().Run()
}
