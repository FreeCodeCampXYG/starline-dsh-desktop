package dshversion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

type parsedVersion struct {
	major      int
	minor      int
	patch      int
	prerelease []string
}

// Normalize 校验 npm DSH 版本，并返回去除首尾空白后的 SemVer。
func Normalize(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if _, err := parse(value); err != nil {
		return "", err
	}
	return value, nil
}

// Compare 按 SemVer 比较两个 DSH 版本；左侧较新返回 1，相同返回 0，较旧返回 -1。
func Compare(left, right string) (int, error) {
	leftVersion, err := parse(strings.TrimSpace(left))
	if err != nil {
		return 0, err
	}
	rightVersion, err := parse(strings.TrimSpace(right))
	if err != nil {
		return 0, err
	}
	for _, pair := range [][2]int{{leftVersion.major, rightVersion.major}, {leftVersion.minor, rightVersion.minor}, {leftVersion.patch, rightVersion.patch}} {
		if pair[0] > pair[1] {
			return 1, nil
		}
		if pair[0] < pair[1] {
			return -1, nil
		}
	}
	return comparePrerelease(leftVersion.prerelease, rightVersion.prerelease), nil
}

func parse(value string) (parsedVersion, error) {
	match := versionPattern.FindStringSubmatch(value)
	if len(match) != 5 {
		return parsedVersion{}, fmt.Errorf("DSH 版本格式无效：%q", value)
	}
	major, majorErr := strconv.Atoi(match[1])
	minor, minorErr := strconv.Atoi(match[2])
	patch, patchErr := strconv.Atoi(match[3])
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return parsedVersion{}, fmt.Errorf("DSH 版本数字超出范围：%q", value)
	}
	var prerelease []string
	if match[4] != "" {
		prerelease = strings.Split(match[4], ".")
		for _, identifier := range prerelease {
			if isNumeric(identifier) && len(identifier) > 1 && identifier[0] == '0' {
				return parsedVersion{}, fmt.Errorf("DSH 预发布版本包含非法前导零：%q", value)
			}
		}
	}
	return parsedVersion{major: major, minor: minor, patch: patch, prerelease: prerelease}, nil
}

func comparePrerelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}
	limit := min(len(left), len(right))
	for index := 0; index < limit; index++ {
		leftIdentifier := left[index]
		rightIdentifier := right[index]
		leftNumeric := isNumeric(leftIdentifier)
		rightNumeric := isNumeric(rightIdentifier)
		switch {
		case leftNumeric && rightNumeric:
			if len(leftIdentifier) > len(rightIdentifier) {
				return 1
			}
			if len(leftIdentifier) < len(rightIdentifier) {
				return -1
			}
			if leftIdentifier > rightIdentifier {
				return 1
			}
			if leftIdentifier < rightIdentifier {
				return -1
			}
		case leftNumeric:
			return -1
		case rightNumeric:
			return 1
		case leftIdentifier > rightIdentifier:
			return 1
		case leftIdentifier < rightIdentifier:
			return -1
		}
	}
	if len(left) > len(right) {
		return 1
	}
	if len(left) < len(right) {
		return -1
	}
	return 0
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
