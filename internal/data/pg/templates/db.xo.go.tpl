// Storage is the helper struct for database operations
type Storage struct {
    db         *pgdb.DB
}

// New - returns new instance of storage
func New(db *pgdb.DB) *Storage {
	return &Storage{
		db,
	}
}

// DB - returns db used by Storage
func (s *Storage) DB() *pgdb.DB {
	return s.db
}

// Clone - returns new storage with clone of db
func (s *Storage) Clone() *Storage {
	return New(s.db.Clone())
}

// Transaction begins a transaction on repo.
func (s *Storage) Transaction(tx func() error) error {
    return s.db.Transaction(tx)
}