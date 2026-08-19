package launcher

import (
	"net/url"
	"strings"
)

const (
	npmMirrorRegistry       = "https://registry.npmmirror.com"
	npmFetchTimeout         = "10000"
	npmFetchRetries         = "1"
	npmFetchRetryMinTimeout = "1000"
	npmFetchRetryMaxTimeout = "3000"
)

func childEnvironment(environment []string, proxyMode, proxyURL string) []string {
	if proxyMode == "custom" || proxyMode == "disabled" {
		environment = withoutProxyVariables(environment)
	}
	if proxyMode == "custom" {
		environment = append(
			environment,
			"HTTP_PROXY="+proxyURL,
			"HTTPS_PROXY="+proxyURL,
			"http_proxy="+proxyURL,
			"https_proxy="+proxyURL,
			"npm_config_proxy="+proxyURL,
			"npm_config_https_proxy="+proxyURL,
		)
	}
	if shouldUseDomesticMirror(environment, proxyMode) {
		environment = append(environment, "npm_config_registry="+npmMirrorRegistry)
	}
	// 限制 npm 的单次网络等待和重试，避免代理端口失效时 npx 长时间无响应。
	environment = append(environment,
		"npm_config_fetch_timeout="+npmFetchTimeout,
		"npm_config_fetch_retries="+npmFetchRetries,
		"npm_config_fetch_retry_mintimeout="+npmFetchRetryMinTimeout,
		"npm_config_fetch_retry_maxtimeout="+npmFetchRetryMaxTimeout,
		"npm_config_update_notifier=false",
	)
	return mergeNoProxy(environment, []string{"127.0.0.1", "localhost", "::1"})
}

func shouldUseDomesticMirror(environment []string, proxyMode string) bool {
	if hasNPMRegistry(environment) {
		return false
	}
	if proxyMode == "disabled" {
		return true
	}
	if proxyMode != "inherit" {
		return false
	}
	return !hasProxy(environment)
}

func hasNPMRegistry(environment []string) bool {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "npm_config_registry") && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func hasProxy(environment []string) bool {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if !found || strings.TrimSpace(value) == "" {
			continue
		}
		switch strings.ToLower(key) {
		case "http_proxy", "https_proxy", "all_proxy", "npm_config_proxy", "npm_config_https_proxy":
			return true
		}
	}
	return false
}

func npmRegistry(environment []string) string {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "npm_config_registry") {
			if parsed, err := url.Parse(strings.TrimSpace(value)); err == nil && parsed.Scheme != "" && parsed.Hostname() != "" {
				return strings.TrimRight(parsed.String(), "/")
			}
		}
	}
	return "https://registry.npmjs.org"
}

func withoutProxyVariables(environment []string) []string {
	proxyKeys := map[string]bool{
		"http_proxy":             true,
		"https_proxy":            true,
		"all_proxy":              true,
		"npm_config_proxy":       true,
		"npm_config_https_proxy": true,
		"npm_config_noproxy":     true,
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && proxyKeys[strings.ToLower(key)] {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func mergeNoProxy(environment []string, required []string) []string {
	result := make([]string, 0, len(environment)+1)
	values := make([]string, 0, len(required)+4)
	seen := make(map[string]bool)
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "NO_PROXY") {
			for _, item := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" && !seen[strings.ToLower(trimmed)] {
					values = append(values, trimmed)
					seen[strings.ToLower(trimmed)] = true
				}
			}
			continue
		}
		result = append(result, entry)
	}
	for _, item := range required {
		if !seen[strings.ToLower(item)] {
			values = append(values, item)
			seen[strings.ToLower(item)] = true
		}
	}
	return append(result, "NO_PROXY="+strings.Join(values, ","))
}
