// Package web provides embedded static files for the API Gateway
package web

import "embed"

//go:embed index.html config.js
var StaticFiles embed.FS
