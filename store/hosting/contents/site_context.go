package contents

import "maps"

type SiteContext struct {
	Title     string
	Variables map[string]any
	Indexes   []map[string]any
}

func (c SiteContext) VariablesMap(posts map[string]any) map[string]any {
	values := make(map[string]any, 4)
	values["title"] = c.Title
	if c.Variables == nil {
		values["variables"] = map[string]any{}
	} else {
		values["variables"] = maps.Clone(c.Variables)
	}
	if c.Indexes == nil {
		values["indexes"] = []map[string]any{}
	} else {
		indexes := make([]map[string]any, 0, len(c.Indexes))
		for _, item := range c.Indexes {
			indexes = append(indexes, maps.Clone(item))
		}
		values["indexes"] = indexes
	}
	if posts != nil {
		values["posts"] = posts
	}
	return values
}

func (c SiteContext) VariablesMapWithLocale(posts map[string]any, locale string) map[string]any {
	values := c.VariablesMap(posts)
	values["currentLocale"] = locale
	return values
}
