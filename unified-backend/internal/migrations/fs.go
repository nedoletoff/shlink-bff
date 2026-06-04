// Package migrations embeds SQL migration files into the binary.
package migrations

import "embed"

//go:embed sql/*.sql
var FS embed.FS
