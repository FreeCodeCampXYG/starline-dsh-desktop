package launcher

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func safeLoopbackURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// availableLoopbackPort 让操作系统选择高位端口；关闭监听后立即交给 DSH 使用。
func availableLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.Port == 0 {
		return 0, errors.New("操作系统没有返回有效端口")
	}
	return address.Port, nil
}
