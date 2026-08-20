package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"starline-dsh-desktop/internal/config"
	"starline-dsh-desktop/internal/dshversion"
)

const (
	registryDistTagsURL       = "https://registry.npmjs.org/-/package/@deepseek-ai%2Fdsh/dist-tags"
	registryMirrorDistTagsURL = "https://registry.npmmirror.com/-/package/@deepseek-ai%2Fdsh/dist-tags"
	maxMetadataBytes          = 64 << 10
	registryRequestTimeout    = 8 * time.Second
)

type DSHRelease struct {
	CurrentVersion         string `json:"currentVersion"`
	LatestVersion          string `json:"latestVersion"`
	NextVersion            string `json:"nextVersion,omitempty"`
	LatestUpdateAvailable  bool   `json:"latestUpdateAvailable"`
	NextUpdateAvailable    bool   `json:"nextUpdateAvailable"`
	CurrentNewerThanLatest bool   `json:"currentNewerThanLatest"`
}

type registryDistTags struct {
	Latest string `json:"latest"`
	Next   string `json:"next"`
}

type registryEndpoint struct {
	url      string
	settings config.Settings
}

// CheckDSHChannelsAutomatic 自动检查固定优先直连国内镜像，不依赖本机代理是否启动。
func CheckDSHChannelsAutomatic(ctx context.Context, currentVersion string) (DSHRelease, error) {
	return checkDSHChannelEndpoints(ctx, currentVersion, []registryEndpoint{
		{registryMirrorDistTagsURL, config.Settings{ProxyMode: config.ProxyModeDisabled}},
		{registryDistTagsURL, config.Settings{ProxyMode: config.ProxyModeDisabled}},
	})
}

// CheckDSHChannels 供用户手动刷新和应用更新；优先国内镜像，并按当前设置决定是否使用代理。
func CheckDSHChannels(ctx context.Context, currentVersion string, settings config.Settings) (DSHRelease, error) {
	normalized, err := config.Normalize(settings)
	if err != nil {
		return DSHRelease{}, err
	}
	return checkDSHChannelEndpoints(ctx, currentVersion, []registryEndpoint{
		{registryMirrorDistTagsURL, normalized},
		{registryDistTagsURL, normalized},
	})
}

func checkDSHChannelEndpoints(ctx context.Context, currentVersion string, endpoints []registryEndpoint) (DSHRelease, error) {
	var lastErr error
	for _, endpoint := range endpoints {
		client, clientErr := registryClient(endpoint.settings)
		if clientErr != nil {
			lastErr = clientErr
			continue
		}
		release, checkErr := checkDSHChannels(ctx, client, endpoint.url, currentVersion)
		client.CloseIdleConnections()
		if checkErr == nil {
			return release, nil
		}
		lastErr = checkErr
	}
	if lastErr == nil {
		lastErr = errors.New("没有可用的 npm registry")
	}
	return DSHRelease{}, lastErr
}

func checkDSHChannels(ctx context.Context, client *http.Client, endpoint, currentVersion string) (DSHRelease, error) {
	currentVersion, err := dshversion.Normalize(currentVersion)
	if err != nil {
		return DSHRelease{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法创建 DSH 更新请求：%w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("User-Agent", "Starline-DSH-Desktop")
	response, err := client.Do(request)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法连接 npm registry：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return DSHRelease{}, fmt.Errorf("npm registry 返回 HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMetadataBytes+1))
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法读取 DSH 版本信息：%w", err)
	}
	if len(body) > maxMetadataBytes {
		return DSHRelease{}, errors.New("npm DSH 版本信息超过安全大小限制")
	}
	var tags registryDistTags
	if err := json.Unmarshal(body, &tags); err != nil {
		return DSHRelease{}, fmt.Errorf("npm DSH 版本信息格式无效：%w", err)
	}
	latestVersion, err := dshversion.Normalize(tags.Latest)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("npm latest 返回了无效的 DSH 版本：%w", err)
	}
	latestComparison, err := dshversion.Compare(latestVersion, currentVersion)
	if err != nil {
		return DSHRelease{}, err
	}
	release := DSHRelease{
		CurrentVersion:         currentVersion,
		LatestVersion:          latestVersion,
		LatestUpdateAvailable:  latestComparison > 0,
		CurrentNewerThanLatest: latestComparison < 0,
	}
	if strings.TrimSpace(tags.Next) == "" {
		return release, nil
	}
	nextVersion, err := dshversion.Normalize(tags.Next)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("npm next 返回了无效的 DSH 版本：%w", err)
	}
	nextComparison, err := dshversion.Compare(nextVersion, currentVersion)
	if err != nil {
		return DSHRelease{}, err
	}
	release.NextVersion = nextVersion
	release.NextUpdateAvailable = nextComparison > 0
	return release, nil
}

func registryClient(settings config.Settings) (*http.Client, error) {
	normalized, err := config.Normalize(settings)
	if err != nil {
		return nil, err
	}
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("系统 HTTP 传输器类型不受支持")
	}
	transport := defaultTransport.Clone()
	transport.DialContext = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = registryRequestTimeout
	transport.ExpectContinueTimeout = 1 * time.Second
	switch normalized.ProxyMode {
	case config.ProxyModeInherit:
		transport.Proxy = http.ProxyFromEnvironment
	case config.ProxyModeCustom:
		proxyURL, parseErr := url.Parse(normalized.ProxyURL)
		if parseErr != nil {
			return nil, fmt.Errorf("代理地址格式无效：%w", parseErr)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	case config.ProxyModeDisabled:
		transport.Proxy = nil
	default:
		return nil, fmt.Errorf("未知代理模式：%s", normalized.ProxyMode)
	}
	return &http.Client{
		Timeout:   registryRequestTimeout,
		Transport: transport,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("npm DSH 版本检查重定向次数过多")
			}
			host := strings.ToLower(request.URL.Hostname())
			if request.URL.Scheme != "https" || (host != "registry.npmjs.org" && host != "registry.npmmirror.com") {
				return errors.New("npm DSH 版本检查拒绝了非受信 registry 重定向")
			}
			return nil
		},
	}, nil
}
