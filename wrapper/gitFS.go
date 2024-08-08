package wrapper

import (
	"github.com/go-git/go-billy/v5"
	"io/fs"
	"time"
)

type GitFS struct {
	fs billy.Filesystem
}

func (g GitFS) Open(name string) (fs.File, error) {
	f, err := g.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return newWrapWile(f), nil

}

func NewGitFS(fs billy.Filesystem) GitFS {
	return GitFS{
		fs,
	}
}

type wrapFile struct {
	f billy.File
}

func (w wrapFile) Stat() (fs.FileInfo, error) {
	var b []byte

	dir := false

	size, err := w.f.Read(b)
	if err != nil {
		dir = true
	}

	return dummyFileInfo{
		name:  w.f.Name(),
		size:  int64(size),
		isDir: dir,
		t:     time.Now(),
	}, nil
}

func (w wrapFile) Read(bytes []byte) (int, error) {
	return w.f.Read(bytes)
}

func (w wrapFile) Close() error {
	return w.f.Close()
}

func newWrapWile(f billy.File) wrapFile {
	return wrapFile{
		f,
	}
}

type dummyFileInfo struct {
	name  string
	size  int64
	isDir bool
	t     time.Time
}

func (d dummyFileInfo) Name() string {
	return d.name
}

func (d dummyFileInfo) Size() int64 {
	return d.size
}

func (d dummyFileInfo) Mode() fs.FileMode {
	if d.isDir {
		return 0o777 + 0x8000
	}

	return 0o0777
}

func (d dummyFileInfo) ModTime() time.Time {
	return d.t
}

func (d dummyFileInfo) IsDir() bool {
	return d.isDir
}

func (d dummyFileInfo) Sys() any {
	return ""
}
