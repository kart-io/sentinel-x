// Package middleware provides version endpoint registration.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	versionopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
	"github.com/kart-io/version"
)

// VersionResponse represents the version endpoint response.
type VersionResponse struct {
	ServiceName  string `json:"service_name,omitempty"`
	GitVersion   string `json:"git_version"`
	GitCommit    string `json:"git_commit,omitempty"`
	GitBranch    string `json:"git_branch,omitempty"`
	GitTreeState string `json:"git_tree_state,omitempty"`
	BuildDate    string `json:"build_date,omitempty"`
	GoVersion    string `json:"go_version,omitempty"`
	Compiler     string `json:"compiler,omitempty"`
	Platform     string `json:"platform,omitempty"`
}

// RegisterVersionRoutes registers the version endpoint.
func RegisterVersionRoutes(router transport.Router, opts versionopts.VersionOptions) {
	if !opts.Enabled {
		return
	}

	// 确保路径有效
	if opts.Path == "" {
		opts.Path = "/version"
	}

	router.Handle(http.MethodGet, opts.Path, func(c *gin.Context) {
		info := version.Get()

		resp := VersionResponse{
			GitVersion: info.GitVersion,
		}

		// 根据 HideDetails 选项决定是否显示详细信息
		if !opts.HideDetails {
			resp.ServiceName = info.ServiceName
			resp.GitCommit = info.GitCommit
			resp.GitBranch = info.GitBranch
			resp.GitTreeState = info.GitTreeState
			resp.BuildDate = info.BuildDate
			resp.GoVersion = info.GoVersion
			resp.Compiler = info.Compiler
			resp.Platform = info.Platform
		}

		c.JSON(http.StatusOK, resp)
	})
}
