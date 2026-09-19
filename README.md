# zim2md

`zim2md` is a standalone command line tool that converts the HTML pages inside
[OpenZIM](https://openzim.org/) archives (`.zim`) into clean, Reader-Mode-style
Markdown files.

It strips away navigation, sidebars, menus, scripts, advertisements and other
page chrome, then writes one Markdown file per HTML page using the same
directory layout as the archive. Only entries whose MIME type is `text/html`
(or `application/xhtml+xml`) are converted; all other entries (images, CSS,
JavaScript, fonts, …) keep their original paths and are left untouched unless
`--assets` is enabled.

```
example-123.zim + "path/to/page.html"  ->  example-123/path/to/page.md
example-123.zim + "path/to/index"      ->  example-123/path/to/index.md
example-123.zim + "dir/"               ->  example-123/dir/index.md
```

## Features

- Reads the OpenZIM index and converts every HTML page in the archive.
- Reader-Mode content extraction with a deterministic, failsafe scoring
  algorithm: navigation, sidebars, footers, menus, forms, scripts and styles
  are removed; if no obvious content container is found it falls back to the
  page body rather than producing nothing.
- Sanitized Markdown output: text is escaped so no raw HTML is ever emitted,
  except inside a fenced code block tagged `html`.
- Code-block language detection from `data-language`, `lang`, `class`
  (`language-*`, `lang-*`, `highlight-source-*`, `brush:`) and filename hints,
  with content heuristics for Go, HTML/XML, JSON, shell, Python, C, Rust, SQL,
  YAML and diffs. Undecorated fragments inherit the first detected language of
  the page (handy for annotated examples split over many `<pre>` blocks).
- Internal links are rewritten so they point at the exported `.md` files;
  `javascript:`, `vbscript:` and `data:` URLs are dropped.
- GFM tables for data tables; layout tables are unwrapped in reading order.
- Redirect entries are skipped and listed in a report.
- Optional asset export (`--assets`).
- Parallel conversion using a bounded worker pool. The ZIM index is read once
  on the main goroutine; decompression, conversion and file writes run
  concurrently.

## Requirements

- Go 1.27 or newer to build.
- No runtime dependencies.

`zim2md` is implemented with the Go standard library plus
`golang.org/x/net/html` for HTML parsing. ZIM access uses
[`github.com/cookiengineer/gozim`](https://github.com/cookiengineer/gozim).

## Install

```bash
go install github.com/cookiengineer/zim2md@latest
```

Or build from a checkout:

```bash
git clone https://github.com/cookiengineer/zim2md
cd zim2md
go build -o zim2md .
```

## Usage

```text
zim2md [options] <archive.zim> [more.zim ...]
```

Run `zim2md --help` (or just `zim2md`) for the full overview.

### Examples

```bash
# Convert one archive into ./example-123/
zim2md example-123.zim

# Convert several archives into ./markdown/ using 8 workers
zim2md --output ./markdown --workers 8 enwiki.zim dewiki.zim

# Also export referenced images and other non-HTML assets
zim2md --assets --output ./markdown example-123.zim

# Convert only a subset of paths
zim2md --include '^Go_' --exclude '_talk$' enwiki.zim

# Preview without writing files, print the report to stdout
zim2md --dry-run --report - example-123.zim

# Set a fallback code language and never overwrite existing files
zim2md --default-code-language text --no-clobber example-123.zim
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output DIR` | `.` | Output root directory |
| `--assets` | `false` | Export referenced non-HTML assets as well |
| `--assets-max-size N` | `52428800` | Maximum asset size in bytes |
| `--workers N` | CPU count | Number of parallel conversion workers |
| `--include REGEX` | – | Only convert entries matching this regexp |
| `--exclude REGEX` | – | Skip entries matching this regexp |
| `--link-rewrite MODE` | `index` | Internal link mode: `index`, `suffix`, `off` |
| `--default-code-language LANG` | – | Fallback fenced-code language |
| `--title-heading MODE` | `auto` | `auto`, `always`, `never` |
| `--no-clobber` | `false` | Do not overwrite existing files |
| `--report FILE` | `<output>/<basename>.zim2md.report.txt` | Report path (`-` = stdout) |
| `--dry-run` | `false` | Do not write any files |
| `--verbose` | `false` | Verbose progress output |
| `--quiet` | `false` | Suppress the summary |
| `--version` | – | Print version and exit |

## Output layout

For an archive named `example-123.zim` the output is written below
`<output>/example-123/`. The final extension of each HTML path is replaced with
`.md`; a path ending in `/` becomes `index.md`; a dotfile such as `.NET` becomes
`.NET.md`.

| Source entry path | Output relative path |
|-------------------|----------------------|
| `path/to/page.html` | `path/to/page.md` |
| `path/to/page.htm` | `path/to/page.md` |
| `path/to/index` | `path/to/index.md` |
| `path/to/index.php` | `path/to/index.md` |
| `dir/` | `dir/index.md` |
| `/` | `index.md` |
| `.NET` | `.NET.md` |

Path segments are sanitized (control characters and `<>:"|?*\` are replaced
with `_`, `.`/`..` are neutralized, long segments are truncated with a hash
suffix). Mapping can never escape the output root. If two HTML entries would
collide, the later one gets a short content hash suffix and the collision is
listed in the report.

Non-HTML entries are **never** renamed to `.md`. For example, a page named
`Node.js` becomes `Node.md` because it is HTML, while a `node.js` script asset
stays `node.js` and is only copied when `--assets` is given.

## Report

Each run writes a report (by default next to the archive output directory) with
counts, skipped redirects, conversion errors, path collisions and skipped
assets:

```text
zim2md report
=============
archive:            samples/example.zim
output:             ./example
total entries:      95
html pages found:   87
markdown written:   87
assets exported:    0
assets skipped:     7
redirects skipped:  1
conversion errors:  0
path collisions:    0

redirects:
  mainPage -> example/
```

Exit codes: `0` success, `1` fatal (for example an unreadable archive), `2`
finished with per-page conversion errors.

## How it works

The conversion pipeline is split into small, single-purpose packages:

| Package | Responsibility |
|---------|----------------|
| `internal/adapters` | The only package that talks to `gozim`: open the archive, classify the index, read entry bytes, resolve redirects |
| `internal/mappers` | HTML path → Markdown path mapping, segment sanitization, collision-free allocation, relative links |
| `internal/parsers` | HTML parsing, charset handling, Reader-Mode content scoring, sanitization |
| `internal/converters` | DOM → Markdown writer, element mapping, code-language detection, link/image rewriting |
| `internal/exporters` | Atomic file writes and the run report |
| `internal/nodes` | Shared helpers for working with `x/net/html` node trees |
| `internal/cli` | Flag parsing, orchestration, worker pool, summary and exit codes |

Reader-Mode extraction works in three steps:

1. **Remove** structural noise: `script`, `style`, `noscript`, `iframe`, forms,
   hidden elements, ARIA navigation roles and elements whose class or id
   contains tokens such as `nav`, `sidebar`, `menu`, `footer`, `breadcrumb`,
   `pagination`, `related`, `mw-editsection`, `catlinks`, …
2. **Score** candidate containers from paragraph-like elements, class/id
   weights and link density, then pick the best candidate (falling back to
   `<body>`).
3. **Clean** the result: drop empty wrappers, tracking images and empty links.

The sanitizing writer escapes Markdown metacharacters and `<`/`>` in text, so
untrusted page content cannot inject HTML into the output. Raw markup is only
preserved inside fenced code blocks whose detected language is `html`/`xml`.

## Development

```bash
# Format, vet, build
gofmt -l .
go vet ./...
go build ./...

# Unit and end-to-end tests
go test ./...

# Skip the large archlinux archive (fast feedback)
go test -short ./...

# Race detector
go test -race ./...
```

The end-to-end tests use the sample archives in `./samples`:

- `gobyexample.com_en_all_2026-08.zim` — layout tables and chroma code blocks.
- `devdocs_en_go_2026-07.zim` — `data-language` code blocks and navigation.
- `archlinux_en_all_maxi_2026-07.zim` — MediaWiki markup, tables and redirects.

On a 16-core machine, converting all 6748 HTML pages of the archlinux sample
takes roughly 2.5 seconds and reports zero conversion errors.

`implementation_plan.md` contains the task checklist and verification log.
