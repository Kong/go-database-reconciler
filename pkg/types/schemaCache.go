package types

import (
	"github.com/kong/go-database-reconciler/pkg/schema"
)

// We are keeping this file as deck relies on types.SchemaCache.
// We can remove this once we update deck to use the new schema.Cache directly.
// Otherwise, test builds would fail.

// SchemaFetcher is the function signature for fetching a schema by identifier.
// Deprecated: Use schema.Fetcher instead.
type SchemaFetcher = schema.Fetcher

// SchemaCache is a thread-safe cache for schemas keyed by identifier.
// Deprecated: Use schema.Cache instead.
type SchemaCache = schema.Cache

// NewSchemaCache creates a new SchemaCache with the given fetcher.
// Deprecated: Use schema.NewCache instead.
var NewSchemaCache = schema.NewCache
