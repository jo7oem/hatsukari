package routing

import (
	"path"
	"strings"
)

func RequestURLToRelPath(mountPath, urlPath string) (string, bool) {
	cleanMount := path.Clean("/" + strings.TrimPrefix(mountPath, "/"))
	if cleanMount != "/" && strings.HasSuffix(mountPath, "/") {
		cleanMount += "/"
	}

	cleanURL := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))

	if cleanMount == "/" {
		if cleanURL == "/" {
			return "index", true
		}
		return strings.TrimPrefix(cleanURL, "/"), true
	}

	if cleanURL == strings.TrimSuffix(cleanMount, "/") {
		return "index", true
	}
	if !strings.HasPrefix(cleanURL, cleanMount) {
		return "", false
	}

	rel := strings.TrimPrefix(cleanURL, cleanMount)
	if rel == "" {
		return "index", true
	}
	return rel, true
}

func IsHiddenOrUnsafeRelPath(relPath string) bool {
	if relPath == "" || relPath == "." {
		return true
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") {
		return true
	}
	for seg := range strings.SplitSeq(relPath, "/") {
		if seg == "" {
			continue
		}
		if seg[0] == '.' {
			return true
		}
	}
	return false
}
