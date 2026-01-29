package site

import (
	"io"
	"net/http"
	"os"

	"github.com/goccy/go-yaml"
)

type ContentConfig struct {
	RegisterIndexing bool   `yaml:"registerIndexing,omitempty"`
	templatesDir     string `yaml:"templatesDir,omitempty"`
}

func openContentConfig(fs *os.Root) (*ContentConfig, error) {
	const confNameYaml = ".content.yaml"
	const confNameYml = ".content.yml"

	f, err := fs.Open(confNameYaml)
	if os.IsNotExist(err) {
		f, err = fs.Open(confNameYml)
	}

	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var config ContentConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func OpenContentDir(fs *os.Root, path string) (*Content, error) {
	root, err := fs.OpenRoot(path)
	if err != nil {
		return nil, err
	}

	conf, err := openContentConfig(root)
	switch {
	case os.IsNotExist(err):
		conf = &ContentConfig{}
	case err == nil:
		// ok
	default:
		return nil, err
	}

	return &Content{config: *conf,
		root: root}, nil
}

type Content struct {
	config ContentConfig
	root   *os.Root
}

func (c Content) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//TODO implement me
	panic("implement me")
}
