package db

import (
	"database/sql"
	"encoding/json"
	"time"
)

// JSONNullString serializes sql.NullString as a plain string or null.
type JSONNullString struct {
	sql.NullString
}

func (s JSONNullString) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(s.String)
}

func (s *JSONNullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.Valid = false
		return nil
	}
	s.Valid = true
	return json.Unmarshal(data, &s.String)
}

// JSONNullTime serializes sql.NullTime as an ISO string or null.
type JSONNullTime struct {
	sql.NullTime
}

func (t JSONNullTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time)
}

func (t *JSONNullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Valid = false
		return nil
	}
	t.Valid = true
	return json.Unmarshal(data, &t.Time)
}

// JSONNullInt64 serializes sql.NullInt64 as a number or null.
type JSONNullInt64 struct {
	sql.NullInt64
}

func (n JSONNullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Int64)
}

func (n *JSONNullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return json.Unmarshal(data, &n.Int64)
}

// Helpers to create values
func NewJSONNullString(s string, valid bool) JSONNullString {
	return JSONNullString{sql.NullString{String: s, Valid: valid}}
}

func NewJSONNullTime(t time.Time, valid bool) JSONNullTime {
	return JSONNullTime{sql.NullTime{Time: t, Valid: valid}}
}

func NewJSONNullInt64(n int64, valid bool) JSONNullInt64 {
	return JSONNullInt64{sql.NullInt64{Int64: n, Valid: valid}}
}
