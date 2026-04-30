module fillmore-labs.com/scopeguard

go 1.25.0

toolchain go1.26.5

require (
	github.com/golangci/plugin-module-register v0.1.2
	golang.org/x/tools v0.48.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
)

tool (
	fillmore-labs.com/scopeguard/internal/cmd/bitmask
	golang.org/x/tools/cmd/stringer
)
