package tags

import "strings"

type Definition struct {
	DefaultLang string            `yaml:"defaultLang,omitempty"`
	Label       map[string]string `yaml:"label,omitempty"`
	About       map[string]string `yaml:"about,omitempty"`
}

func ParseRoute(relPath string) (handled bool, tagKey string, invalid bool) {
	if relPath != "tags" && !strings.HasPrefix(relPath, "tags/") {
		return false, "", false
	}

	tagKey = strings.TrimPrefix(relPath, "tags/")
	if relPath == "tags" || tagKey == "" {
		return true, "", false
	}
	if strings.Contains(tagKey, "/") {
		return true, "", true
	}
	return true, tagKey, false
}

func NormalizeDefinition(definition Definition) Definition {
	defaultLang := strings.TrimSpace(definition.DefaultLang)
	if defaultLang == "" {
		defaultLang = "ja"
	}
	return Definition{
		DefaultLang: defaultLang,
		Label:       normalizeLocalizedMap(definition.Label),
		About:       normalizeLocalizedMap(definition.About),
	}
}

func BuildLocalizedPublicValue(defaultLang string, values map[string]string, fallback string) map[string]string {
	lang := strings.TrimSpace(strings.ToLower(defaultLang))
	if lang == "" {
		lang = "ja"
	}
	defaultValue := strings.TrimSpace(values[lang])
	if defaultValue == "" {
		defaultValue = strings.TrimSpace(values["ja"])
	}
	if defaultValue == "" {
		defaultValue = fallback
	}
	jaValue := strings.TrimSpace(values["ja"])
	if jaValue == "" {
		jaValue = defaultValue
	}
	return map[string]string{"default": defaultValue, "ja": jaValue}
}

func normalizeLocalizedMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		trimmedKey := strings.TrimSpace(strings.ToLower(key))
		if trimmedKey == "" {
			continue
		}
		dst[trimmedKey] = strings.TrimSpace(value)
	}
	return dst
}
