// Package mappers translates OpenZIM URL paths into filesystem-safe Markdown
// output paths and computes the relative links between them. Only HTML entries
// are mapped to ".md"; assets keep their original names.
package mappers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxPathSegmentBytes = 200
	markdownExtension   = ".md"
	markdownIndexName   = "index.md"
	collisionHashBytes  = 4
)

// MapHTMLPathToMarkdown maps an HTML entry path to its Markdown output path,
// relative to the archive output root. The newNamespaceScheme flag controls
// whether an old-style single-letter namespace prefix is stripped.
func MapHTMLPathToMarkdown(sourcePath string, newNamespaceScheme bool) string {
	cleaned := strings.ReplaceAll(sourcePath, "\\", "/")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "/")
	cleaned = StripNamespacePrefix(cleaned, newNamespaceScheme)

	if cleaned == "" || cleaned == "." {
		return markdownIndexName
	}

	if strings.HasSuffix(sourcePath, "/") {
		return SanitizeRelativePath(cleaned) + "/" + markdownIndexName
	}

	segments := strings.Split(cleaned, "/")
	segments[len(segments)-1] = ReplaceFinalExtensionWithMarkdown(segments[len(segments)-1])
	return SanitizeRelativePath(strings.Join(segments, "/"))
}

// ReplaceFinalExtensionWithMarkdown replaces the last path extension of a
// single name with ".md". A leading-dot name without another dot is treated as
// extensionless (".NET" -> ".NET.md"). A name without an extension gets ".md".
func ReplaceFinalExtensionWithMarkdown(name string) string {
	if name == "" || name == "." || name == ".." {
		return markdownIndexName
	}

	if strings.HasPrefix(name, ".") {
		if rest := name[1:]; !strings.Contains(rest, ".") {
			return name + markdownExtension
		}
	}

	extension := path.Ext(name)
	if extension == "" {
		return name + markdownExtension
	}
	return name[:len(name)-len(extension)] + markdownExtension
}

// SanitizeRelativePath cleans every segment of a slash separated relative path
// so it can never escape the output root and never contains characters that
// are illegal on common filesystems.
func SanitizeRelativePath(relativePath string) string {
	segments := strings.Split(relativePath, "/")
	sanitized := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		sanitized = append(sanitized, sanitizePathSegment(segment))
	}
	return strings.Join(sanitized, "/")
}

// RelativeMarkdownLink computes a URL-safe relative link from one output file
// to another, preserving an optional fragment.
func RelativeMarkdownLink(fromOutputPath, toOutputPath, fragment string) string {
	fromDirectory := path.Dir(fromOutputPath)
	relative, err := filepath.Rel(filepath.FromSlash(fromDirectory), filepath.FromSlash(toOutputPath))
	if err != nil {
		relative = toOutputPath
	}
	relative = filepath.ToSlash(relative)

	if relative == "" {
		if fragment == "" {
			return "#"
		}
		return "#" + fragment
	}

	escaped := (&url.URL{Path: relative}).EscapedPath()
	if fragment != "" {
		return escaped + "#" + fragment
	}
	return escaped
}

// OutputPathAllocator assigns unique output paths and reports collisions.
type OutputPathAllocator struct {
	used map[string]string
}

// NewOutputPathAllocator returns an empty allocator.
func NewOutputPathAllocator() *OutputPathAllocator {
	return &OutputPathAllocator{used: make(map[string]string)}
}

// Allocate reserves desiredOutputPath for sourcePath. When the path is already
// taken a deterministic suffixed variant is returned and collision is true.
func (a *OutputPathAllocator) Allocate(sourcePath, desiredOutputPath string) (outputPath string, collision bool) {
	if previous, exists := a.used[desiredOutputPath]; exists {
		collision = true
		outputPath = addCollisionSuffix(desiredOutputPath, sourcePath)
		for {
			if _, taken := a.used[outputPath]; !taken {
				break
			}
			outputPath = addCollisionSuffix(desiredOutputPath, sourcePath+"#"+previous)
		}
	} else {
		outputPath = desiredOutputPath
	}

	a.used[outputPath] = sourcePath
	return outputPath, collision
}

// StripNamespacePrefix removes an old-style single-letter namespace prefix
// ("A/", "I/", "J/", "-/") when the archive does not use the new scheme.
func StripNamespacePrefix(sourcePath string, newNamespaceScheme bool) string {
	if newNamespaceScheme {
		return sourcePath
	}
	if len(sourcePath) >= 2 && sourcePath[1] == '/' && strings.ContainsRune("AIJ-C", rune(sourcePath[0])) {
		return sourcePath[2:]
	}
	return sourcePath
}

func sanitizePathSegment(segment string) string {
	if segment == "." || segment == ".." {
		return "_"
	}

	var builder strings.Builder
	for _, character := range segment {
		switch {
		case character == 0, character < 0x20, character == 0x7f:
			builder.WriteByte('_')
		case strings.ContainsRune(`<>:"|?*\`, character):
			builder.WriteByte('_')
		default:
			builder.WriteRune(character)
		}
	}

	cleaned := strings.TrimRight(builder.String(), ". ")
	if cleaned == "" {
		cleaned = "_"
	}

	if len(cleaned) > maxPathSegmentBytes {
		digest := sha256.Sum256([]byte(segment))
		suffix := hex.EncodeToString(digest[:collisionHashBytes])
		limit := maxPathSegmentBytes - len(suffix) - 1
		cleaned = truncateUTF8(cleaned, limit) + "-" + suffix
	}

	return cleaned
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	truncated := value[:limit]
	for len(truncated) > 0 {
		character, size := utf8.DecodeLastRuneInString(truncated)
		if character != utf8.RuneError || size > 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func addCollisionSuffix(outputPath, sourcePath string) string {
	digest := sha256.Sum256([]byte(sourcePath))
	suffix := hex.EncodeToString(digest[:collisionHashBytes])
	extension := path.Ext(outputPath)
	stem := strings.TrimSuffix(outputPath, extension)
	return fmt.Sprintf("%s-%s%s", stem, suffix, extension)
}
