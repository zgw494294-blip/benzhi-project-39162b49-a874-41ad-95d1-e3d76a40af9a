package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultAddress = "127.0.0.1:19081"

type config struct {
	address          string
	database         string
	selfCheck        bool
	selfCheckTimeout time.Duration
}

func parseConfig(arguments []string) (config, error) {
	set := flag.NewFlagSet("rigging-workbench", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	address := set.String("addr", "", "HTTP 监听地址")
	database := set.String("db", "rigging-workbench.db", "SQLite 数据库路径")
	selfCheck := set.Bool("selfcheck", false, "运行完整 HTTP 自检后退出")
	selfCheckTimeout := set.Duration("selfcheck-timeout", 20*time.Second, "自检超时")
	if err := set.Parse(arguments); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, fmt.Errorf("不支持位置参数：%s", strings.Join(set.Args(), " "))
	}
	resolvedAddress := strings.TrimSpace(*address)
	if resolvedAddress == "" {
		resolvedAddress = addressFromPort(os.Getenv("PORT"))
	}
	if resolvedAddress == "" {
		resolvedAddress = defaultAddress
	}
	if err := validateAddress(resolvedAddress); err != nil {
		return config{}, err
	}
	if *selfCheckTimeout <= 0 {
		return config{}, fmt.Errorf("selfcheck-timeout 必须大于 0")
	}
	return config{
		address: resolvedAddress, database: *database,
		selfCheck: *selfCheck, selfCheckTimeout: *selfCheckTimeout,
	}, nil
}

func addressFromPort(value string) string {
	trimmed := strings.TrimSpace(value)
	port, err := strconv.Atoi(trimmed)
	if err != nil || port < 1 || port > 65535 {
		return ""
	}
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("addr 必须使用 host:port 格式：%w", err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("addr 必须明确指定监听主机")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("addr 端口必须在 1 到 65535 之间")
	}
	return nil
}
