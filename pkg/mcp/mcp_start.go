package mcp

import (
	"context"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"github.com/weibaohui/k8m/pkg/constants"
	"github.com/weibaohui/kom/mcp"
	"github.com/weibaohui/kom/mcp/metadata"
)

func Start(version string, port int) {
	var ctxFn = func(ctx context.Context, r *http.Request) context.Context {
		username := r.Header.Get(constants.JwtUserName)
		role := r.Header.Get(constants.JwtUserRole)
		// 发起mcp调用请求时注入用户名、角色信息
		ctx = context.WithValue(ctx, constants.JwtUserName, username)
		ctx = context.WithValue(ctx, constants.JwtUserRole, role)
		return ctx
	}
	cfg := metadata.ServerConfig{
		Name:    "k8m mcp server",
		Version: version,
		Port:    port,
		ServerOptions: []server.ServerOption{
			server.WithResourceCapabilities(false, false),
			server.WithPromptCapabilities(false),
			server.WithLogging(),
		},
		SSEOption: []server.SSEOption{
			server.WithSSEContextFunc(ctxFn),
		},
		AuthKey:     constants.JwtUserName,
		AuthRoleKey: constants.JwtUserRole,
	}
	mcp.RunMCPServerWithOption(&cfg)
}
