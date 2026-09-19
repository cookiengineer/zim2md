package adapters

import (
	"testing"
)

const sampleGobyExample = "../../samples/gobyexample.com_en_all_2026-08.zim"

func TestOpenArchiveSourceIndexesEntries(t *testing.T) {
	source, err := OpenArchiveSource(sampleGobyExample)
	if err != nil {
		t.Fatalf("OpenArchiveSource: %v", err)
	}
	defer source.Close()

	if source.EntryCount() == 0 {
		t.Fatalf("no entries indexed")
	}
	if !source.IsNewNamespaceScheme() {
		t.Fatalf("expected new namespace scheme")
	}
	if source.OutputRootName() != "gobyexample.com_en_all_2026-08" {
		t.Fatalf("OutputRootName = %q", source.OutputRootName())
	}

	htmlCount := 0
	redirectCount := 0
	var firstHTML Entry
	for _, entry := range source.Entries() {
		if entry.IsRedirect {
			redirectCount++
			continue
		}
		if entry.IsHTML() {
			if htmlCount == 0 {
				firstHTML = entry
			}
			htmlCount++
		}
	}

	if htmlCount < 50 {
		t.Fatalf("expected many HTML entries, got %d", htmlCount)
	}
	if redirectCount == 0 {
		t.Fatalf("expected at least one redirect")
	}

	data, err := source.ReadBytes(firstHTML)
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("empty HTML payload for %q", firstHTML.Path)
	}
}
