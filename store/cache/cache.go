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
	if err := c.MkdirAll(dir, 0700); err != nil {
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
