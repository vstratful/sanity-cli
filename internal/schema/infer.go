package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/vstratful/sanity-cli/internal/api"
	"github.com/vstratful/sanity-cli/internal/config"
)

// Options configures schema introspection.
type Options struct {
	SampleSize int
	MaxDepth   int
	// ResolveReferences, if true, runs a follow-up GROQ batch to resolve the
	// _type of every reference observed in samples.
	ResolveReferences bool
}

func (o Options) withDefaults() Options {
	if o.SampleSize <= 0 {
		o.SampleSize = 50
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = 6
	}
	return o
}

var rfc3339Re = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})$`)

// Introspect builds a schema document by querying the live dataset.
func Introspect(ctx context.Context, client api.Client, inst *config.Instance, opts Options) (*Doc, error) {
	opts = opts.withDefaults()

	// 1. List distinct _type values.
	rawTypes, err := client.Query(ctx, `array::unique(*[]._type)`, nil)
	if err != nil {
		return nil, fmt.Errorf("listing types: %w", err)
	}
	var allTypes []string
	if err := json.Unmarshal(rawTypes, &allTypes); err != nil {
		return nil, fmt.Errorf("parsing types: %w", err)
	}
	sort.Strings(allTypes)

	doc := &Doc{
		ProjectID:      inst.ProjectID,
		Dataset:        inst.Dataset,
		APIVersion:     inst.EffectiveAPIVersion(),
		Perspective:    inst.EffectivePerspective(),
		IntrospectedAt: time.Now().UTC().Format(time.RFC3339),
		SampleSize:     opts.SampleSize,
		MaxDepth:       opts.MaxDepth,
		Types:          map[string]*TypeInfo{},
	}

	refTargets := map[string]struct{}{} // collected reference _refs

	for _, t := range allTypes {
		if t == "" {
			continue
		}
		ti, refs, err := introspectType(ctx, client, t, opts)
		if err != nil {
			return nil, fmt.Errorf("type %q: %w", t, err)
		}
		doc.Types[t] = ti
		if opts.ResolveReferences {
			for r := range refs {
				refTargets[r] = struct{}{}
			}
		}
	}

	if opts.ResolveReferences && len(refTargets) > 0 {
		_ = resolveReferenceTargets(ctx, client, doc, refTargets)
	}

	return doc, nil
}

// introspectType samples docs of a given type and infers their field shapes.
func introspectType(ctx context.Context, client api.Client, t string, opts Options) (*TypeInfo, map[string]struct{}, error) {
	groq := fmt.Sprintf(`*[_type == $t][0...$n]`)
	params := map[string]any{"t": t, "n": opts.SampleSize}
	raw, err := client.Query(ctx, groq, params)
	if err != nil {
		return nil, nil, fmt.Errorf("sampling: %w", err)
	}
	var samples []map[string]any
	if err := json.Unmarshal(raw, &samples); err != nil {
		return nil, nil, fmt.Errorf("parsing samples: %w", err)
	}

	ti := &TypeInfo{SampleSize: len(samples)}
	if len(samples) == 0 {
		ti.Empty = true
		// Best-effort: still try to count.
		if cnt, err := countOf(ctx, client, t); err == nil {
			ti.Count = cnt
		}
		return ti, nil, nil
	}

	if cnt, err := countOf(ctx, client, t); err == nil {
		ti.Count = cnt
	}

	fields := map[string]*Field{}
	refs := map[string]struct{}{}
	for _, doc := range samples {
		mergeObject(fields, doc, 0, opts.MaxDepth, refs)
	}
	finalize(fields, len(samples))
	ti.Fields = fields
	return ti, refs, nil
}

func countOf(ctx context.Context, client api.Client, t string) (int, error) {
	raw, err := client.Query(ctx, `count(*[_type == $t])`, map[string]any{"t": t})
	if err != nil {
		return 0, err
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, err
	}
	return n, nil
}

// mergeObject folds the keys/values of obj into fields, recording how many
// times each field was observed. finalize() uses those counts (along with the
// parent's seen count) to compute optional/nullable flags.
func mergeObject(fields map[string]*Field, obj map[string]any, depth, maxDepth int, refs map[string]struct{}) {
	for k, v := range obj {
		f, ok := fields[k]
		if !ok {
			f = &Field{}
			fields[k] = f
		}
		f.seenCount++
		if v == nil {
			f.Nullable = true
			continue
		}
		mergeValue(f, v, depth, maxDepth, refs)
	}
}

// mergeValue updates a field shape with the observation `v`.
func mergeValue(f *Field, v any, depth, maxDepth int, refs map[string]struct{}) {
	if depth >= maxDepth {
		f.Truncated = true
		setOrUnion(f, "object")
		return
	}

	switch val := v.(type) {
	case string:
		if rfc3339Re.MatchString(val) {
			setOrUnion(f, "datetime")
		} else {
			setOrUnion(f, "string")
		}
	case bool:
		setOrUnion(f, "boolean")
	case float64, int, int64:
		setOrUnion(f, "number")
	case nil:
		f.Nullable = true
	case []any:
		setOrUnion(f, "array")
		// Detect portable text: array of {_type:"block"}
		blockCount := 0
		styles := uniqueSet{}
		marks := uniqueSet{}
		elementShapes := f.Of
		for _, el := range val {
			if elObj, ok := el.(map[string]any); ok {
				if t, _ := elObj["_type"].(string); t == "block" {
					blockCount++
					if s, _ := elObj["style"].(string); s != "" {
						styles.add(s)
					}
					if mds, ok := elObj["markDefs"].([]any); ok {
						for _, md := range mds {
							if mdo, ok := md.(map[string]any); ok {
								if mt, _ := mdo["_type"].(string); mt != "" {
									marks.add(mt)
								}
							}
						}
					}
					if children, ok := elObj["children"].([]any); ok {
						for _, ch := range children {
							if cho, ok := ch.(map[string]any); ok {
								if mks, ok := cho["marks"].([]any); ok {
									for _, m := range mks {
										if ms, _ := m.(string); ms != "" {
											marks.add(ms)
										}
									}
								}
							}
						}
					}
					continue
				}
			}
			// Generic element: build a child Field and merge its observation.
			elField := newOrMatch(elementShapes, el)
			elementShapes = elField.parent
			mergeValue(elField.field, el, depth+1, maxDepth, refs)
		}
		f.Of = elementShapes
		if blockCount > 0 && blockCount == len(val) {
			f.Tag = "portableText"
			if f.Blocks == nil {
				f.Blocks = &BlockSummary{}
			}
			mergeUnique(&f.Blocks.Styles, styles.values())
			mergeUnique(&f.Blocks.Marks, marks.values())
		}
	case map[string]any:
		// References
		if t, _ := val["_type"].(string); t == "reference" {
			setOrUnion(f, "reference")
			if r, _ := val["_ref"].(string); r != "" {
				refs[r] = struct{}{}
			}
			return
		}
		// Assets
		if t, _ := val["_type"].(string); t == "image" || t == "file" {
			setOrUnion(f, "asset")
			f.AssetKind = t
			return
		}
		setOrUnion(f, "object")
		if f.Fields == nil {
			f.Fields = map[string]*Field{}
		}
		mergeObject(f.Fields, val, depth+1, maxDepth, refs)
	default:
		setOrUnion(f, "unknown")
	}
}

// setOrUnion sets f.Type to t, or marks it as a union if it already had a
// different type.
func setOrUnion(f *Field, t string) {
	if f.Type == "" {
		f.Type = t
		return
	}
	if f.Type == t {
		return
	}
	// Promote to union.
	if f.Type != "union" {
		prev := f.Type
		f.Type = "union"
		f.Union = appendUnique(f.Union, prev)
	}
	f.Union = appendUnique(f.Union, t)
}

// elementOf is a tracked array-element child Field plus a back-pointer so we
// can replace it in the slice on union promotion.
type elementOf struct {
	field  *Field
	parent []*Field
}

func newOrMatch(of []*Field, v any) elementOf {
	// Try to find an existing element shape that matches the JSON kind of v.
	kind := jsonKind(v)
	for _, f := range of {
		if f.Type == kind {
			return elementOf{field: f, parent: of}
		}
		if f.Type == "union" {
			for _, u := range f.Union {
				if u == kind {
					return elementOf{field: f, parent: of}
				}
			}
		}
	}
	// Create a new shape entry.
	f := &Field{}
	of = append(of, f)
	return elementOf{field: f, parent: of}
}

func jsonKind(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	case string:
		if rfc3339Re.MatchString(x) {
			return "datetime"
		}
		return "string"
	case []any:
		return "array"
	case map[string]any:
		if t, _ := x["_type"].(string); t == "reference" {
			return "reference"
		}
		if t, _ := x["_type"].(string); t == "image" || t == "file" {
			return "asset"
		}
		return "object"
	default:
		return "unknown"
	}
}

// finalize converts internal counters into Optional/Nullable flags.
func finalize(fields map[string]*Field, totalSamples int) {
	for _, f := range fields {
		if f.totalCount == 0 {
			f.totalCount = totalSamples
		}
		if f.seenCount < f.totalCount {
			f.Optional = true
		}
		if f.Fields != nil {
			finalize(f.Fields, f.seenCount)
		}
	}
}

func resolveReferenceTargets(ctx context.Context, client api.Client, doc *Doc, refs map[string]struct{}) error {
	ids := make([]string, 0, len(refs))
	for r := range refs {
		ids = append(ids, r)
	}
	if len(ids) > 200 {
		ids = ids[:200]
	}
	raw, err := client.Query(ctx, `*[_id in $ids]{_id, _type}`, map[string]any{"ids": ids})
	if err != nil {
		return err
	}
	var items []struct {
		ID    string `json:"_id"`
		Type_ string `json:"_type"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	idToType := map[string]string{}
	for _, it := range items {
		idToType[it.ID] = it.Type_
	}
	// Walk every Field in every type and any reference field gets target types
	// matched against refs we observed (best-effort: union of all resolved
	// targets across the dataset).
	allTargets := uniqueSet{}
	for _, t := range idToType {
		if t != "" {
			allTargets.add(t)
		}
	}
	walk := func(_ string, f *Field) {
		if f.Type == "reference" {
			f.To = appendUniqueAll(f.To, allTargets.values())
			if len(f.To) == 0 {
				f.Unresolved = true
			}
		}
	}
	for _, ti := range doc.Types {
		walkFields(ti.Fields, walk)
	}
	return nil
}

func walkFields(fields map[string]*Field, fn func(name string, f *Field)) {
	for k, f := range fields {
		fn(k, f)
		if f.Fields != nil {
			walkFields(f.Fields, fn)
		}
		for _, of := range f.Of {
			fn(k+"[]", of)
			if of.Fields != nil {
				walkFields(of.Fields, fn)
			}
		}
	}
}

// uniqueSet is a tiny string-set that preserves insertion order via a slice.
type uniqueSet struct {
	idx  map[string]struct{}
	keys []string
}

func (u *uniqueSet) add(s string) {
	if u.idx == nil {
		u.idx = map[string]struct{}{}
	}
	if _, ok := u.idx[s]; ok {
		return
	}
	u.idx[s] = struct{}{}
	u.keys = append(u.keys, s)
}

func (u *uniqueSet) values() []string {
	if u == nil {
		return nil
	}
	out := make([]string, len(u.keys))
	copy(out, u.keys)
	sort.Strings(out)
	return out
}

func appendUnique(slice []string, v string) []string {
	for _, s := range slice {
		if s == v {
			return slice
		}
	}
	return append(slice, v)
}

func appendUniqueAll(slice []string, vs []string) []string {
	for _, v := range vs {
		slice = appendUnique(slice, v)
	}
	return slice
}

func mergeUnique(dst *[]string, vs []string) {
	for _, v := range vs {
		*dst = appendUnique(*dst, v)
	}
}
