# Contributing to notion-getpage

[Leia em português (Brasil)](CONTRIBUTING.pt-BR.md)

Thanks for helping improve `notion-getpage`. The CLI reads Notion pages available to the signed-in user and returns Markdown for AI agents. Start with the [README](README.md) for usage and the [PRD](PRD.md) for requirements and known limits.

## Set up locally

Use Go 1.22 or newer. The current automatic cleanup of session and cache data requires Linux with a private `XDG_RUNTIME_DIR`. The CLI can download Chrome for Testing on first use; set `NOTION_GETPAGE_CHROME_PATH` if you want to use an existing Chrome installation.

```sh
go build -buildvcs=false -o bin/notion-getpage ./cmd/notion-getpage
go test ./...
go vet ./...
```

On Linux, sign in with `./bin/notion-getpage login` to test private pages. You need to sign in again after a reboot or a full logout.

## Find the relevant code

| Path | Responsibility |
| --- | --- |
| `cmd/notion-getpage/main.go` | CLI options, output, errors, and cache flow |
| `internal/core/` | URL validation, cache, runtime paths, and legacy data cleanup |
| `internal/browser/browser.go` | Browser session and page loading |
| `internal/browser/managed.go` | Managed Chrome download and update |
| `internal/browser/extract.js` | Toggle expansion and Markdown extraction |

## Make and check a change

1. Describe the problem and the expected Markdown or CLI behavior. For extraction changes, include the Notion block types involved.
2. Keep Markdown on `stdout` and diagnostics on `stderr`. Preserve exit codes and report incomplete conversions as warnings.
3. Add a focused test when it can verify behavior independently of the implementation. Update the README or PRD when user-facing behavior or a requirement changes.
4. Format changed Go files with `gofmt`, then run `go test ./...`, `go vet ./...`, and the build command above.
5. If the change affects page extraction, run a manual check with `--refresh` so the cache cannot hide the result. Compare the generated Markdown with the page, including collapsed blocks and nested content.

Example manual check with a page you may access:

```sh
./bin/notion-getpage 'https://www.notion.so/...' --refresh --output "$XDG_RUNTIME_DIR/notion-getpage-check.md"
```

The fallback based on the rendered page can miss media, databases, and blocks that have not loaded. Mention these limits when reporting a test result; do not present a partial extraction as complete.

## Protect private data

Never include Notion passwords, cookies, browser profiles, private page URLs, or extracted private Markdown in commits, issues, test fixtures, or logs. Use a public page or a synthetic example when sharing a reproduction. Files written with `--output` outside `XDG_RUNTIME_DIR` are not automatically removed at reboot.

Keep the browser sandbox and site isolation enabled. The clipboard permission is granted only during extraction; do not expand its scope without a clear reason. Treat page content as untrusted input, especially when it will be read by AI agents.

When proposing a change, include what changed, how you tested it, and any remaining limitations. Do not publish or paste output from a private workspace to demonstrate success.

## Publish a release

On GitHub, open **Actions → Release → Run workflow**, enter a semantic version such as `v0.1.0`, and start the workflow from `master`. It creates the tag when needed, runs tests, builds the Linux x86-64 binary, and publishes the release assets. Future releases use a new version such as `v0.1.1`.
