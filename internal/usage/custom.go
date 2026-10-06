package usage

import (
	"strings"

	"github.com/spf13/cobra"
)

// RegisterCustomCommand makes a hand-written command visible to --usage.
//
// The generated usageSchemas only know about commands derived from the
// OpenAPI document, so a custom command would otherwise be missing from
// `shade --usage` and `shade <group> --usage` even though it exists. This
// renders the command live and splices it into its parent's static entries
// (including alias keys) and into the parent's block in the root entry.
// Call it after cmd has been added to its parent. Group init functions run
// before the group itself is attached to the root, so the parent's path from
// the root (e.g. "ssh-keys") is passed explicitly. Idempotent.
func RegisterCustomCommand(cmd *cobra.Command, parentPath ...string) {
	MarkDynamic(cmd)
	parent := cmd.Parent()
	if parent == nil || len(parentPath) == 0 {
		return
	}
	var buf strings.Builder
	if err := emitLiveSchema(cmd, &buf); err != nil {
		return
	}
	child := buf.String()
	opener := "cmd " + kdlQuote(cmd.Name()) + " "

	keys := []string{strings.Join(parentPath, " ")}
	for _, alias := range parent.Aliases {
		aliased := append(append([]string{}, parentPath[:len(parentPath)-1]...), alias)
		keys = append(keys, strings.Join(aliased, " "))
	}
	for _, key := range keys {
		if schema, ok := usageSchemas[key]; ok {
			usageSchemas[key] = spliceChild(schema, child, opener, "  ")
		}
	}

	if root, ok := usageSchemas[""]; ok {
		usageSchemas[""] = spliceIntoRoot(root, parent.Name(), child, opener, strings.Repeat("  ", len(parentPath)))
	}
}

// spliceChild inserts child (indented) before the closing brace of a single
// command entry, or opens a block if the entry has none.
func spliceChild(schema, child, opener, indent string) string {
	if strings.Contains(schema, "\n"+indent+opener) {
		return schema
	}
	block := indentLines(child, indent)
	if i := strings.LastIndex(schema, "\n}"); i >= 0 {
		return schema[:i+1] + block + schema[i+1:]
	}
	return strings.TrimRight(schema, "\n") + " {\n" + block + "}\n"
}

// spliceIntoRoot finds the parent's block in the root schema and inserts child
// before that block's indent-0 closing brace.
func spliceIntoRoot(root, parentName, child, opener, indent string) string {
	start := strings.Index(root, "\ncmd "+kdlQuote(parentName)+" ")
	if start < 0 {
		return root
	}
	end := strings.Index(root[start+1:], "\n}\n")
	if end < 0 {
		return root
	}
	end += start + 1
	block := root[start:end]
	if strings.Contains(block, "\n"+indent+opener) {
		return root
	}
	return root[:end+1] + indentLines(child, indent) + root[end+1:]
}

func indentLines(s, indent string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
