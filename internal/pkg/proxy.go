package pkg

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

func BuildHTTPClient(proxyStr string) (*http.Client, error) {
	if proxyStr == "" {
		return &http.Client{}, nil
	}

	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
			Timeout: 30 * time.Second,
		}, nil

	case "socks5", "socks5h":
		dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: 30 * time.Second,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}
