package web

import (
	"net/http"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/workbench", http.StatusTemporaryRedirect)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Workbench(w http.ResponseWriter, r *http.Request) {
	data, err := fsReadFile(h.assets, "index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ASSET_ERROR", "工作台资源不可用", nil)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
