// Package main is the entry point for the Swagger documentation server.
package main

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	docs_apisvc "github.com/kart-io/sentinel-x/api/swagger/apisvc"
	docs_rag "github.com/kart-io/sentinel-x/api/swagger/rag"
	docs_usercenter "github.com/kart-io/sentinel-x/api/swagger/user-center"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//go:embed index.html
var indexHTML embed.FS

func main() {
	r := gin.Default()

	// 统一入口页面
	r.GET("/", func(c *gin.Context) {
		t, err := template.ParseFS(indexHTML, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if err := t.Execute(c.Writer, nil); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
	})

	// API 服务文档
	// docs_api.SwaggerInfo.InstanceName() 默认为 "swagger"
	// API 服务文档 (InstanceName: apisvc)
	// 原 swaggerFiles.Handler 不带 Prefix，直接使用可能会导致路径解析错误（取决于 gin-swagger 内部处理）
	// 这里手动设置 Prefix 以确保静态文件能被正确找到
	apiHandler := *swaggerFiles.Handler
	apiHandler.Prefix = "/swagger/apisvc"
	r.GET("/swagger/apisvc/*any", ginSwagger.WrapHandler(&apiHandler, ginSwagger.InstanceName("apisvc")))

	// User Center 服务文档
	r.GET("/swagger/user-center/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("usercenter")))

	// RAG 服务文档
	r.GET("/swagger/rag/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("rag")))

	// 设置 Info
	docs_apisvc.SwaggerInfoapisvc.Title = "Sentinel-X API Service"
	docs_apisvc.SwaggerInfoapisvc.BasePath = "/api/v1"

	docs_usercenter.SwaggerInfousercenter.Title = "Sentinel-X User Center Service"
	docs_usercenter.SwaggerInfousercenter.BasePath = "/v1"

	docs_rag.SwaggerInforag.Title = "Sentinel-X RAG Service"
	docs_rag.SwaggerInforag.BasePath = "/"

	if err := r.Run(":8082"); err != nil {
		panic(err)
	}
}
