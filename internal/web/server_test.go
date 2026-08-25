package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/checks"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/store"
)

func testRoutes(t *testing.T) http.Handler {
	t.Helper()
	repository, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	service := application.NewService(repository, checks.New(checks.DefaultConfig()))
	return NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes()
}

func TestWorkbenchServesCompleteHTML(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/workbench", nil)
	response := httptest.NewRecorder()
	testRoutes(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	body := response.Body.String()
	for _, required := range []string{"<!doctype html>", "<body>", "舞台吊挂安全启用工作台", "/assets/app.js"} {
		if !strings.Contains(body, required) {
			t.Fatalf("工作台缺少 %q", required)
		}
	}
}

func TestCreatePlanReportsFieldErrors(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/rigging-plans", strings.NewReader(`{"title":""}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testRoutes(t).ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "VALIDATION_ERROR") {
		t.Fatalf("未返回结构化字段错误：%s", response.Body.String())
	}
}
