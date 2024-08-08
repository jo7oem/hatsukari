package builder

import (
	"errors"
	"github.com/jo7oem/hatsukari/store/config"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
)

func NewHandler(conf *config.ContentConfig) (http.Handler, error) {
	if conf == nil {
		return nil, errors.New("invalid arguments. conf is nil")
	}

	blg, err := loadTopConf(conf.GetFS())
	if err != nil {
		return nil, err
	}

	return blg.NewHandler(), nil
}

type HatsukariConf struct {
	BlogName string             `yaml:"blogName"`
	Contents map[string]content `yaml:"contents"`
}

func loadTopConf(fSys fs.FS) (*HatsukariConf, error) {
	if fSys == nil {
		return nil, errors.New("nil")
	}

	file, err := fSys.Open("hatsukari.yaml")
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var res HatsukariConf
	if err := yaml.Unmarshal(buf, &res); err != nil {
		return nil, err
	}

	for k, v := range res.Contents {
		v.name = k
		res.Contents[k] = v
	}

	return &res, nil
}

func (h HatsukariConf) NewHandler() http.Handler {
	handler := http.NewServeMux()
	for _, v := range h.Contents {
		hl, err := v.getHandler()
		if err != nil {
			continue
		}

		handler.Handle(v.URL, http.StripPrefix(v.URL, hl))
	}

	return handler
}

const (
	TypeStaticFiles = "staticFiles"
	TypeStaticDir   = "staticDir"
	TypeTemplate    = "template"
)

type content struct {
	name     string
	FilePath string `yaml:"filePath"`
	URL      string `yaml:"url"`
	Type     string `yaml:"type"`
	confName string `yaml:"confName"`
}

func (c content) getHandler() (http.Handler, error) {
	switch c.Type {
	case TypeStaticFiles:
		return http.FileServer(http.Dir(c.FilePath)), nil

	}

	return nil, errors.New("invalid type")
}
