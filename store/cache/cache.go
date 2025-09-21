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
