package site

import (
	"sort"
	"strings"

	"github.com/jo7oem/hatsukari/store/hosting/contents"
)

func buildSiteIndexes(seeds []contents.IndexSeed) []map[string]any {
	sort.SliceStable(seeds, func(i, j int) bool {
		if seeds[i].Priority != seeds[j].Priority {
			return seeds[i].Priority < seeds[j].Priority
		}
		return seeds[i].LoadOrder < seeds[j].LoadOrder
	})

	indexes := make([]map[string]any, 0, len(seeds))
	for _, seed := range seeds {
		jaTitle := resolveTitle(seed, "ja")
		if jaTitle == "" {
			jaTitle = seed.URL
		}

		defaultLocale := strings.TrimSpace(seed.DefaultLocale)
		if defaultLocale == "" {
			defaultLocale = "ja"
		}

		defaultTitle := resolveTitle(seed, defaultLocale)
		if defaultTitle == "" {
			defaultTitle = jaTitle
		}
		if defaultTitle == "" {
			defaultTitle = seed.URL
		}

		indexes = append(indexes, map[string]any{
			"url": seed.URL,
			"title": map[string]string{
				"default": defaultTitle,
				"ja":      jaTitle,
			},
		})
	}

	return indexes
}

func resolveTitle(seed contents.IndexSeed, locale string) string {
	if len(seed.IndexTitle) == 0 {
		return ""
	}
	return strings.TrimSpace(seed.IndexTitle[locale])
}
