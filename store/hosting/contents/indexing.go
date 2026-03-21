package contents

import "maps"

import "strings"

type IndexSeed struct {
	URL           string
	IndexTitle    map[string]string
	DefaultLocale string
	Priority      int
	LoadOrder     int
}

func (c *Content) CollectIndexSeeds() []IndexSeed {
	seeds := make([]IndexSeed, 0)
	loadOrder := 0
	c.collectIndexSeeds(&seeds, &loadOrder)
	return seeds
}

func (c *Content) CollectPostsContents() []*Content {
	collected := make([]*Content, 0)
	c.collectPostsContents(&collected)
	return collected
}

func (c *Content) collectPostsContents(collected *[]*Content) {
	if c.isPostsContent() {
		*collected = append(*collected, c)
	}
	for _, child := range c.children {
		child.collectPostsContents(collected)
	}
}

func (c *Content) collectIndexSeeds(seeds *[]IndexSeed, loadOrder *int) {
	if c.config.RegisterIndexing {
		seed := IndexSeed{
			URL:           normalizeIndexURL(c.Path()),
			IndexTitle:    copyIndexTitleMap(c.config.IndexTitle),
			DefaultLocale: strings.TrimSpace(c.config.DefaultLocale),
			Priority:      c.priority(),
			LoadOrder:     *loadOrder,
		}
		*loadOrder = *loadOrder + 1
		*seeds = append(*seeds, seed)
	}

	for _, child := range c.children {
		child.collectIndexSeeds(seeds, loadOrder)
	}
}

func copyIndexTitleMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

func normalizeIndexURL(rawURL string) string {
	if rawURL == "/" {
		return rawURL
	}
	return strings.TrimSuffix(rawURL, "/") + "/"
}

func (c *Content) priority() int {
	if c.config.Priority == nil {
		return defaultContentPriority
	}
	return *c.config.Priority
}
