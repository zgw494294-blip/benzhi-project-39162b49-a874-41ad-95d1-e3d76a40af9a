package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/application"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/checks"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/store"
	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	configuration, err := parseConfig(os.Args[1:])
	if err != nil {
		logger.Error("配置无效", "error", err)
		os.Exit(2)
	}
	if configuration.selfCheck {
		ctx, cancel := context.WithTimeout(context.Background(), configuration.selfCheckTimeout)
		defer cancel()
		if err := runSelfCheck(ctx, configuration, logger); err != nil {
			logger.Error("selfcheck 失败", "error", err)
			os.Exit(1)
		}
		logger.Info("selfcheck 完成", "result", "完整业务流程及授权验证通过")
		return
	}
	if err := runServer(configuration, logger); err != nil {
		logger.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func buildHandler(ctx context.Context, database string, logger *slog.Logger) (*store.SQLiteRepository, http.Handler, error) {
	repository, err := store.Open(ctx, database)
	if err != nil {
		return nil, nil, err
	}
	checker := checks.New(checks.DefaultConfig())
	service := application.NewService(repository, checker)
	handler := web.NewHandler(service, logger)
	return repository, handler.Routes(), nil
}

func runServer(configuration config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repository, handler, err := buildHandler(ctx, configuration.database, logger)
	if err != nil {
		return err
	}
	defer repository.Close()
	server := &http.Server{
		Addr: configuration.address, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("舞台吊挂安全启用工作台已启动", "addr", configuration.address, "workbench", "http://"+configuration.address+"/workbench")
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("优雅关闭：%w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
