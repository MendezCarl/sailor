package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MendezCarl/sailor.git/internal/request"
)

// --- helpers -----------------------------------------------------------------

func writeCollection(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeCollection: %v", err)
	}
	return path
}

func mustLoadFile(t *testing.T, path string) *Collection {
	t.Helper()
	c, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return c
}

// --- splitDotPath ------------------------------------------------------------

func TestSplitDotPath_NoDot(t *testing.T) {
	got := splitDotPath("List Users")
	if len(got) != 1 || got[0] != "List Users" {
		t.Errorf("got %v, want [List Users]", got)
	}
}

func TestSplitDotPath_SingleDot(t *testing.T) {
	got := splitDotPath("Users.List Users")
	want := []string{"Users", "List Users"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitDotPath_Nested(t *testing.T) {
	got := splitDotPath("Admin.Users.List All Users")
	want := []string{"Admin", "Users", "List All Users"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitDotPath_EscapedDot(t *testing.T) {
	// "Admin\.V2.List" → ["Admin.V2", "List"]
	got := splitDotPath(`Admin\.V2.List`)
	want := []string{"Admin.V2", "List"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// --- FindRequest (top-level) -------------------------------------------------

func TestFindRequest_TopLevel(t *testing.T) {
	c := &Collection{
		Name: "Test",
		Requests: []*request.Request{
			{Name: "List Users", Method: "GET", URL: "/users"},
			{Name: "Get User", Method: "GET", URL: "/users/1"},
		},
	}
	req, err := FindRequest(c, "List Users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "List Users" {
		t.Errorf("got %q, want %q", req.Name, "List Users")
	}
}

func TestFindRequest_TopLevelNotFound(t *testing.T) {
	c := &Collection{
		Name:     "Test",
		Requests: []*request.Request{{Name: "List Users", Method: "GET", URL: "/users"}},
	}
	_, err := FindRequest(c, "Delete User")
	if err == nil {
		t.Fatal("expected error for missing request")
	}
}

// --- FindRequest (folder paths) ----------------------------------------------

func TestFindRequest_FolderPath(t *testing.T) {
	c := &Collection{
		Name: "Test",
		Folders: []*Folder{
			{
				Name: "Users",
				Requests: []*request.Request{
					{Name: "List Users", Method: "GET", URL: "/users"},
				},
			},
		},
	}
	req, err := FindRequest(c, "Users.List Users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "List Users" {
		t.Errorf("got %q, want %q", req.Name, "List Users")
	}
}

func TestFindRequest_NestedFolderPath(t *testing.T) {
	c := &Collection{
		Name: "Test",
		Folders: []*Folder{
			{
				Name: "Admin",
				Folders: []*Folder{
					{
						Name: "Users",
						Requests: []*request.Request{
							{Name: "List All Users", Method: "GET", URL: "/admin/users"},
						},
					},
				},
			},
		},
	}
	req, err := FindRequest(c, "Admin.Users.List All Users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "List All Users" {
		t.Errorf("got %q, want %q", req.Name, "List All Users")
	}
}

func TestFindRequest_EscapedDot(t *testing.T) {
	c := &Collection{
		Name: "Test",
		Folders: []*Folder{
			{
				Name: "Admin.V2",
				Requests: []*request.Request{
					{Name: "List", Method: "GET", URL: "/admin/v2/list"},
				},
			},
		},
	}
	req, err := FindRequest(c, `Admin\.V2.List`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != "List" {
		t.Errorf("got %q, want %q", req.Name, "List")
	}
}

func TestFindRequest_FolderMissError_ListsAvailable(t *testing.T) {
	c := &Collection{
		Name:     "Test",
		Requests: []*request.Request{{Name: "Health", Method: "GET", URL: "/health"}},
		Folders: []*Folder{
			{
				Name: "Users",
				Requests: []*request.Request{
					{Name: "List Users", Method: "GET", URL: "/users"},
				},
			},
		},
	}
	_, err := FindRequest(c, "No Such Request")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !contains(msg, "Health") {
		t.Errorf("error should mention top-level request %q, got: %s", "Health", msg)
	}
	if !contains(msg, "Users.List Users") {
		t.Errorf("error should mention folder path %q, got: %s", "Users.List Users", msg)
	}
}

// --- LoadFile (method normalization in folders) ------------------------------

func TestLoadFile_FolderRequestsMethodNormalized(t *testing.T) {
	dir := t.TempDir()
	content := `
name: Test Collection
folders:
  - name: Users
    requests:
      - name: List Users
        method: get
        url: /users
`
	path := writeCollection(t, dir, "col.yaml", content)
	c := mustLoadFile(t, path)

	if len(c.Folders) == 0 || len(c.Folders[0].Requests) == 0 {
		t.Fatal("expected folder with requests")
	}
	got := c.Folders[0].Requests[0].Method
	if got != "GET" {
		t.Errorf("method: got %q, want \"GET\"", got)
	}
}

func TestLoadFile_TopLevelAndFolderRequests(t *testing.T) {
	dir := t.TempDir()
	content := `
name: Mixed
requests:
  - name: Health Check
    method: GET
    url: /health
folders:
  - name: Users
    requests:
      - name: List Users
        method: GET
        url: /users
`
	path := writeCollection(t, dir, "col.yaml", content)
	c := mustLoadFile(t, path)

	if len(c.Requests) != 1 {
		t.Errorf("top-level requests: got %d, want 1", len(c.Requests))
	}
	if len(c.Folders) != 1 || len(c.Folders[0].Requests) != 1 {
		t.Errorf("folder requests: unexpected structure")
	}
}

// --- ListAll -----------------------------------------------------------------

func TestListAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	entries, err := ListAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListAll_MissingDir(t *testing.T) {
	entries, err := ListAll("/no/such/directory")
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestListAll_SortedByName(t *testing.T) {
	dir := t.TempDir()
	writeCollection(t, dir, "b.yaml", "name: Zebra Collection\nrequests:\n  - name: R\n    method: GET\n    url: /r\n")
	writeCollection(t, dir, "a.yaml", "name: Alpha Collection\nrequests:\n  - name: R\n    method: GET\n    url: /r\n")

	entries, err := ListAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Collection.Name != "Alpha Collection" {
		t.Errorf("first: got %q, want \"Alpha Collection\"", entries[0].Collection.Name)
	}
	if entries[1].Collection.Name != "Zebra Collection" {
		t.Errorf("second: got %q, want \"Zebra Collection\"", entries[1].Collection.Name)
	}
}

func TestListAll_InvalidFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	writeCollection(t, dir, "bad.yaml", "this is not valid: [yaml collection")
	writeCollection(t, dir, "good.yaml", "name: Good\nrequests:\n  - name: R\n    method: GET\n    url: /r\n")

	entries, err := ListAll(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Collection.Name != "Good" {
		t.Errorf("expected only good collection, got %v", entries)
	}
}

// --- CountRequests -----------------------------------------------------------

func TestCountRequests_TopLevelOnly(t *testing.T) {
	c := &Collection{
		Requests: []*request.Request{{}, {}, {}},
	}
	if n := CountRequests(c); n != 3 {
		t.Errorf("got %d, want 3", n)
	}
}

func TestCountRequests_WithFolders(t *testing.T) {
	c := &Collection{
		Requests: []*request.Request{{}},
		Folders: []*Folder{
			{
				Requests: []*request.Request{{}, {}},
				Folders: []*Folder{
					{Requests: []*request.Request{{}}},
				},
			},
		},
	}
	if n := CountRequests(c); n != 4 {
		t.Errorf("got %d, want 4", n)
	}
}

// --- helpers -----------------------------------------------------------------

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
