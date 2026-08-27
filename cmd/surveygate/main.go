package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subsurface-survey-gate/internal/application"
	"subsurface-survey-gate/internal/eventstore"
	"subsurface-survey-gate/internal/httpapi"
	"subsurface-survey-gate/internal/quality"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg, err := parseConfig()
	if err != nil {
		logger.Error("配置无效", "error", err)
		os.Exit(2)
	}
	if cfg.selfcheck {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := runSelfcheck(ctx, cfg, logger); err != nil {
			logger.Error("自检失败", "error", err)
			os.Exit(1)
		}
		logger.Info("自检通过")
		return
	}
	store, err := eventstore.Open(cfg.dataDir)
	if err != nil {
		logger.Error("打开事件账本失败", "error", err)
		os.Exit(1)
	}
	service := application.NewService(store, quality.NewScanner())
	server := newHTTPServer(cfg.addr, httpapi.New(service, logger))
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		logger.Error("监听失败", "error", err)
		os.Exit(1)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	logger.Info("服务已启动", "addr", listener.Addr().String(), "dataDir", cfg.dataDir)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-signals:
		logger.Info("收到停止信号", "signal", sig.String())
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务异常", "error", err)
			os.Exit(1)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("优雅停机失败", "error", err)
		os.Exit(1)
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 8 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10}
}
