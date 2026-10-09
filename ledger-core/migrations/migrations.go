// Package migrations bundles the SQL schema files into the binary so the
// service can create its own tables on first connect.
package migrations

import "embed"

// FS holds every *.sql file in this directory.

//go:embed *.sql
var FS embed.FS
