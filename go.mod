module fillmore-labs.com/scopeguard

go 1.26.0

toolchain go1.27.4

require (
	github.com/golangci/plugin-module-register v0.1.2
	golang.org/x/tools v0.53.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
)

tool (
	fillmore-labs.com/scopeguard/internal/cmd/bitmask
	golang.org/x/tools/cmd/stringer
)
