// Package webserver 测试端HTTP服务器，提供静态文件服务与REST API。
package webserver

import "embed"

//go:embed all:web
var embeddedFS embed.FS
