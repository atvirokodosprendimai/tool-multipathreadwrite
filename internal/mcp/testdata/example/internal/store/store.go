// Package store keeps rows in memory, keyed by id.
//
// It is the fixture the MCP surface's shipped examples run against
// (ADR-090): the worked plan rewrites Get, and the worked read serves Put.
// Move a line here and the examples stop matching it, which is the point.
package store

import "errors"

// ErrExists is returned by Put when the id is already taken.
var ErrExists = errors.New("store: id exists")

// ErrMissing is returned by Delete when there is no row to remove.
var ErrMissing = errors.New("store: no such id")

// Row is one stored record.
type Row struct {
	ID    string
	Name  string
	Count int
}

// Store holds rows. The zero value is not usable; call New.
type Store struct {
	rows map[string]Row
}

// New returns an empty Store.
func New() *Store {
	return &Store{rows: map[string]Row{}}
}

// Len reports how many rows the store holds.
func (s *Store) Len() int {
	return len(s.rows)
}

// Get returns the row stored under id, or the zero Row when there is none.
// A caller cannot tell a missing row from a stored zero Row, which is what
// the worked plan fixes.
// The declaration spans lines 42-44, the range the worked plan replaces.
func (s *Store) Get(id string) Row {
	return s.rows[id]
}

// Put stores r under r.ID, refusing an id that is already taken.
func (s *Store) Put(r Row) error {
	if _, ok := s.rows[r.ID]; ok {
		return ErrExists
	}
	s.rows[r.ID] = r
	return nil
}

// Delete removes the row stored under id.
func (s *Store) Delete(id string) error {
	if _, ok := s.rows[id]; !ok {
		return ErrMissing
	}
	delete(s.rows, id)
	return nil
}
