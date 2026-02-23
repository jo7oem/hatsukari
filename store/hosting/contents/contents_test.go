package contents

import (
	"testing"
)

func TestContent_Path(t *testing.T) {
	tests := []struct {
		name   string
		parent *Content
		path   string
		want   string
	}{
		{
			name:   "root",
			parent: nil,
			path:   "",
			want:   "/",
		},
		{
			name: "child",
			parent: &Content{
				parent: nil,
				path:   "parent",
			},
			path: "child",
			want: "/child/",
		},
		{
			name: "grandchild",
			parent: &Content{
				parent: &Content{
					parent: nil,
					path:   "",
				},
				path: "parent",
			},
			path: "grandchild",
			want: "/parent/grandchild/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Content{
				parent: tt.parent,
				path:   tt.path,
			}
			if got := c.Path(); got != tt.want {
				t.Errorf("Path() = %v, want %v", got, tt.want)
			}
		})
	}
}
