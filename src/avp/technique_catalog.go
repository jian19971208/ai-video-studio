package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

// Catalog discovery is offline and does not assert any provider is runnable.
func techniqueCatalogCommand(cmd string, args []string) M {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	query := f.String("query", "", "filter by technique name, purpose or scope")
	id := f.String("id", "", "technique ID")
	check(f.Parse(args))
	root := filepath.Join(skillRoot(), "references", "techniques")
	items, ok := read(filepath.Join(root, "catalog.json")).([]any)
	require(ok, "technique catalog must be an array")
	matched := []M{}
	for _, raw := range items {
		item := object(raw)
		if *id != "" && item["id"] != *id {
			continue
		}
		if *query != "" && !strings.Contains(strings.ToLower(text(item["title"])+" "+text(item["summary"])+" "+text(item["scope"])), strings.ToLower(*query)) {
			continue
		}
		copy := object(decode(encoded(item)))
		copy["availability"] = "requires-current-provider-check"
		copy["guide_file"] = filepath.Join(root, filepath.FromSlash(text(item["guide"])))
		matched = append(matched, copy)
	}
	if cmd == "technique-describe" {
		require(*id != "" && len(matched) == 1, "provide an existing --id from technique-list")
		body, err := os.ReadFile(text(matched[0]["guide_file"]))
		check(err)
		matched[0]["instructions"] = string(body)
		return M{"technique": matched[0], "read_only": true, "runtime_verified": false}
	}
	return M{"techniques": matched, "count": len(matched), "read_only": true, "runtime_verified": false, "scope": "documented methods, not a capability whitelist; check actual providers before use"}
}
