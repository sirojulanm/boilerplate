// Package migrations embeds SQL migration files so the compiled binary can run
// migrations without shipping loose files.
package migrations

import "embed"

// FS contains every migration in this directory.
//
//go:embed *.sql
var FS embed.FS
