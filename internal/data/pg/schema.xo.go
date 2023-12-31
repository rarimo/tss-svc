// Package pg contains generated code for schema 'public'.
package pg

// Code generated by xo. DO NOT EDIT.

import (
	"context"
	"database/sql"

	"github.com/rarimo/tss-svc/internal/data"
	"gitlab.com/distributed_lab/kit/pgdb"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

// Storage is the helper struct for database operations
type Storage struct {
	db *pgdb.DB
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
} // DefaultSessionDatumQ represents helper struct to access row of 'default_session_data'.
type DefaultSessionDatumQ struct {
	db *pgdb.DB
}

// NewDefaultSessionDatumQ  - creates new instance
func NewDefaultSessionDatumQ(db *pgdb.DB) *DefaultSessionDatumQ {
	return &DefaultSessionDatumQ{
		db,
	}
}

// DefaultSessionDatumQ  - creates new instance of DefaultSessionDatumQ
func (s Storage) DefaultSessionDatumQ() *DefaultSessionDatumQ {
	return NewDefaultSessionDatumQ(s.DB())
}

var colsDefaultSessionDatum = `id, status, begin_block, end_block, parties, proposer, indexes, root, accepted, signature`

// InsertCtx inserts a DefaultSessionDatum to the database.
func (q DefaultSessionDatumQ) InsertCtx(ctx context.Context, dsd *data.DefaultSessionDatum) error {
	// sql insert query, primary key must be provided
	sqlstr := `INSERT INTO public.default_session_data (` +
		`id, status, begin_block, end_block, parties, proposer, indexes, root, accepted, signature` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7, $8, $9, $10` +
		`)`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, dsd.ID, dsd.Status, dsd.BeginBlock, dsd.EndBlock, dsd.Parties, dsd.Proposer, dsd.Indexes, dsd.Root, dsd.Accepted, dsd.Signature)
	return errors.Wrap(err, "failed to execute insert query")
}

// Insert insert a DefaultSessionDatum to the database.
func (q DefaultSessionDatumQ) Insert(dsd *data.DefaultSessionDatum) error {
	return q.InsertCtx(context.Background(), dsd)
}

// UpdateCtx updates a DefaultSessionDatum in the database.
func (q DefaultSessionDatumQ) UpdateCtx(ctx context.Context, dsd *data.DefaultSessionDatum) error {
	// update with composite primary key
	sqlstr := `UPDATE public.default_session_data SET ` +
		`status = $1, begin_block = $2, end_block = $3, parties = $4, proposer = $5, indexes = $6, root = $7, accepted = $8, signature = $9 ` +
		`WHERE id = $10`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, dsd.Status, dsd.BeginBlock, dsd.EndBlock, dsd.Parties, dsd.Proposer, dsd.Indexes, dsd.Root, dsd.Accepted, dsd.Signature, dsd.ID)
	return errors.Wrap(err, "failed to execute update")
}

// Update updates a DefaultSessionDatum in the database.
func (q DefaultSessionDatumQ) Update(dsd *data.DefaultSessionDatum) error {
	return q.UpdateCtx(context.Background(), dsd)
}

// UpsertCtx performs an upsert for DefaultSessionDatum.
func (q DefaultSessionDatumQ) UpsertCtx(ctx context.Context, dsd *data.DefaultSessionDatum) error {
	// upsert
	sqlstr := `INSERT INTO public.default_session_data (` +
		`id, status, begin_block, end_block, parties, proposer, indexes, root, accepted, signature` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7, $8, $9, $10` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`status = EXCLUDED.status, begin_block = EXCLUDED.begin_block, end_block = EXCLUDED.end_block, parties = EXCLUDED.parties, proposer = EXCLUDED.proposer, indexes = EXCLUDED.indexes, root = EXCLUDED.root, accepted = EXCLUDED.accepted, signature = EXCLUDED.signature `
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, dsd.ID, dsd.Status, dsd.BeginBlock, dsd.EndBlock, dsd.Parties, dsd.Proposer, dsd.Indexes, dsd.Root, dsd.Accepted, dsd.Signature); err != nil {
		return errors.Wrap(err, "failed to execute upsert stmt")
	}
	return nil
}

// Upsert performs an upsert for DefaultSessionDatum.
func (q DefaultSessionDatumQ) Upsert(dsd *data.DefaultSessionDatum) error {
	return q.UpsertCtx(context.Background(), dsd)
}

// DeleteCtx deletes the DefaultSessionDatum from the database.
func (q DefaultSessionDatumQ) DeleteCtx(ctx context.Context, dsd *data.DefaultSessionDatum) error {
	// delete with single primary key
	sqlstr := `DELETE FROM public.default_session_data ` +
		`WHERE id = $1`
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, dsd.ID); err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
	return nil
}

// Delete deletes the DefaultSessionDatum from the database.
func (q DefaultSessionDatumQ) Delete(dsd *data.DefaultSessionDatum) error {
	return q.DeleteCtx(context.Background(), dsd)
} // GorpMigrationQ represents helper struct to access row of 'gorp_migrations'.
type GorpMigrationQ struct {
	db *pgdb.DB
}

// NewGorpMigrationQ  - creates new instance
func NewGorpMigrationQ(db *pgdb.DB) *GorpMigrationQ {
	return &GorpMigrationQ{
		db,
	}
}

// GorpMigrationQ  - creates new instance of GorpMigrationQ
func (s Storage) GorpMigrationQ() *GorpMigrationQ {
	return NewGorpMigrationQ(s.DB())
}

var colsGorpMigration = `id, applied_at`

// InsertCtx inserts a GorpMigration to the database.
func (q GorpMigrationQ) InsertCtx(ctx context.Context, gm *data.GorpMigration) error {
	// sql insert query, primary key must be provided
	sqlstr := `INSERT INTO public.gorp_migrations (` +
		`id, applied_at` +
		`) VALUES (` +
		`$1, $2` +
		`)`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, gm.ID, gm.AppliedAt)
	return errors.Wrap(err, "failed to execute insert query")
}

// Insert insert a GorpMigration to the database.
func (q GorpMigrationQ) Insert(gm *data.GorpMigration) error {
	return q.InsertCtx(context.Background(), gm)
}

// UpdateCtx updates a GorpMigration in the database.
func (q GorpMigrationQ) UpdateCtx(ctx context.Context, gm *data.GorpMigration) error {
	// update with composite primary key
	sqlstr := `UPDATE public.gorp_migrations SET ` +
		`applied_at = $1 ` +
		`WHERE id = $2`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, gm.AppliedAt, gm.ID)
	return errors.Wrap(err, "failed to execute update")
}

// Update updates a GorpMigration in the database.
func (q GorpMigrationQ) Update(gm *data.GorpMigration) error {
	return q.UpdateCtx(context.Background(), gm)
}

// UpsertCtx performs an upsert for GorpMigration.
func (q GorpMigrationQ) UpsertCtx(ctx context.Context, gm *data.GorpMigration) error {
	// upsert
	sqlstr := `INSERT INTO public.gorp_migrations (` +
		`id, applied_at` +
		`) VALUES (` +
		`$1, $2` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`applied_at = EXCLUDED.applied_at `
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, gm.ID, gm.AppliedAt); err != nil {
		return errors.Wrap(err, "failed to execute upsert stmt")
	}
	return nil
}

// Upsert performs an upsert for GorpMigration.
func (q GorpMigrationQ) Upsert(gm *data.GorpMigration) error {
	return q.UpsertCtx(context.Background(), gm)
}

// DeleteCtx deletes the GorpMigration from the database.
func (q GorpMigrationQ) DeleteCtx(ctx context.Context, gm *data.GorpMigration) error {
	// delete with single primary key
	sqlstr := `DELETE FROM public.gorp_migrations ` +
		`WHERE id = $1`
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, gm.ID); err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
	return nil
}

// Delete deletes the GorpMigration from the database.
func (q GorpMigrationQ) Delete(gm *data.GorpMigration) error {
	return q.DeleteCtx(context.Background(), gm)
} // KeygenSessionDatumQ represents helper struct to access row of 'keygen_session_data'.
type KeygenSessionDatumQ struct {
	db *pgdb.DB
}

// NewKeygenSessionDatumQ  - creates new instance
func NewKeygenSessionDatumQ(db *pgdb.DB) *KeygenSessionDatumQ {
	return &KeygenSessionDatumQ{
		db,
	}
}

// KeygenSessionDatumQ  - creates new instance of KeygenSessionDatumQ
func (s Storage) KeygenSessionDatumQ() *KeygenSessionDatumQ {
	return NewKeygenSessionDatumQ(s.DB())
}

var colsKeygenSessionDatum = `id, status, begin_block, end_block, parties, key`

// InsertCtx inserts a KeygenSessionDatum to the database.
func (q KeygenSessionDatumQ) InsertCtx(ctx context.Context, ksd *data.KeygenSessionDatum) error {
	// sql insert query, primary key must be provided
	sqlstr := `INSERT INTO public.keygen_session_data (` +
		`id, status, begin_block, end_block, parties, key` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6` +
		`)`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, ksd.ID, ksd.Status, ksd.BeginBlock, ksd.EndBlock, ksd.Parties, ksd.Key)
	return errors.Wrap(err, "failed to execute insert query")
}

// Insert insert a KeygenSessionDatum to the database.
func (q KeygenSessionDatumQ) Insert(ksd *data.KeygenSessionDatum) error {
	return q.InsertCtx(context.Background(), ksd)
}

// UpdateCtx updates a KeygenSessionDatum in the database.
func (q KeygenSessionDatumQ) UpdateCtx(ctx context.Context, ksd *data.KeygenSessionDatum) error {
	// update with composite primary key
	sqlstr := `UPDATE public.keygen_session_data SET ` +
		`status = $1, begin_block = $2, end_block = $3, parties = $4, key = $5 ` +
		`WHERE id = $6`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, ksd.Status, ksd.BeginBlock, ksd.EndBlock, ksd.Parties, ksd.Key, ksd.ID)
	return errors.Wrap(err, "failed to execute update")
}

// Update updates a KeygenSessionDatum in the database.
func (q KeygenSessionDatumQ) Update(ksd *data.KeygenSessionDatum) error {
	return q.UpdateCtx(context.Background(), ksd)
}

// UpsertCtx performs an upsert for KeygenSessionDatum.
func (q KeygenSessionDatumQ) UpsertCtx(ctx context.Context, ksd *data.KeygenSessionDatum) error {
	// upsert
	sqlstr := `INSERT INTO public.keygen_session_data (` +
		`id, status, begin_block, end_block, parties, key` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`status = EXCLUDED.status, begin_block = EXCLUDED.begin_block, end_block = EXCLUDED.end_block, parties = EXCLUDED.parties, key = EXCLUDED.key `
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, ksd.ID, ksd.Status, ksd.BeginBlock, ksd.EndBlock, ksd.Parties, ksd.Key); err != nil {
		return errors.Wrap(err, "failed to execute upsert stmt")
	}
	return nil
}

// Upsert performs an upsert for KeygenSessionDatum.
func (q KeygenSessionDatumQ) Upsert(ksd *data.KeygenSessionDatum) error {
	return q.UpsertCtx(context.Background(), ksd)
}

// DeleteCtx deletes the KeygenSessionDatum from the database.
func (q KeygenSessionDatumQ) DeleteCtx(ctx context.Context, ksd *data.KeygenSessionDatum) error {
	// delete with single primary key
	sqlstr := `DELETE FROM public.keygen_session_data ` +
		`WHERE id = $1`
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, ksd.ID); err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
	return nil
}

// Delete deletes the KeygenSessionDatum from the database.
func (q KeygenSessionDatumQ) Delete(ksd *data.KeygenSessionDatum) error {
	return q.DeleteCtx(context.Background(), ksd)
} // ReshareSessionDatumQ represents helper struct to access row of 'reshare_session_data'.
type ReshareSessionDatumQ struct {
	db *pgdb.DB
}

// NewReshareSessionDatumQ  - creates new instance
func NewReshareSessionDatumQ(db *pgdb.DB) *ReshareSessionDatumQ {
	return &ReshareSessionDatumQ{
		db,
	}
}

// ReshareSessionDatumQ  - creates new instance of ReshareSessionDatumQ
func (s Storage) ReshareSessionDatumQ() *ReshareSessionDatumQ {
	return NewReshareSessionDatumQ(s.DB())
}

var colsReshareSessionDatum = `id, status, begin_block, end_block, parties, proposer, old_key, new_key, key_signature, signature, root`

// InsertCtx inserts a ReshareSessionDatum to the database.
func (q ReshareSessionDatumQ) InsertCtx(ctx context.Context, rsd *data.ReshareSessionDatum) error {
	// sql insert query, primary key must be provided
	sqlstr := `INSERT INTO public.reshare_session_data (` +
		`id, status, begin_block, end_block, parties, proposer, old_key, new_key, key_signature, signature, root` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11` +
		`)`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, rsd.ID, rsd.Status, rsd.BeginBlock, rsd.EndBlock, rsd.Parties, rsd.Proposer, rsd.OldKey, rsd.NewKey, rsd.KeySignature, rsd.Signature, rsd.Root)
	return errors.Wrap(err, "failed to execute insert query")
}

// Insert insert a ReshareSessionDatum to the database.
func (q ReshareSessionDatumQ) Insert(rsd *data.ReshareSessionDatum) error {
	return q.InsertCtx(context.Background(), rsd)
}

// UpdateCtx updates a ReshareSessionDatum in the database.
func (q ReshareSessionDatumQ) UpdateCtx(ctx context.Context, rsd *data.ReshareSessionDatum) error {
	// update with composite primary key
	sqlstr := `UPDATE public.reshare_session_data SET ` +
		`status = $1, begin_block = $2, end_block = $3, parties = $4, proposer = $5, old_key = $6, new_key = $7, key_signature = $8, signature = $9, root = $10 ` +
		`WHERE id = $11`
	// run
	err := q.db.ExecRawContext(ctx, sqlstr, rsd.Status, rsd.BeginBlock, rsd.EndBlock, rsd.Parties, rsd.Proposer, rsd.OldKey, rsd.NewKey, rsd.KeySignature, rsd.Signature, rsd.Root, rsd.ID)
	return errors.Wrap(err, "failed to execute update")
}

// Update updates a ReshareSessionDatum in the database.
func (q ReshareSessionDatumQ) Update(rsd *data.ReshareSessionDatum) error {
	return q.UpdateCtx(context.Background(), rsd)
}

// UpsertCtx performs an upsert for ReshareSessionDatum.
func (q ReshareSessionDatumQ) UpsertCtx(ctx context.Context, rsd *data.ReshareSessionDatum) error {
	// upsert
	sqlstr := `INSERT INTO public.reshare_session_data (` +
		`id, status, begin_block, end_block, parties, proposer, old_key, new_key, key_signature, signature, root` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`status = EXCLUDED.status, begin_block = EXCLUDED.begin_block, end_block = EXCLUDED.end_block, parties = EXCLUDED.parties, proposer = EXCLUDED.proposer, old_key = EXCLUDED.old_key, new_key = EXCLUDED.new_key, key_signature = EXCLUDED.key_signature, signature = EXCLUDED.signature, root = EXCLUDED.root `
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, rsd.ID, rsd.Status, rsd.BeginBlock, rsd.EndBlock, rsd.Parties, rsd.Proposer, rsd.OldKey, rsd.NewKey, rsd.KeySignature, rsd.Signature, rsd.Root); err != nil {
		return errors.Wrap(err, "failed to execute upsert stmt")
	}
	return nil
}

// Upsert performs an upsert for ReshareSessionDatum.
func (q ReshareSessionDatumQ) Upsert(rsd *data.ReshareSessionDatum) error {
	return q.UpsertCtx(context.Background(), rsd)
}

// DeleteCtx deletes the ReshareSessionDatum from the database.
func (q ReshareSessionDatumQ) DeleteCtx(ctx context.Context, rsd *data.ReshareSessionDatum) error {
	// delete with single primary key
	sqlstr := `DELETE FROM public.reshare_session_data ` +
		`WHERE id = $1`
	// run
	if err := q.db.ExecRawContext(ctx, sqlstr, rsd.ID); err != nil {
		return errors.Wrap(err, "failed to exec delete stmt")
	}
	return nil
}

// Delete deletes the ReshareSessionDatum from the database.
func (q ReshareSessionDatumQ) Delete(rsd *data.ReshareSessionDatum) error {
	return q.DeleteCtx(context.Background(), rsd)
}

// DefaultSessionDatumByIDCtx retrieves a row from 'public.default_session_data' as a DefaultSessionDatum.
//
// Generated from index 'default_session_data_pkey'.
func (q DefaultSessionDatumQ) DefaultSessionDatumByIDCtx(ctx context.Context, id int64, isForUpdate bool) (*data.DefaultSessionDatum, error) {
	// query
	sqlstr := `SELECT ` +
		`id, status, begin_block, end_block, parties, proposer, indexes, root, accepted, signature ` +
		`FROM public.default_session_data ` +
		`WHERE id = $1`
	// run
	if isForUpdate {
		sqlstr += " for update"
	}
	var res data.DefaultSessionDatum
	err := q.db.GetRawContext(ctx, &res, sqlstr, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to exec select")
	}

	return &res, nil
}

// DefaultSessionDatumByID retrieves a row from 'public.default_session_data' as a DefaultSessionDatum.
//
// Generated from index 'default_session_data_pkey'.
func (q DefaultSessionDatumQ) DefaultSessionDatumByID(id int64, isForUpdate bool) (*data.DefaultSessionDatum, error) {
	return q.DefaultSessionDatumByIDCtx(context.Background(), id, isForUpdate)
}

// GorpMigrationByIDCtx retrieves a row from 'public.gorp_migrations' as a GorpMigration.
//
// Generated from index 'gorp_migrations_pkey'.
func (q GorpMigrationQ) GorpMigrationByIDCtx(ctx context.Context, id string, isForUpdate bool) (*data.GorpMigration, error) {
	// query
	sqlstr := `SELECT ` +
		`id, applied_at ` +
		`FROM public.gorp_migrations ` +
		`WHERE id = $1`
	// run
	if isForUpdate {
		sqlstr += " for update"
	}
	var res data.GorpMigration
	err := q.db.GetRawContext(ctx, &res, sqlstr, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to exec select")
	}

	return &res, nil
}

// GorpMigrationByID retrieves a row from 'public.gorp_migrations' as a GorpMigration.
//
// Generated from index 'gorp_migrations_pkey'.
func (q GorpMigrationQ) GorpMigrationByID(id string, isForUpdate bool) (*data.GorpMigration, error) {
	return q.GorpMigrationByIDCtx(context.Background(), id, isForUpdate)
}

// KeygenSessionDatumByIDCtx retrieves a row from 'public.keygen_session_data' as a KeygenSessionDatum.
//
// Generated from index 'keygen_session_data_pkey'.
func (q KeygenSessionDatumQ) KeygenSessionDatumByIDCtx(ctx context.Context, id int64, isForUpdate bool) (*data.KeygenSessionDatum, error) {
	// query
	sqlstr := `SELECT ` +
		`id, status, begin_block, end_block, parties, key ` +
		`FROM public.keygen_session_data ` +
		`WHERE id = $1`
	// run
	if isForUpdate {
		sqlstr += " for update"
	}
	var res data.KeygenSessionDatum
	err := q.db.GetRawContext(ctx, &res, sqlstr, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to exec select")
	}

	return &res, nil
}

// KeygenSessionDatumByID retrieves a row from 'public.keygen_session_data' as a KeygenSessionDatum.
//
// Generated from index 'keygen_session_data_pkey'.
func (q KeygenSessionDatumQ) KeygenSessionDatumByID(id int64, isForUpdate bool) (*data.KeygenSessionDatum, error) {
	return q.KeygenSessionDatumByIDCtx(context.Background(), id, isForUpdate)
}

// ReshareSessionDatumByIDCtx retrieves a row from 'public.reshare_session_data' as a ReshareSessionDatum.
//
// Generated from index 'reshare_session_data_pkey'.
func (q ReshareSessionDatumQ) ReshareSessionDatumByIDCtx(ctx context.Context, id int64, isForUpdate bool) (*data.ReshareSessionDatum, error) {
	// query
	sqlstr := `SELECT ` +
		`id, status, begin_block, end_block, parties, proposer, old_key, new_key, key_signature, signature, root ` +
		`FROM public.reshare_session_data ` +
		`WHERE id = $1`
	// run
	if isForUpdate {
		sqlstr += " for update"
	}
	var res data.ReshareSessionDatum
	err := q.db.GetRawContext(ctx, &res, sqlstr, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to exec select")
	}

	return &res, nil
}

// ReshareSessionDatumByID retrieves a row from 'public.reshare_session_data' as a ReshareSessionDatum.
//
// Generated from index 'reshare_session_data_pkey'.
func (q ReshareSessionDatumQ) ReshareSessionDatumByID(id int64, isForUpdate bool) (*data.ReshareSessionDatum, error) {
	return q.ReshareSessionDatumByIDCtx(context.Background(), id, isForUpdate)
}
