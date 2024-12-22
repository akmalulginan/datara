package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SchemaState menyimpan state dari schema database
type SchemaState struct {
	Version string           `json:"version"`
	Tables  map[string]Table `json:"tables"`
}

// Table merepresentasikan state dari sebuah tabel
type Table struct {
	Name        string            `json:"name"`
	Columns     map[string]Column `json:"columns"`
	Indexes     map[string]Index  `json:"indexes"`
	Constraints []Constraint      `json:"constraints"`
}

// Column merepresentasikan state dari sebuah kolom
type Column struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Nullable      bool        `json:"nullable"`
	DefaultValue  interface{} `json:"default_value,omitempty"`
	AutoIncrement bool        `json:"auto_increment,omitempty"`
}

// Index merepresentasikan state dari sebuah index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// Constraint merepresentasikan constraint pada tabel
type Constraint struct {
	Name string `json:"name"`
	Type string `json:"type"` // e.g., "PRIMARY KEY", "FOREIGN KEY", etc.
	Def  string `json:"def"`  // SQL definition
}

// NewSchemaState membuat instance baru dari SchemaState
func NewSchemaState() *SchemaState {
	return &SchemaState{
		Version: "1.0",
		Tables:  make(map[string]Table),
	}
}

// SaveToFile menyimpan state ke file
func (s *SchemaState) SaveToFile(path string) error {
	// Pastikan direktori ada
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal ke JSON dengan indentasi
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Tulis ke file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadFromFile membaca state dari file
func LoadFromFile(path string) (*SchemaState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Jika file tidak ada, return state baru
			return NewSchemaState(), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state SchemaState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// AddTable menambahkan atau memperbarui tabel ke state
func (s *SchemaState) AddTable(table Table) {
	s.Tables[table.Name] = table
}

// GetTable mengambil tabel dari state
func (s *SchemaState) GetTable(name string) (Table, bool) {
	table, exists := s.Tables[name]
	return table, exists
}

// RemoveTable menghapus tabel dari state
func (s *SchemaState) RemoveTable(name string) {
	delete(s.Tables, name)
}
