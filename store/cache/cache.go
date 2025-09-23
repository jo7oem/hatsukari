package cache

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type CacheManager struct {
	rootPath string
	*os.Root

	doCleanup bool
	closer    func() error
}

func (c *CacheManager) Open(name string) (fs.File, error) {
	return c.Root.Open(name)
}
func (c *CacheManager) Close() error {
	return c.closer()
}
func (c *CacheManager) close() error {
	if err := c.Root.Close(); err != nil {
		return err
	}

	if !c.doCleanup {
		return nil
	}

	return os.RemoveAll(c.rootPath)
}
func (c *CacheManager) Store(path string, content []byte) error {
	// ファイルはc.Rootを使って操作します。
	// まず格納先のディレクトリが存在するか確認し、存在しない場合は作成します。
	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)
	if err := c.MkdirAll(dir); err != nil {
		return err
	}

	file, err := c.Root.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	return err
}

// MkdirAll は、path に指定されたディレクトリとその親ディレクトリをすべて作成します。
// すでに存在するディレクトリがあってもエラーにはなりません。
// perm は新しいディレクトリのパーミッションを指定します。
// MkdirAll は、path が空文字列の場合や、"." の場合は何もせずに nil を返します。
// また、path の最終コンポーネントが "." の場合も同様です。
// MkdirAll は、path の親ディレクトリが存在しない場合は再帰的に作成します。
// 例えば、MkdirAll("a/b/c") は "a"、"a/b"、"a/b/c" の各ディレクトリを作成します。
// MkdirAll は、途中のディレクトリがファイルであった場合や、その他のエラーが発生した場合はそのエラーを返します。
func (c *CacheManager) MkdirAll(path string) error {
	if path == "" || path == "." {
		return nil
	}

	// pathを分割して、c.Rootから順にディレクトリを作成
	parts := filepath.SplitList(filepath.Clean(path))
	curr := c.Root
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		// ディレクトリが存在するか確認
		info, err := curr.Stat(part)
		if err == nil {
			if !info.IsDir() {
				return &os.PathError{Op: "mkdir", Path: part, Err: os.ErrExist}
			}
			// 既存ディレクトリなら次へ
			curr, err = curr.OpenRoot(part)
			if err != nil {
				return err
			}
		}
		// 存在しない場合は作成
		if err := curr.Mkdir(part, 0700); err != nil {
			return err
		}
	}
	return nil
}

func NewCacheManager(rootPath string) (*CacheManager, error) {
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(rootPath, 0700); err != nil {
		return nil, err
	}

	rootfs, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}

	cm := &CacheManager{
		rootPath:  rootPath,
		Root:      rootfs,
		doCleanup: false,
	}

	cm.closer = sync.OnceValue(cm.close)

	return cm, nil
}
