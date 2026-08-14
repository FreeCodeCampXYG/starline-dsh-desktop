package launcher

import "strings"

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
	return mergeNoProxy(environment, []string{"127.0.0.1", "localhost", "::1"})
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
