package materials

import "os"

type Materials struct {
	root *os.Root
}

func New(targetPath string) (*Materials, error) {
	root, err := os.OpenRoot(targetPath)
	if err != nil {
		return nil, err
	}
	return &Materials{
		root: root,
	}, nil
}

func (m *Materials) Close() error {
	if m.root != nil {
		return m.root.Close()
	}

	return nil
}
