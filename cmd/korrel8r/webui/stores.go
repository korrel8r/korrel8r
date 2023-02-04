package webui

import (
	"context"
	"net/http"
	"path"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// storeHandler serves JSON results from store GET calls.
type storeHandler struct{ ui *WebUI }

func (h *storeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	domain, err := h.ui.Engine.DomainErr(path.Base(req.URL.Path))
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	store, err := h.ui.Engine.StoreErr(
		domain.String())
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	query, err := domain.UnmarshalQuery([]byte(params.Get("query")))
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	result := korrel8r.NewResult(query.Class())
	err = store.Get(context.Background(), query, result)
	data := map[string]any{
		"query":  query,
		"err":    err,
		"result": result.List(),
	}
	serveTemplate(w, h.ui.Page("stores"), storeHTML, data)
}

const storeHTML = `
{{define "body"}}
    Query: {{json .query}}<br>
    <hr>
    {{if .err}}
        Error: {{.err}}<br>
    {{else}}
        Results ({{len .result}})<br>
            {{range .result}}<hr><pre>{{yaml .}}</pre>{{end}}
        </pre>
    {{end}}
{{end}}
    `
