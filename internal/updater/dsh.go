package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"starline-dsh-desktop/internal/config"
	"starline-dsh-desktop/internal/dshversion"
)

const (
	dshPackageName    = "@deepseek-ai/dsh"
	registryLatestURL = "https://registry.npmjs.org/@deepseek-ai%2fdsh/latest"
	maxMetadataBytes  = 1 << 20
)

type DSHRelease struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	CurrentNewer    bool   `json:"currentNewer"`
}

type registryMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CheckDSHLatest 仅在用户主动请求时查询 npm 官方 latest，不修改任何本地状态。
func CheckDSHLatest(ctx context.Context, currentVersion string, settings config.Settings) (DSHRelease, error) {
	client, err := registryClient(settings)
	if err != nil {
		return DSHRelease{}, err
	}
	return checkDSHLatest(ctx, client, registryLatestURL, currentVersion)
}

func checkDSHLatest(ctx context.Context, client *http.Client, endpoint, currentVersion string) (DSHRelease, error) {
	currentVersion, err := dshversion.Normalize(currentVersion)
	if err != nil {
		return DSHRelease{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法创建 DSH 更新请求：%w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Starline-DSH-Desktop")
	response, err := client.Do(request)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法连接 npm 官方仓库：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return DSHRelease{}, fmt.Errorf("npm 官方仓库返回 HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMetadataBytes+1))
	if err != nil {
		return DSHRelease{}, fmt.Errorf("无法读取 DSH 版本信息：%w", err)
	}
	if len(body) > maxMetadataBytes {
		return DSHRelease{}, errors.New("npm DSH 版本信息超过安全大小限制")
	}
	var metadata registryMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return DSHRelease{}, fmt.Errorf("npm DSH 版本信息格式无效：%w", err)
	}
	if metadata.Name != dshPackageName {
		return DSHRelease{}, fmt.Errorf("npm 返回了意外的软件包：%s", metadata.Name)
	}
	latestVersion, err := dshversion.Normalize(metadata.Version)
	if err != nil {
		return DSHRelease{}, fmt.Errorf("npm 返回了无效的 DSH 版本：%w", err)
	}
	comparison, err := dshversion.Compare(latestVersion, currentVersion)
	if err != nil {
		return DSHRelease{}, err
	}
	return DSHRelease{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: comparison > 0,
		CurrentNewer:    comparison < 0,
	}, nil
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
		Timeout:   15 * time.Second,
		Transport: transport,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("npm DSH 版本检查重定向次数过多")
			}
			if request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Hostname(), "registry.npmjs.org") {
				return errors.New("npm DSH 版本检查拒绝了非官方重定向")
			}
			return nil
		},
	}, nil
}
