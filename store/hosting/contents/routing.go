package contents

import (
	"path"
	"strings"
)

func requestURLToRelPath(mountPath, urlPath string) (string, bool) {
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

func isHiddenOrUnsafeRelPath(relPath string) bool {
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

func (c *Content) resolveContentPathByPriority(relPath string) (string, bool) {
	if path.Ext(relPath) != "" {
		return relPath, true
	}

	candidate, ok := c.resolveNoExtRelPath(relPath)
	if !ok {
		return "", false
	}

	// http.FileServerFS() のディレクトリハンドリング挙動に合わせるため、index.html を URL 末尾の '/' として扱う。
	candidate, _ = strings.CutSuffix(candidate, "index.html")
	return candidate, true
}

func (c *Content) resolveNoExtRelPath(relPath string) (string, bool) {
	base := relPath
	for {
		resolved, isDir, ok := c.openFirstExistingCandidate(base)
		if !ok {
			return "", false
		}
		if !isDir {
			return resolved, true
		}
		base = path.Join(base, "index")
	}
}

func (c *Content) openFirstExistingCandidate(relPath string) (resolved string, isDir bool, ok bool) {
	for _, ext := range []string{"", ".html", ".md"} {
		p := relPath + ext
		f, err := c.root.Open(p)
		if err != nil {
			continue
		}

		if ext == "" {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && stat.IsDir() {
				return relPath, true, true
			}
			return relPath, false, true
		}

		_ = f.Close()
		return p, false, true
	}
	return "", false, false
}

func (c *Content) Path() string {
	return c.path()
}

func (c *Content) calcPath() string {
	if c.parent == nil {
		return "/"
	}
	return path.Join(c.parent.Path(), c.relPath) + "/"
}
