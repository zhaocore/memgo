package api

import (
	_ "embed"
	"net/http"
)

//go:embed static/openapi.json
var openapiJSON []byte

// serveOpenAPI 提供从 Python server 采集的 openapi 合同快照 (R3: 静态 golden 直出)。
func (s *Server) serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(openapiJSON)
}

// serveDocs /serveRedoc: 静态 HTML (套件只断言可达; UI 经 CDN 加载)。
func (s *Server) serveDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docsHTML))
}

func (s *Server) serveRedoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(redocHTML))
}

const docsHTML = `<!doctype html>
<html><head><title>Mem0 REST APIs - Swagger UI</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url: "/openapi.json", dom_id: "#swagger-ui"});</script>
</body></html>`

const redocHTML = `<!doctype html>
<html><head><title>Mem0 REST APIs - ReDoc</title></head>
<body><redoc spec-url="/openapi.json"></redoc>
<script src="https://unpkg.com/redoc@2/bundles/redoc.standalone.js"></script>
</body></html>`
