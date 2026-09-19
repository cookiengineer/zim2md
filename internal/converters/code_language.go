package converters

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

var languageAttributeNames = []string{
	"data-language",
	"data-lang",
	"data-syntax",
	"data-highlight-language",
	"data-code-language",
	"lang",
}

var (
	markupTagPattern    = regexp.MustCompile(`(?i)<[a-z][a-z0-9]*(\s|>|/)`)
	goPackagePattern    = regexp.MustCompile(`(?m)^\s*package\s+[a-zA-Z_][a-zA-Z0-9_]*`)
	goFuncPattern       = regexp.MustCompile(`(?m)^\s*func\s+`)
	shellPromptPattern  = regexp.MustCompile(`(?m)^\s*[$#]\s+\S`)
	shellCommandPattern = regexp.MustCompile(`(?m)^\s*(sudo|pacman|systemctl|apt|apt-get|yum|dnf|brew|git|make|cmake|\./configure|export|cd|ls|cp|mv|rm|mkdir|chmod|chown|mount|umount|tar|curl|wget)\b`)
	pythonPattern       = regexp.MustCompile(`(?m)^\s*(def|class)\s+\w+.*:\s*$|^\s*(import|from)\s+\w+`)
	cIncludePattern     = regexp.MustCompile(`(?m)^\s*#include\s*[<"]`)
	rustPattern         = regexp.MustCompile(`(?m)^\s*(fn\s+main|let\s+mut\s|use\s+std::|impl\s+\w+|pub\s+fn\s+)`)
	sqlPattern          = regexp.MustCompile(`(?i)\b(SELECT\s+.+\s+FROM|INSERT\s+INTO|CREATE\s+TABLE|UPDATE\s+\w+\s+SET|DELETE\s+FROM)\b`)
	diffPattern         = regexp.MustCompile(`(?m)^(@@ .*@@|\+\+\+ |--- )`)
	yamlLinePattern     = regexp.MustCompile(`^[A-Za-z0-9_.-]+\s*:\s*.*$`)
)

var languageAliases = map[string]string{
	"golang":     "go",
	"py":         "python",
	"python3":    "python",
	"js":         "javascript",
	"ts":         "typescript",
	"sh":         "bash",
	"shell":      "bash",
	"zsh":        "bash",
	"console":    "bash",
	"ps1":        "powershell",
	"c++":        "cpp",
	"cxx":        "cpp",
	"htm":        "html",
	"xhtml":      "html",
	"yml":        "yaml",
	"rb":         "ruby",
	"rs":         "rust",
	"md":         "markdown",
	"cs":         "csharp",
	"c#":         "csharp",
	"dockerfile": "dockerfile",
	"makefile":   "makefile",
}

// renderCodeBlock turns a <pre> element into a fenced code block.
func (c *Converter) renderCodeBlock(pre *html.Node) string {
	code := strings.Trim(extractCodeText(pre), "\n")
	code = strings.TrimRight(code, " \t")
	if strings.TrimSpace(code) == "" {
		return ""
	}

	language := c.detectCodeLanguage(pre, code)
	fence := fenceForCode(code)

	var builder strings.Builder
	builder.WriteString(fence)
	builder.WriteString(language)
	builder.WriteByte('\n')
	builder.WriteString(code)
	if !strings.HasSuffix(code, "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString(fence)
	return builder.String()
}

// extractCodeText returns the literal code, handling chroma-style line spans.
func extractCodeText(pre *html.Node) string {
	lineSpans := findLineSpans(pre)
	if len(lineSpans) > 0 {
		lines := make([]string, 0, len(lineSpans))
		for _, span := range lineSpans {
			lines = append(lines, strings.TrimRight(nodes.TextContent(span), "\n"))
		}
		return strings.Join(lines, "\n")
	}

	var builder strings.Builder
	collectPreformattedText(pre, &builder)
	return builder.String()
}

func findLineSpans(pre *html.Node) []*html.Node {
	spans := make([]*html.Node, 0, 8)
	nodes.Walk(pre, func(node *html.Node) bool {
		if node.Type == elementNode && nodes.HasClass(node, "line") {
			spans = append(spans, node)
			return false
		}
		return true
	})
	return spans
}

func collectPreformattedText(node *html.Node, builder *strings.Builder) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case textNode:
			builder.WriteString(child.Data)
		case elementNode:
			if child.Data == "br" {
				builder.WriteByte('\n')
				continue
			}
			collectPreformattedText(child, builder)
		}
	}
}

// detectCodeLanguage resolves a fenced-code language from attributes, class
// hints, filename hints and finally content heuristics.
func (c *Converter) detectCodeLanguage(pre *html.Node, code string) string {
	inspected := []*html.Node{pre}
	if codeNode := findCodeDescendant(pre); codeNode != nil && codeNode != pre {
		inspected = append(inspected, codeNode)
	}

	for _, node := range inspected {
		if isProgramOutput(node) {
			return "text"
		}
	}

	for _, node := range inspected {
		for _, attribute := range languageAttributeNames {
			if value := nodes.Attribute(node, attribute); value != "" {
				if language := canonicalizeLanguage(value); language != "" {
					return c.rememberLanguage(language)
				}
			}
		}
	}

	for _, node := range inspected {
		classes := nodes.ClassList(node)
		for index, class := range classes {
			if language := languageHintFromClass(class, classes, index); language != "" {
				return c.rememberLanguage(language)
			}
		}
	}

	if language := filenameLanguageHint(pre); language != "" {
		return c.rememberLanguage(language)
	}
	if guessed := guessLanguageFromCode(code); guessed != "" {
		return c.rememberLanguage(guessed)
	}
	if c.documentLanguageSet {
		return c.documentLanguage
	}
	return c.options.DefaultCodeLanguage
}

// rememberLanguage stores the first confidently detected language of a document
// so that undecorated fragments inherit it.
func (c *Converter) rememberLanguage(language string) string {
	if !c.documentLanguageSet && language != "" && language != "text" {
		c.documentLanguage = language
		c.documentLanguageSet = true
	}
	return language
}

func findCodeDescendant(pre *html.Node) *html.Node {
	if nodes.IsElement(pre, "code") {
		return pre
	}
	var found *html.Node
	nodes.Walk(pre, func(node *html.Node) bool {
		if nodes.IsElement(node, "code") {
			found = node
			return false
		}
		return true
	})
	return found
}

func isProgramOutput(node *html.Node) bool {
	for _, class := range nodes.ClassList(node) {
		switch strings.ToLower(class) {
		case "output", "program-output", "command-output", "terminal-output", "shell-session", "bash-session":
			return true
		}
	}
	return false
}

func languageHintFromClass(class string, classes []string, index int) string {
	token := strings.ToLower(class)

	for _, prefix := range []string{"language-", "lang-", "highlight-source-", "source-"} {
		if strings.HasPrefix(token, prefix) {
			return strings.TrimPrefix(token, prefix)
		}
	}

	if token == "brush:" && index+1 < len(classes) {
		return classes[index+1]
	}

	return ""
}

func filenameLanguageHint(pre *html.Node) string {
	var names []string
	for _, attribute := range []string{"data-filename", "data-file", "filename"} {
		if value := nodes.Attribute(pre, attribute); value != "" {
			names = append(names, value)
		}
	}
	nodes.Walk(pre, func(node *html.Node) bool {
		if node.Type == elementNode && (nodes.HasClass(node, "filename") || nodes.HasClass(node, "code-caption") || node.Data == "figcaption") {
			if text := strings.TrimSpace(nodes.TextContent(node)); text != "" {
				names = append(names, text)
			}
		}
		return true
	})

	for _, name := range names {
		extension := strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(name)), "."))
		if language := canonicalizeLanguage(extension); language != "" {
			return language
		}
	}
	return ""
}

func canonicalizeLanguage(alias string) string {
	normalized := strings.ToLower(strings.TrimSpace(alias))
	normalized = strings.TrimPrefix(normalized, ".")
	if normalized == "" {
		return ""
	}
	if mapped, ok := languageAliases[normalized]; ok {
		return mapped
	}
	for _, character := range normalized {
		valid := (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '+' || character == '#'
		if !valid {
			return ""
		}
	}
	return normalized
}

func guessLanguageFromCode(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "<?xml"):
		return "xml"
	case strings.HasPrefix(lower, "<!doctype html"), strings.HasPrefix(lower, "<html"):
		return "html"
	case strings.HasPrefix(trimmed, "{"), strings.HasPrefix(trimmed, "["):
		if json.Valid([]byte(trimmed)) {
			return "json"
		}
	}

	switch {
	case diffPattern.MatchString(code):
		return "diff"
	case shellPromptPattern.MatchString(code), shellCommandPattern.MatchString(code):
		return "bash"
	case goPackagePattern.MatchString(code), goFuncPattern.MatchString(code):
		return "go"
	case cIncludePattern.MatchString(code):
		return "c"
	case rustPattern.MatchString(code):
		return "rust"
	case pythonPattern.MatchString(code):
		return "python"
	case sqlPattern.MatchString(code):
		return "sql"
	case strings.Contains(code, ":="):
		return "go"
	case len(markupTagPattern.FindAllString(code, 3)) >= 2:
		return "html"
	}

	if looksLikeYAML(code) {
		return "yaml"
	}

	return ""
}

// looksLikeYAML is deliberately conservative so that Go map/slice fragments
// (which contain ":=" or braces) are not misdetected.
func looksLikeYAML(code string) bool {
	if strings.Contains(code, ":=") || strings.Contains(code, "{") ||
		strings.Contains(code, ";") || strings.Contains(code, "()") {
		return false
	}

	keyLines := 0
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "- ") {
			continue
		}
		if yamlLinePattern.MatchString(trimmed) {
			keyLines++
			continue
		}
		return false
	}
	return keyLines >= 2
}

func fenceForCode(code string) string {
	longest, current := 0, 0
	for _, character := range code {
		if character == '`' {
			current++
			if current > longest {
				longest = current
			}
			continue
		}
		current = 0
	}
	length := longest + 1
	if length < 3 {
		length = 3
	}
	return strings.Repeat("`", length)
}
