package webui

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type storeHandler struct{ ui *WebUI }

func (h *storeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()
	domain, err := h.ui.Engine.DomainErr("domain")
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	store, err := h.ui.Engine.StoreErr(
		domain.String())
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	query := domain.Query(nil)
	if httpError(w, json.Unmarshal([]byte(params.Get("query")), query), http.StatusNotFound) {
		return
	}
	result := korrel8r.NewResult(query.Class())
	err = store.Get(context.Background(), query, result)
	data := map[string]any{
		"query":  query,
		"err":    err,
		"result": result.List(),
	}
	t, err := h.ui.Page("stores").Parse(`
{{define "body"}}
    Query: {{.query}}<br>
    <hr>
    {{if .err}}
        Error: {{.err}}<br>
    {{else}}
        Results ({{len .result}})<br>
            {{range .result}}<hr><pre>{{yaml .}}</pre>{{end}}
        </pre>
    {{end}}
{{end}}
    `)
	if !httpError(w, err, http.StatusInternalServerError) {
		httpError(w, t.Execute(w, data), http.StatusInternalServerError)
	}
}
