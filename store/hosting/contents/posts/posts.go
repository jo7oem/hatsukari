package posts

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"
)

type Visibility string

const (
	VisibilityPublic     Visibility = "public"
	VisibilityUnlisted   Visibility = "unlisted"
	VisibilityDirectOnly Visibility = "directOnly"
	VisibilityPrivate    Visibility = "private"
)

type Revision struct {
	RevisedAt time.Time
	Summary   string
}

type Tag struct {
	Key   string
	URL   string
	Label map[string]string
	About map[string]string
}

type Entry struct {
	URL        string
	Title      string
	PostedAt   time.Time
	PublishAt  *time.Time
	Visibility Visibility
	Tags       []Tag
	Summary    string
	Revisions  []Revision
}

type TagFeed struct {
	Key   string
	URL   string
	Count int
	Label map[string]string
	About map[string]string
	Posts []Entry
}

func (p Entry) IsDirectVisible(now time.Time) bool {
	if !p.isPublishedAt(now) {
		return false
	}
	switch p.Visibility {
	case VisibilityPublic, VisibilityUnlisted, VisibilityDirectOnly:
		return true
	default:
		return false
	}
}

func (p Entry) IsListVisible(now time.Time) bool {
	return p.isPublishedAt(now) && p.Visibility == VisibilityPublic
}

func (p Entry) IsTagVisible(now time.Time) bool {
	if !p.isPublishedAt(now) {
		return false
	}
	return p.Visibility == VisibilityPublic || p.Visibility == VisibilityUnlisted
}

func (p Entry) LatestRevision() *Revision {
	if len(p.Revisions) == 0 {
		return nil
	}
	latest := p.Revisions[0]
	for i := 1; i < len(p.Revisions); i++ {
		if p.Revisions[i].RevisedAt.After(latest.RevisedAt) {
			latest = p.Revisions[i]
		}
	}
	return &latest
}

func ParseMetaTime(value any, location *time.Location) (time.Time, bool) {
	if location == nil {
		location = time.UTC
	}
	if value == nil {
		return time.Time{}, false
	}

	switch v := value.(type) {
	case time.Time:
		return v.In(location), true
	case string:
		parsed, ok := parseTimeString(v, location)
		return parsed, ok
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return time.Time{}, false
		}
		parsed, ok := parseTimeString(s, location)
		return parsed, ok
	}
}

func ParseVisibility(value any) Visibility {
	v := strings.TrimSpace(fmt.Sprint(value))
	if v == "" || v == "<nil>" {
		return VisibilityPublic
	}
	visibility := Visibility(v)
	switch visibility {
	case VisibilityPublic, VisibilityUnlisted, VisibilityDirectOnly, VisibilityPrivate:
		return visibility
	default:
		return VisibilityPrivate
	}
}

func ParseTagKeys(value any) []string {
	switch v := value.(type) {
	case []string:
		return uniqueTags(v)
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			t := strings.TrimSpace(fmt.Sprint(item))
			if t == "" || t == "<nil>" {
				continue
			}
			tags = append(tags, t)
		}
		return uniqueTags(tags)
	default:
		return nil
	}
}

func ParseRevisions(value any, location *time.Location) []Revision {
	v, ok := value.([]any)
	if !ok {
		return nil
	}
	revisions := make([]Revision, 0, len(v))
	for _, item := range v {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		revisedAt, ok := ParseMetaTime(m["revisedAt"], location)
		if !ok {
			continue
		}
		summary := strings.TrimSpace(fmt.Sprint(m["summary"]))
		if summary == "<nil>" {
			summary = ""
		}
		revisions = append(revisions, Revision{RevisedAt: revisedAt, Summary: summary})
	}
	return revisions
}

func BuildTagList(entries []Entry) []TagFeed {
	feeds := buildTagFeeds(entries)
	list := make([]TagFeed, 0, len(feeds))
	for _, feed := range feeds {
		list = append(list, feed)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Key < list[j].Key
	})
	return list
}

func BuildTagMap(entries []Entry) map[string]TagFeed {
	feeds := buildTagFeeds(entries)
	byTag := make(map[string]TagFeed, len(feeds))
	for key, feed := range feeds {
		if feed.URL == "" {
			continue
		}
		byTag[key] = feed
	}
	return byTag
}

func Limit(entries []Entry, max int) []Entry {
	if max <= 0 || len(entries) <= max {
		return entries
	}
	return entries[:max]
}

func normalizeLatest(value int, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}

func NormalizeLatest(value int, defaultValue int) int {
	return normalizeLatest(value, defaultValue)
}

func (p Entry) isPublishedAt(now time.Time) bool {
	if p.PublishAt == nil {
		return true
	}
	return !p.PublishAt.After(now)
}

func parseTimeString(raw string, location *time.Location) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.In(location), true
		}
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t.In(location), true
		}
	}
	return time.Time{}, false
}

func uniqueTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	unique := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func buildTagFeeds(entries []Entry) map[string]TagFeed {
	feeds := make(map[string]TagFeed)
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			feed, ok := feeds[tag.Key]
			if !ok {
				feed = TagFeed{Key: tag.Key, URL: tag.URL, Label: copyStringMap(tag.Label), About: copyStringMap(tag.About)}
			}
			feed.Count++
			feed.Posts = append(feed.Posts, entry)
			feeds[tag.Key] = feed
		}
	}
	return feeds
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}
