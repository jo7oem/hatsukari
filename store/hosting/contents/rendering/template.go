package rendering

import (
	"path"
	"strings"
)

func ResolveContentTemplateName(configured string) string {
	templateName := strings.TrimSpace(configured)
	if templateName == "" {
		return "template.html"
	}
	return templateName
}

func ResolveSiteTemplateCandidates(siteTemplate string, resolvedPath string) []string {
	ext := path.Ext(resolvedPath)
	if ext == "" {
		return nil
	}
	if strings.TrimSpace(siteTemplate) != "" {
		return []string{strings.TrimSpace(siteTemplate)}
	}
	if ext == ".html" {
		return []string{"site-template.html"}
	}
	return []string{"site-template" + ext, "site-template.html"}
}
