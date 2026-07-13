// Template: commands/write.go — copy into a generated CLI (GOAL.md §3d).
// Universal write flags for create/update, so a writable resource needs NO per-resource flag code:
// the user supplies the exact documented attributes, so we never fabricate or hardcode field names.
//
//	--data/-d  attributes as a JSON object; @file reads from a file; - reads from stdin
//	--set      attribute key=value, repeatable (value is JSON-parsed, else a string)
//
// writeAttrs returns a map[string]any of attributes. Wrap it however your API expects at the call
// site — send it as the request body directly (most REST APIs), or wrap in an envelope. For a
// JSON:API service, wrap as {"data":{"type":<t>,"id":<id>,"attributes":attrs,"relationships":rels}}
// and also register the --rel flag (see the commented block at the bottom).
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// writeAttrs registers --data/--set on cmd and returns a builder that assembles the attributes.
func writeAttrs(cmd *cobra.Command) func() (map[string]any, error) {
	var dataStr string
	var sets []string
	cmd.Flags().StringVarP(&dataStr, "data", "d", "", "attributes as a JSON object; @file or - for stdin")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "attribute key=value (repeatable)")

	return func() (map[string]any, error) {
		attrs := map[string]any{}
		if dataStr != "" {
			raw := dataStr
			switch {
			case dataStr == "-":
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("read --data from stdin: %w", err)
				}
				raw = string(b)
			case strings.HasPrefix(dataStr, "@"):
				// User-named file, like `curl -d @file`; comes from the invocation, not API data,
				// so it's not subject to any data-path-confinement rule.
				b, err := os.ReadFile(dataStr[1:]) // #nosec G304 -- explicit user-supplied data file
				if err != nil {
					return nil, fmt.Errorf("read --data file: %w", err)
				}
				raw = string(b)
			}
			if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
				return nil, fmt.Errorf("parse --data JSON: %w", err)
			}
		}
		for _, s := range sets {
			k, v, ok := strings.Cut(s, "=")
			if !ok {
				return nil, fmt.Errorf("invalid --set %q (want key=value)", s)
			}
			attrs[strings.TrimSpace(k)] = coerceValue(v)
		}
		return attrs, nil
	}
}

// coerceValue parses a --set value as JSON so numbers/bools/null/arrays/objects round-trip with
// their real type, falling back to the raw string when it isn't valid JSON.
func coerceValue(v string) any {
	var out any
	if json.Unmarshal([]byte(v), &out) == nil {
		return out
	}
	return v
}

// JSON:API adaptation — if your API keys writes by {type, id, relationships}, add this alongside
// the flags above and thread it into the envelope:
//
//	var rels []string
//	cmd.Flags().StringArrayVar(&rels, "rel", nil, "relationship name=type:id (repeatable), e.g. store=stores:1")
//	// in the builder:
//	rel := map[string]any{}
//	for _, rdef := range rels {
//		name, spec, ok := strings.Cut(rdef, "=")
//		typ, id, ok2 := strings.Cut(spec, ":")
//		if !ok || !ok2 || typ == "" || id == "" {
//			return nil, fmt.Errorf("invalid --rel %q (want name=type:id)", rdef)
//		}
//		rel[strings.TrimSpace(name)] = map[string]any{"data": map[string]any{"type": typ, "id": id}}
//	}
