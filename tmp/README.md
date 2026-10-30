> #301 / #1009 / #1010: the tracked tmp/go.mod and old managed debug/tool chains are retired. This directory is not an executable or maintained Go module. Any older module instructions below are historical; no existing local files are inspected or cleaned.

# tmp

Local one-off debug programs live here.

This directory is intentionally a separate Go module so `go test ./...` from the
main repository does not try to compile every ad-hoc `main` package under `tmp/`.
