package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// LabelListOptions controls the fields included in text label listings.
type LabelListOptions struct {
	// IncludeIDs includes the Linear ID on each label and group header.
	IncludeIDs bool
}

// FormatLabels renders labels as a deterministic, grouped text list. Labels
// with a Parent are rendered as alternatives under one parent header, even
// when the API returns the parent after its children (or omits it entirely).
// The input labels are never modified.
func FormatLabels(labels []core.Label, options LabelListOptions) string {
	if len(labels) == 0 {
		return "No labels found."
	}

	parents := make(map[string]core.Label)
	groups := make(map[string][]core.Label)
	for _, label := range labels {
		if label.Parent == nil {
			parents[label.ID] = label
			continue
		}

		// Parent ID is the authoritative group key. Keep a name fallback for
		// incomplete fixtures/responses so children are not accidentally merged.
		key := label.Parent.ID
		if key == "" {
			key = "name:" + label.Parent.Name
		}
		groups[key] = append(groups[key], label)
	}

	// A parent that has children is a group header, not also a standalone
	// record. Other parent labels remain standalone labels.
	groupParents := make(map[string]core.Label)
	for key := range groups {
		if parent, ok := parents[key]; ok {
			groupParents[key] = parent
			delete(parents, key)
		}
		groups[key] = sortLabels(groups[key])
	}
	standalone := make([]core.Label, 0, len(parents))
	for _, label := range parents {
		standalone = append(standalone, label)
	}
	standalone = sortLabels(standalone)

	type entry struct {
		key    string
		parent *core.Label
		ref    *core.LabelRef
		labels []core.Label
	}
	entries := make([]entry, 0, len(groups)+len(standalone))
	for key, children := range groups {
		if parent, ok := groupParents[key]; ok {
			parentCopy := parent
			entries = append(entries, entry{key: labelSortKey(parent), parent: &parentCopy, labels: children})
			continue
		}
		ref := groupReference(labels, key)
		entries = append(entries, entry{key: labelSortKeyRef(ref), ref: ref, labels: children})
	}
	for _, label := range standalone {
		labelCopy := label
		entries = append(entries, entry{key: labelSortKey(label), parent: &labelCopy})
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

	var b strings.Builder
	b.WriteString(fmt.Sprintf("LABELS (%d)\n", len(labels)))
	b.WriteString(line(40))
	b.WriteByte('\n')
	for _, item := range entries {
		if item.parent != nil && len(item.labels) > 0 {
			b.WriteString("  GROUP: ")
			writeLabel(&b, *item.parent, options.IncludeIDs)
			writeDescription(&b, "    ", item.parent.Description)
			for _, child := range item.labels {
				writeLabelIndented(&b, child, "    ", options.IncludeIDs)
				writeDescription(&b, "      ", child.Description)
			}
			continue
		}
		if item.ref != nil {
			b.WriteString("  GROUP: ")
			b.WriteString(labelReferenceName(*item.ref))
			if options.IncludeIDs && item.ref.ID != "" {
				b.WriteString(" [")
				b.WriteString(item.ref.ID)
				b.WriteString("]")
			}
			b.WriteByte('\n')
			for _, child := range item.labels {
				writeLabelIndented(&b, child, "    ", options.IncludeIDs)
				writeDescription(&b, "      ", child.Description)
			}
			continue
		}
		writeLabelIndented(&b, *item.parent, "  ", options.IncludeIDs)
		writeDescription(&b, "    ", item.parent.Description)
	}
	return b.String()
}

// LabelList renders the label-list text format used by services.
func (f *Formatter) LabelList(labels []core.Label, includeIDs bool) string {
	return FormatLabels(labels, LabelListOptions{IncludeIDs: includeIDs})
}

// Labels renders the compact label-list text format.
// Deprecated compatibility entry point; new callers should use LabelList.
func (f *Formatter) Labels(labels []core.Label) string {
	return FormatLabels(labels, LabelListOptions{})
}

func sortLabels(labels []core.Label) []core.Label {
	sort.SliceStable(labels, func(i, j int) bool {
		return labelSortKey(labels[i]) < labelSortKey(labels[j])
	})
	return labels
}

func labelSortKey(label core.Label) string {
	return strings.ToLower(label.Name) + "\x00" + label.Name + "\x00" + label.ID
}

func labelSortKeyRef(ref *core.LabelRef) string {
	if ref == nil {
		return ""
	}
	return strings.ToLower(ref.Name) + "\x00" + ref.Name + "\x00" + ref.ID
}

func groupReference(labels []core.Label, key string) *core.LabelRef {
	var refs []core.LabelRef
	for _, label := range labels {
		if label.Parent == nil {
			continue
		}
		parentKey := label.Parent.ID
		if parentKey == "" {
			parentKey = "name:" + label.Parent.Name
		}
		if parentKey == key {
			refs = append(refs, *label.Parent)
		}
	}
	if len(refs) == 0 {
		return nil
	}
	sort.Slice(refs, func(i, j int) bool { return labelSortKeyRef(&refs[i]) < labelSortKeyRef(&refs[j]) })
	return &refs[0]
}

func labelReferenceName(ref core.LabelRef) string {
	if ref.Name != "" {
		return ref.Name
	}
	return ref.ID
}

func writeLabelIndented(b *strings.Builder, label core.Label, indent string, includeID bool) {
	b.WriteString(indent)
	writeLabel(b, label, includeID)
}

func writeLabel(b *strings.Builder, label core.Label, includeID bool) {
	b.WriteString(label.Name)
	if label.Color != "" {
		b.WriteString(" [")
		b.WriteString(label.Color)
		b.WriteString("]")
	}
	if includeID && label.ID != "" {
		b.WriteByte(' ')
		b.WriteString(label.ID)
	}
	b.WriteByte('\n')
}

func writeDescription(b *strings.Builder, indent, description string) {
	if description != "" {
		b.WriteString(indent)
		b.WriteString(description)
		b.WriteByte('\n')
	}
}
