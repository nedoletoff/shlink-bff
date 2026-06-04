// Package migrations embeds SQL migration files.
package migrations

import "embed"

// FS contains all *.sql files from the sql/ subdirectory.
// go:embed path must not contain "..", so SQL files live next to this file.
//
//go:embed sql/*.sql
var FS embed.FS
