// Keep the newest three versions per chart in a Helm-generated, sorted index.
// Leave published packages and extension bundles available to existing installs.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/prune-helm-index.go INDEX_PATH")
		os.Exit(1)
	}
	if err := prune(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prune(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var index map[string]json.RawMessage
	if err := yaml.Unmarshal(content, &index); err != nil {
		return err
	}
	var entries map[string][]json.RawMessage
	if err := json.Unmarshal(index["entries"], &entries); err != nil {
		return err
	}
	for chart, versions := range entries {
		// helm repo index sorts versions newest first using semantic versioning.
		if len(versions) > 3 {
			entries[chart] = versions[:3]
		}
		fmt.Printf("%s: keeping %d of %d versions\n", chart, len(entries[chart]), len(versions))
	}
	index["entries"], err = json.Marshal(entries)
	if err != nil {
		return err
	}
	content, err = yaml.Marshal(index)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
