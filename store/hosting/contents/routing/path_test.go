package routing

import "testing"

func TestPath_RequestURLToRelPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mountPath string
		urlPath   string
		wantPath  string
		wantOK    bool
	}{
		{name: "RootIndex", mountPath: "/", urlPath: "/", wantPath: "index", wantOK: true},
		{name: "RootChild", mountPath: "/", urlPath: "/posts", wantPath: "posts", wantOK: true},
		{name: "SubMountIndex", mountPath: "/posts/", urlPath: "/posts", wantPath: "index", wantOK: true},
		{name: "SubMountChild", mountPath: "/posts/", urlPath: "/posts/tags", wantPath: "tags", wantOK: true},
		{name: "OutsideMount", mountPath: "/posts/", urlPath: "/about", wantPath: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotPath, gotOK := RequestURLToRelPath(tt.mountPath, tt.urlPath)
			if gotPath != tt.wantPath || gotOK != tt.wantOK {
				t.Fatalf("RequestURLToRelPath() = (%q, %v), want (%q, %v)", gotPath, gotOK, tt.wantPath, tt.wantOK)
			}
		})
	}
}

func TestPath_IsHiddenOrUnsafeRelPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		relPath string
		want    bool
	}{
		{name: "Empty", relPath: "", want: true},
		{name: "Current", relPath: ".", want: true},
		{name: "Parent", relPath: "../x", want: true},
		{name: "DotFile", relPath: ".secret", want: true},
		{name: "DotDir", relPath: "foo/.git/config", want: true},
		{name: "Safe", relPath: "posts/index", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsHiddenOrUnsafeRelPath(tt.relPath); got != tt.want {
				t.Fatalf("IsHiddenOrUnsafeRelPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
