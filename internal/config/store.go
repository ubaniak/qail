package config

// Store is the persistence seam for Config. Two adapters live in this
// package: SQLiteStore (production, ~/.qail/qail.db) and MemoryStore (tests).
//
// Read returns a snapshot; the caller mutates and hands it back via Write.
// Write replaces the entire stored Config — there are no partial updates at
// this layer. Atomic domain operations (AddRepo, RemoveWorkspace) belong in
// a higher-level Action module that wraps Read+mutate+Write.
type Store interface {
	Read() (Config, error)
	Write(cfg Config) error
}
