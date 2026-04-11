//go:build tools

// tools.go pins gqlgen as a dependency so `go run github.com/99designs/gqlgen`
// works. This file is never compiled into the binary — the build tag excludes it.
package tools

import _ "github.com/99designs/gqlgen"
