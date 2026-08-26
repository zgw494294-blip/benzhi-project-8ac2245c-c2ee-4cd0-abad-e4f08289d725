package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

type config struct {
	addr      string
	dataDir   string
	selfcheck bool
}

func parseConfig() (config, error) {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", "", "HTTP 监听地址，例如 127.0.0.1:19081")
	flag.StringVar(&cfg.dataDir, "data", "data", "事件账本数据目录")
	flag.BoolVar(&cfg.selfcheck, "selfcheck", false, "运行真实 HTTP 全流程自检后退出")
	flag.Parse()
	if cfg.addr == "" {
		if portText := os.Getenv("PORT"); portText != "" {
			port, err := strconv.Atoi(portText)
			if err != nil || port < 1 || port > 65535 {
				return cfg, errors.New("PORT 必须是 1 到 65535 的端口号")
			}
			cfg.addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		} else {
			cfg.addr = "127.0.0.1:19081"
		}
	}
	host, port, err := net.SplitHostPort(cfg.addr)
	if err != nil {
		return cfg, fmt.Errorf("-addr 无效: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return cfg, errors.New("-addr 必须使用明确的回环 IP 地址")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return cfg, errors.New("-addr 端口无效")
	}
	return cfg, nil
}
