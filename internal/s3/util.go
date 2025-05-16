package s3

import (
	"regexp"
	"strings"
)

var (
	bucketNameRegex      = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	ipAddressRegex       = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	disallowedPrefixes   = []string{"xn--"}
	disallowedSuffixes   = []string{"-s3alias", "--ol-s3"}
	disallowedSubstrings = []string{"..", ".-", "-."}
)

func IsValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if !bucketNameRegex.MatchString(name) {
		return false
	}
	if ipAddressRegex.MatchString(name) {
		return false
	}
	for _, prefix := range disallowedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	for _, suffix := range disallowedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	for _, sub := range disallowedSubstrings {
		if strings.Contains(name, sub) {
			return false
		}
	}
	return true
}
