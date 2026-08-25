package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
)

type Handler struct {
	service *application.Service
	assets  fs.FS
	logger  *slog.Logger
}

func NewHandler(service *application.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: service, assets: embeddedAssets(), logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Home)
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /workbench", h.Workbench)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(h.assets)))
	mux.HandleFunc("GET /api/v1/rigging-plans", h.RiggingPlans)
	mux.HandleFunc("POST /api/v1/rigging-plans", h.RiggingPlans)
	mux.HandleFunc("GET /api/v1/rigging-plans/{planID}", h.RiggingPlan)
	mux.HandleFunc("POST /api/v1/rigging-plans/{planID}/revisions", h.PlanRevisions)
	mux.HandleFunc("POST /api/v1/rigging-plans/{planID}/checks", h.PlanChecks)
	mux.HandleFunc("POST /api/v1/rigging-plans/{planID}/rehearsals", h.PlanRehearsals)
	mux.HandleFunc("POST /api/v1/rigging-plans/{planID}/reviews", h.PlanReviews)
	mux.HandleFunc("GET /api/v1/authorizations/{code}", h.AuthorizationVerification)
	return h.recoverPanic(h.logRequest(mux))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (h *Handler) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		h.logger.Info("HTTP 请求", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(started))
	})
}

func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("HTTP 处理异常", "error", recovered)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
