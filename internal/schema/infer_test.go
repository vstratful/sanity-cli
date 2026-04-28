package schema

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/vstratful/sanity-cli/internal/api"
	"github.com/vstratful/sanity-cli/internal/config"
)

// fakeClient is a test stub for api.Client. Each call to Query consults
// queryFunc with the GROQ string and params; queryFunc returns a JSON-encoded
// result.
type fakeClient struct {
	queryFunc func(groq string, params map[string]any) (any, error)
	calls     []fakeCall
}

type fakeCall struct {
	groq   string
	params map[string]any
}

func (f *fakeClient) Query(ctx context.Context, groq string, params map[string]any) (json.RawMessage, error) {
	f.calls = append(f.calls, fakeCall{groq, params})
	v, err := f.queryFunc(groq, params)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func (f *fakeClient) Mutate(context.Context, []json.RawMessage, *api.MutateOptions) (*api.MutateResponse, error) {
	return nil, errors.New("not used in schema tests")
}
func (f *fakeClient) UploadAsset(context.Context, api.AssetKind, io.Reader, *api.AssetUploadOptions) (json.RawMessage, error) {
	return nil, errors.New("not used in schema tests")
}
func (f *fakeClient) ListProjects(context.Context) ([]api.Project, error) {
	return nil, errors.New("not used in schema tests")
}
func (f *fakeClient) ListDatasets(context.Context, string) ([]api.Dataset, error) {
	return nil, errors.New("not used in schema tests")
}

// queryRouter returns a queryFunc that dispatches based on the GROQ string.
func queryRouter(types []string, samples map[string][]map[string]any, refResolution []map[string]string) func(string, map[string]any) (any, error) {
	return func(groq string, params map[string]any) (any, error) {
		switch groq {
		case `array::unique(*[]._type)`:
			return types, nil
		case `*[_type == $t][0...$n]`:
			t, _ := params["t"].(string)
			return samples[t], nil
		case `count(*[_type == $t])`:
			t, _ := params["t"].(string)
			return len(samples[t]), nil
		case `*[_id in $ids]{_id, _type}`:
			return refResolution, nil
		default:
			return nil, errors.New("unexpected query: " + groq)
		}
	}
}

func newInst() *config.Instance {
	return &config.Instance{
		ProjectID: "abc123",
		Dataset:   "production",
		Token:     "skTEST",
	}
}

func TestIntrospect_NoTypes(t *testing.T) {
	fc := &fakeClient{queryFunc: queryRouter([]string{}, nil, nil)}
	doc, err := Introspect(context.Background(), fc, newInst(), Options{})
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	if len(doc.Types) != 0 {
		t.Errorf("Types len=%d, want 0", len(doc.Types))
	}
	if doc.ProjectID != "abc123" {
		t.Errorf("ProjectID=%q, want abc123", doc.ProjectID)
	}
}

func TestIntrospect_PrimitiveFields(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "title": "First", "views": float64(10), "published": true, "createdAt": "2024-01-15T10:30:00Z"},
			{"_type": "post", "title": "Second", "views": float64(20), "published": false, "createdAt": "2024-02-15T10:30:00Z"},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, err := Introspect(context.Background(), fc, newInst(), Options{})
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	post := doc.Types["post"]
	if post == nil {
		t.Fatal("post type missing")
	}
	if post.SampleSize != 2 {
		t.Errorf("SampleSize=%d, want 2", post.SampleSize)
	}
	if post.Count != 2 {
		t.Errorf("Count=%d, want 2", post.Count)
	}
	if post.Empty {
		t.Error("Empty=true, want false")
	}
	checkField(t, post.Fields, "title", "string", false, false)
	checkField(t, post.Fields, "views", "number", false, false)
	checkField(t, post.Fields, "published", "boolean", false, false)
	checkField(t, post.Fields, "createdAt", "datetime", false, false)
}

func TestIntrospect_Optional(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "title": "A", "subtitle": "S"},
			{"_type": "post", "title": "B"}, // no subtitle
			{"_type": "post", "title": "C"},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	checkField(t, doc.Types["post"].Fields, "title", "string", false, false)
	checkField(t, doc.Types["post"].Fields, "subtitle", "string", true, false)
}

func TestIntrospect_Nullable(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "summary": "hello"},
			{"_type": "post", "summary": nil},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	checkField(t, doc.Types["post"].Fields, "summary", "string", false, true)
}

func TestIntrospect_Empty(t *testing.T) {
	fc := &fakeClient{queryFunc: queryRouter([]string{"empty"}, map[string][]map[string]any{}, nil)}
	doc, err := Introspect(context.Background(), fc, newInst(), Options{})
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	emptyTI := doc.Types["empty"]
	if emptyTI == nil {
		t.Fatal("empty type missing")
	}
	if !emptyTI.Empty {
		t.Error("Empty=false, want true")
	}
}

func TestIntrospect_Reference(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "author": map[string]any{"_type": "reference", "_ref": "author-1"}},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	author := doc.Types["post"].Fields["author"]
	if author == nil || author.Type != "reference" {
		t.Fatalf("author Field=%+v, want type=reference", author)
	}
}

func TestIntrospect_ReferenceResolution(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "author": map[string]any{"_type": "reference", "_ref": "author-1"}},
		},
	}
	resolution := []map[string]string{
		{"_id": "author-1", "_type": "author"},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, resolution)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{ResolveReferences: true})
	author := doc.Types["post"].Fields["author"]
	if author == nil || author.Type != "reference" {
		t.Fatalf("author Field=%+v", author)
	}
	if len(author.To) != 1 || author.To[0] != "author" {
		t.Errorf("author.To=%v, want [author]", author.To)
	}
}

func TestIntrospect_AssetImage(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "cover": map[string]any{"_type": "image", "asset": map[string]any{"_ref": "img-1"}}},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	cover := doc.Types["post"].Fields["cover"]
	if cover == nil || cover.Type != "asset" || cover.AssetKind != "image" {
		t.Fatalf("cover=%+v, want type=asset asset_kind=image", cover)
	}
}

func TestIntrospect_PortableText(t *testing.T) {
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "body": []any{
				map[string]any{
					"_type": "block", "style": "h2",
					"children": []any{map[string]any{"marks": []any{"strong"}}},
				},
				map[string]any{
					"_type": "block", "style": "normal",
					"children": []any{map[string]any{"marks": []any{"em"}}},
					"markDefs": []any{map[string]any{"_type": "link"}},
				},
			}},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	body := doc.Types["post"].Fields["body"]
	if body == nil {
		t.Fatal("body field missing")
	}
	if body.Type != "array" {
		t.Errorf("body.Type=%q, want array", body.Type)
	}
	if body.Tag != "portableText" {
		t.Errorf("body.Tag=%q, want portableText", body.Tag)
	}
	if body.Blocks == nil {
		t.Fatal("body.Blocks nil")
	}
	if !contains(body.Blocks.Styles, "h2") || !contains(body.Blocks.Styles, "normal") {
		t.Errorf("Blocks.Styles=%v missing entries", body.Blocks.Styles)
	}
	if !contains(body.Blocks.Marks, "strong") || !contains(body.Blocks.Marks, "em") || !contains(body.Blocks.Marks, "link") {
		t.Errorf("Blocks.Marks=%v missing entries", body.Blocks.Marks)
	}
}

func TestIntrospect_NestedObjectOptional(t *testing.T) {
	// A nested object whose inner field appears in 1 of 3 docs that have the
	// nested object — should be Optional=true.
	samples := map[string][]map[string]any{
		"post": {
			{"_type": "post", "slug": map[string]any{"current": "a", "source": "title"}},
			{"_type": "post", "slug": map[string]any{"current": "b"}},
			{"_type": "post", "slug": map[string]any{"current": "c"}},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"post"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	slug := doc.Types["post"].Fields["slug"]
	if slug == nil || slug.Type != "object" {
		t.Fatalf("slug=%+v, want object", slug)
	}
	current := slug.Fields["current"]
	if current == nil || current.Optional {
		t.Errorf("current.Optional=%v, want false (always present)", current.Optional)
	}
	source := slug.Fields["source"]
	if source == nil || !source.Optional {
		t.Errorf("source.Optional=%v, want true (1 of 3)", source.Optional)
	}
}

func TestIntrospect_MaxDepthTruncates(t *testing.T) {
	// With MaxDepth=2, depths 0 and 1 are explored; depth 2 hits the limit
	// so the field at that level is marked Truncated.
	samples := map[string][]map[string]any{
		"deep": {
			{"_type": "deep", "a": map[string]any{
				"b": map[string]any{
					"c": map[string]any{
						"d": map[string]any{"e": "nested"},
					},
				},
			}},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"deep"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{MaxDepth: 2})
	a := doc.Types["deep"].Fields["a"]
	if a == nil || a.Fields == nil {
		t.Fatal("a or a.Fields missing")
	}
	b := a.Fields["b"]
	if b == nil || b.Fields == nil {
		t.Fatal("b or b.Fields missing")
	}
	c := b.Fields["c"]
	if c == nil {
		t.Fatal("c missing")
	}
	if !c.Truncated {
		t.Errorf("c.Truncated=%v, want true", c.Truncated)
	}
	if c.Fields != nil {
		t.Errorf("c.Fields should be nil at truncation, got %v", c.Fields)
	}
}

func TestIntrospect_UnionMixedTypes(t *testing.T) {
	samples := map[string][]map[string]any{
		"weird": {
			{"_type": "weird", "value": "string-here"},
			{"_type": "weird", "value": float64(42)},
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"weird"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	v := doc.Types["weird"].Fields["value"]
	if v == nil || v.Type != "union" {
		t.Fatalf("value=%+v, want type=union", v)
	}
	if !contains(v.Union, "string") || !contains(v.Union, "number") {
		t.Errorf("Union=%v, want both string and number", v.Union)
	}
}

func TestIntrospect_RFC3339DatetimeDetection(t *testing.T) {
	samples := map[string][]map[string]any{
		"x": {
			{"_type": "x", "a": "2024-01-15T10:30:00Z"},
			{"_type": "x", "b": "2024-01-15T10:30:00.123Z"},
			{"_type": "x", "c": "2024-01-15T10:30:00+02:00"},
			{"_type": "x", "d": "not a date"},
			{"_type": "x", "e": "2024-01-15"}, // date only, not datetime
		},
	}
	fc := &fakeClient{queryFunc: queryRouter([]string{"x"}, samples, nil)}
	doc, _ := Introspect(context.Background(), fc, newInst(), Options{})
	for _, k := range []string{"a", "b", "c"} {
		f := doc.Types["x"].Fields[k]
		if f == nil || f.Type != "datetime" {
			t.Errorf("field %q type=%q, want datetime", k, fieldType(f))
		}
	}
	for _, k := range []string{"d", "e"} {
		f := doc.Types["x"].Fields[k]
		if f == nil || f.Type != "string" {
			t.Errorf("field %q type=%q, want string", k, fieldType(f))
		}
	}
}

// helpers ---------------------------------------------------------------

func checkField(t *testing.T, fields map[string]*Field, name, wantType string, wantOptional, wantNullable bool) {
	t.Helper()
	f := fields[name]
	if f == nil {
		t.Errorf("field %q missing", name)
		return
	}
	if f.Type != wantType {
		t.Errorf("field %q type=%q, want %q", name, f.Type, wantType)
	}
	if f.Optional != wantOptional {
		t.Errorf("field %q optional=%v, want %v", name, f.Optional, wantOptional)
	}
	if f.Nullable != wantNullable {
		t.Errorf("field %q nullable=%v, want %v", name, f.Nullable, wantNullable)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func fieldType(f *Field) string {
	if f == nil {
		return "<nil>"
	}
	return f.Type
}
