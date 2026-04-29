package contents

import (
	"path"
	"strings"

	contentsrouting "github.com/jo7oem/hatsukari/store/hosting/contents/routing"
)

func requestURLToRelPath(mountPath, urlPath string) (string, bool) {
	return contentsrouting.RequestURLToRelPath(mountPath, urlPath)
}

func isHiddenOrUnsafeRelPath(relPath string) bool {
	return contentsrouting.IsHiddenOrUnsafeRelPath(relPath)
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
