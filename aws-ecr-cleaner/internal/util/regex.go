package util

import (
	"log"
	"regexp"
	"strings"
)

// TrimRegistry 去掉仓库 URI 中的 registry 前缀
func TrimRegistry(uri string) string {
	parts := strings.SplitN(uri, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return uri
}

// MultiRegexMatch 判断字符串 s 是否匹配配置字符串 config，支持 "OR" 和 "&&" 分隔逻辑
func MultiRegexMatch(s, config string) bool {
	if strings.Contains(config, "OR") {
		parts := strings.Split(config, "OR")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			r, err := regexp.Compile(part)
			if err != nil {
				log.Fatalf("Invalid regex part '%s': %v", part, err)
			}
			if r.MatchString(s) {
				return true
			}
		}
		return false
	} else if strings.Contains(config, "&&") {
		parts := strings.Split(config, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			r, err := regexp.Compile(part)
			if err != nil {
				log.Fatalf("Invalid regex part '%s': %v", part, err)
			}
			if !r.MatchString(s) {
				return false
			}
		}
		return true
	} else {
		r, err := regexp.Compile(config)
		if err != nil {
			log.Fatalf("Invalid regex config '%s': %v", config, err)
		}
		return r.MatchString(s)
	}
}

// HoldTagMatch 判断合并后的标签是否匹配 holdTagRegex
func HoldTagMatch(combinedTags, holdTagRegex string) bool {
	return MultiRegexMatch(combinedTags, holdTagRegex)
}

// CompositeMatch 用于 TARGET_REPO_REGEX 的匹配，默认采用 AND 逻辑
func CompositeMatch(s, compositeRegex string) bool {
	return MultiRegexMatch(s, compositeRegex)
}
