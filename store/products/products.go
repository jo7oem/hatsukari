package products

import (
	"errors"
	"os"
)

type Products struct {
	root *os.Root
}

func New(targetPath string) (*Products, error) {
	root, err := os.OpenRoot(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(targetPath, 0700); err != nil {
			return nil, err
		}

		root, err = os.OpenRoot(targetPath)
	}
	if err != nil {
		return nil, err
	}
	return &Products{
		root: root,
	}, nil
}

func (p *Products) Close() error {
	if p.root != nil {
		return p.root.Close()
	}

	return nil
}
