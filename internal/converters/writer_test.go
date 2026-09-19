package converters

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func fragment(t *testing.T, source string) *html.Node {
	t.Helper()
	context := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	parsed, err := html.ParseFragment(strings.NewReader(source), context)
	if err != nil {
		t.Fatalf("parsing fragment: %v", err)
	}
	wrapper := &html.Node{Type: html.ElementNode, Data: "div"}
	for _, node := range parsed {
		wrapper.AppendChild(node)
	}
	return wrapper
}

func convertHTML(t *testing.T, source string, options Options) string {
	t.Helper()
	if options.OutputPath == "" {
		options.OutputPath = "page.md"
	}
	if options.SourceEntryPath == "" {
		options.SourceEntryPath = "page"
	}
	converter := NewConverter(options)
	return converter.ConvertDocument(fragment(t, source), "", false)
}

func TestConvertHeadingsAndInlineMarkup(t *testing.T) {
	markdown := convertHTML(t, `<h1>Title</h1><p>Some <strong>bold</strong> and <em>italic</em> and <code>x &lt; y</code>.</p>`, Options{})

	for _, expected := range []string{"# Title", "**bold**", "*italic*", "`x < y`"} {
		if !strings.Contains(markdown, expected) {
			t.Errorf("markdown missing %q in:\n%s", expected, markdown)
		}
	}
	if strings.Contains(markdown, "<strong>") || strings.Contains(markdown, "<em>") {
		t.Errorf("raw HTML leaked:\n%s", markdown)
	}
}

func TestConvertNestedList(t *testing.T) {
	markdown := convertHTML(t, `<ul><li>one</li><li>two<ul><li>nested</li></ul></li></ul>`, Options{})

	for _, expected := range []string{"- one", "- two", "  - nested"} {
		if !strings.Contains(markdown, expected) {
			t.Errorf("markdown missing %q in:\n%s", expected, markdown)
		}
	}
}

func TestConvertOrderedListWithStart(t *testing.T) {
	markdown := convertHTML(t, `<ol start="3"><li>alpha</li><li>beta</li></ol>`, Options{})

	if !strings.Contains(markdown, "3. alpha") || !strings.Contains(markdown, "4. beta") {
		t.Fatalf("unexpected ordered list:\n%s", markdown)
	}
}

func TestConvertBlockquote(t *testing.T) {
	markdown := convertHTML(t, `<blockquote><p>quoted line</p></blockquote>`, Options{})

	if !strings.Contains(markdown, "> quoted line") {
		t.Fatalf("unexpected blockquote:\n%s", markdown)
	}
}

func TestConvertDataTable(t *testing.T) {
	markdown := convertHTML(t, `<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>1</td><td>2</td></tr></tbody></table>`, Options{})

	expected := "| A | B |\n| --- | --- |\n| 1 | 2 |"
	if !strings.Contains(markdown, expected) {
		t.Fatalf("unexpected table:\n%s\nwant contains:\n%s", markdown, expected)
	}
}

func TestConvertLayoutTable(t *testing.T) {
	markdown := convertHTML(t, `<table><tr><td><p>docs text</p></td><td><pre>code block</pre></td></tr></table>`, Options{})

	if strings.Contains(markdown, "| --- |") {
		t.Fatalf("layout table must not become a GFM table:\n%s", markdown)
	}
	for _, expected := range []string{"docs text", "```", "code block"} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("layout table missing %q in:\n%s", expected, markdown)
		}
	}
}

func TestRewriteInternalLinks(t *testing.T) {
	markdown := convertHTML(
		t,
		`<p><a href="target#section">go</a> <a href="https://example.com">ext</a> <a href="#here">anchor</a> <a href="javascript:alert(1)">bad</a></p>`,
		Options{
			OutputPath:      "root/current.md",
			SourceEntryPath: "current",
			HTMLIndex:       map[string]string{"target": "root/target.md"},
			LinkRewrite:     LinkRewriteIndex,
		},
	)

	for _, expected := range []string{"[go](target.md#section)", "[ext](https://example.com)", "[anchor](#here)"} {
		if !strings.Contains(markdown, expected) {
			t.Errorf("markdown missing %q in:\n%s", expected, markdown)
		}
	}
	if strings.Contains(markdown, "javascript:") {
		t.Fatalf("unsafe URL leaked:\n%s", markdown)
	}
	if !strings.Contains(markdown, "bad") {
		t.Fatalf("unsafe link should fall back to its text:\n%s", markdown)
	}
}

func TestSkipIconImages(t *testing.T) {
	markdown := convertHTML(t, `<p><img src="play.png" class="run" alt=""> visible text</p>`, Options{})

	if strings.Contains(markdown, "![") {
		t.Fatalf("icon image should be skipped:\n%s", markdown)
	}
	if !strings.Contains(markdown, "visible text") {
		t.Fatalf("text lost:\n%s", markdown)
	}
}

func TestEscapeLeadingBlockMarker(t *testing.T) {
	markdown := convertHTML(t, `<p>- not a list</p>`, Options{})

	if !strings.Contains(markdown, `\- not a list`) {
		t.Fatalf("leading dash not escaped:\n%s", markdown)
	}
}

func TestCodeLanguageInheritance(t *testing.T) {
	markdown := convertHTML(t, `<pre>package main</pre><pre>fmt.Println("hi")</pre>`, Options{})

	if strings.Count(markdown, "```go") != 2 {
		t.Fatalf("expected both fragments to inherit the go language:\n%s", markdown)
	}
}

func TestConvertDocumentPrependsTitleOnlyWithoutH1(t *testing.T) {
	withH1 := NewConverter(Options{OutputPath: "page.md", SourceEntryPath: "page"})
	if got := withH1.ConvertDocument(fragment(t, `<h1>Own title</h1><p>body</p>`), "Meta title", true); strings.Contains(got, "# Meta title") {
		t.Fatalf("title must not be prepended when an H1 exists:\n%s", got)
	}

	withoutH1 := NewConverter(Options{OutputPath: "page.md", SourceEntryPath: "page"})
	if got := withoutH1.ConvertDocument(fragment(t, `<h2>Sub</h2><p>body</p>`), "Meta title", false); !strings.Contains(got, "# Meta title") {
		t.Fatalf("title must be prepended when no H1 exists:\n%s", got)
	}
}
