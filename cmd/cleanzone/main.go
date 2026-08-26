package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cleanroom-release-go/internal/httpapi"
	"cleanroom-release-go/internal/ledger"
	"cleanroom-release-go/internal/workflow"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cleanzone:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	dataDir := cfg.dataDir
	if cfg.selfcheck {
		temp, err := os.MkdirTemp("", "cleanzone-selfcheck-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(temp)
		dataDir = temp
	}
	store, err := ledger.Open(dataDir)
	if err != nil {
		return fmt.Errorf("打开账本: %w", err)
	}
	service := workflow.New(store, cfg.signingSecret)
	api := httpapi.New(service, slog.Default())
	listener, err := net.Listen("tcp", cfg.address)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.address, err)
	}
	server := &http.Server{Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()
	slog.Info("洁净室放行服务已启动", "address", cfg.address, "data", dataDir)
	if cfg.selfcheck {
		return runBoundedSelfcheck(server, serveErr, listener.Addr().String())
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("HTTP 服务异常: %w", err)
		}
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("服务排空失败: %w", err)
	}
	return nil
}

func runBoundedSelfcheck(server *http.Server, serveErr <-chan error, address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	err := selfcheck(ctx, "http://"+address)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	if err != nil {
		return err
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	if serveResult, ok := <-serveErr; ok && serveResult != nil {
		return serveResult
	}
	fmt.Println("selfcheck 通过：偏差返修、冻结放行与凭据核验流程完成")
	return nil
}
