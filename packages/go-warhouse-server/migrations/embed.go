// Package migrations holds the embedded forward-only SQL schema
// migrations. Kept at the repository root per PLAN.md §4 so the
// schema history stays visible; internal/db consumes FS.
package migrations

import "embed"

// FS contains the *.sql schema migrations in version order by name.
//
//go:embed *.sql
var FS embed.FS
