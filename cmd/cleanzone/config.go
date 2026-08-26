package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	address       string
	dataDir       string
	selfcheck     bool
	signingSecret string
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("cleanzone", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.address, "addr", "127.0.0.1:19081", "HTTP 监听地址")
	fs.StringVar(&cfg.dataDir, "data", "./data/ledger", "本地账本目录")
	fs.BoolVar(&cfg.selfcheck, "selfcheck", false, "运行真实 HTTP 流程自检并退出")
	fs.StringVar(&cfg.signingSecret, "signing-secret", "local-cleanroom-release-signing-key", "凭据签名密钥")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() != 0 {
		return cfg, fmt.Errorf("存在无法识别的位置参数: %s", strings.Join(fs.Args(), " "))
	}
	addrExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			addrExplicit = true
		}
	})
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" && !addrExplicit {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return cfg, fmt.Errorf("PORT 必须是 1 到 65535 的端口号")
		}
		cfg.address = net.JoinHostPort("127.0.0.1", strconv.Itoa(value))
	}
	if err := validateAddress(cfg.address); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.dataDir) == "" {
		return cfg, fmt.Errorf("data 目录不能为空")
	}
	if strings.TrimSpace(cfg.signingSecret) == "" {
		return cfg, fmt.Errorf("signing-secret 不能为空")
	}
	return cfg, nil
}

func validateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 必须采用 host:port 格式: %w", err)
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("addr 端口必须在 1 到 65535 之间")
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("addr 不得省略主机")
	}
	ip := net.ParseIP(host)
	if host == "0.0.0.0" || host == "::" || (ip != nil && ip.IsUnspecified()) {
		return fmt.Errorf("addr 不得监听全部网络接口")
	}
	return nil
}
