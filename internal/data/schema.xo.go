// Package data contains generated code for schema 'public'.
package data

// Code generated by xo. DO NOT EDIT.

import (
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// StringSlice is a slice of strings.
type StringSlice []string

// quoteEscapeRegex is the regex to match escaped characters in a string.
var quoteEscapeRegex = regexp.MustCompile(`([^\\]([\\]{2})*)\\"`)

// Scan satisfies the sql.Scanner interface for StringSlice.
func (ss *StringSlice) Scan(src interface{}) error {
	buf, ok := src.([]byte)
	if !ok {
		return errors.New("invalid StringSlice")
	}

	// change quote escapes for csv parser
	str := quoteEscapeRegex.ReplaceAllString(string(buf), `$1""`)
	str = strings.Replace(str, `\\`, `\`, -1)

	// remove braces
	str = str[1 : len(str)-1]

	// bail if only one
	if len(str) == 0 {
		*ss = StringSlice([]string{})
		return nil
	}

	// parse with csv reader
	cr := csv.NewReader(strings.NewReader(str))
	slice, err := cr.Read()
	if err != nil {
		fmt.Printf("exiting!: %v\n", err)
		return err
	}

	*ss = StringSlice(slice)

	return nil
}

// Value satisfies the driver.Valuer interface for StringSlice.
func (ss StringSlice) Value() (driver.Value, error) {
	v := make([]string, len(ss))
	for i, s := range ss {
		v[i] = `"` + strings.Replace(strings.Replace(s, `\`, `\\\`, -1), `"`, `\"`, -1) + `"`
	}
	return "{" + strings.Join(v, ",") + "}", nil
} // DefaultSessionDatum represents a row from 'public.default_session_data'.
type DefaultSessionDatum struct {
	ID         int64          `db:"id"`          // id
	Status     int            `db:"status"`      // status
	BeginBlock int64          `db:"begin_block"` // begin_block
	EndBlock   int64          `db:"end_block"`   // end_block
	Parties    StringSlice    `db:"parties"`     // parties
	Proposer   sql.NullString `db:"proposer"`    // proposer
	Indexes    StringSlice    `db:"indexes"`     // indexes
	Root       sql.NullString `db:"root"`        // root
	Accepted   StringSlice    `db:"accepted"`    // accepted
	Signature  sql.NullString `db:"signature"`   // signature

}

// GorpMigration represents a row from 'public.gorp_migrations'.
type GorpMigration struct {
	ID        string       `db:"id"`         // id
	AppliedAt sql.NullTime `db:"applied_at"` // applied_at

}

// KeygenSessionDatum represents a row from 'public.keygen_session_data'.
type KeygenSessionDatum struct {
	ID         int64          `db:"id"`          // id
	Status     int            `db:"status"`      // status
	BeginBlock int64          `db:"begin_block"` // begin_block
	EndBlock   int64          `db:"end_block"`   // end_block
	Parties    StringSlice    `db:"parties"`     // parties
	Key        sql.NullString `db:"key"`         // key

}

// ReshareSessionDatum represents a row from 'public.reshare_session_data'.
type ReshareSessionDatum struct {
	ID           int64          `db:"id"`            // id
	Status       int            `db:"status"`        // status
	BeginBlock   int64          `db:"begin_block"`   // begin_block
	EndBlock     int64          `db:"end_block"`     // end_block
	Parties      StringSlice    `db:"parties"`       // parties
	Proposer     sql.NullString `db:"proposer"`      // proposer
	OldKey       sql.NullString `db:"old_key"`       // old_key
	NewKey       sql.NullString `db:"new_key"`       // new_key
	KeySignature sql.NullString `db:"key_signature"` // key_signature
	Signature    sql.NullString `db:"signature"`     // signature
	Root         sql.NullString `db:"root"`          // root

}
