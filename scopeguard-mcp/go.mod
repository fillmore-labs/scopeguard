module fillmore-labs.com/scopeguard/scopeguard-mcp

go 1.26

toolchain go1.26.5

replace fillmore-labs.com/scopeguard => ..

tool fillmore-labs.com/scopeguard/internal/cmd/bitmask

require (
	fillmore-labs.com/scopeguard v0.0.8
	github.com/google/jsonschema-go v0.4.3
	github.com/modelcontextprotocol/go-sdk v1.7.0
	golang.org/x/tools v0.48.0
)

require (
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
