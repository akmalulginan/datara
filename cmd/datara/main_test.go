package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/akmalulginan/datara"
)

func TestGenerateSQL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestGenerateSQL")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	migrationsDir := filepath.Join(tmpDir, "001")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatal(err)
	}

	schema := &datara.Schema{
		Tables: []*datara.Table{
			{
				Name: "users",
				Columns: []*datara.Column{
					{Name: "id", Type: "integer", Tags: map[string]string{"primary_key": "", "autoincrement": ""}},
					{Name: "name", Type: "varchar", Tags: map[string]string{"notnull": ""}},
					{Name: "email", Type: "varchar", Tags: map[string]string{"unique": ""}},
					{Name: "created_at", Type: "timestamp"},
				},
			},
		},
	}

	config := &Config{
		Migration: MigrationConfig{
			Dir:    migrationsDir,
			Format: "sql",
		},
	}

	err = generateSQL(schema, config)
	if err != nil {
		t.Fatalf("generateSQL() error = %v", err)
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 migration file, got %d", len(files))
	}
}

func TestGenerateJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestGenerateJSON")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	outputDir := filepath.Join(tmpDir, "001")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	schema := &datara.Schema{
		Tables: []*datara.Table{
			{
				Name: "users",
				Columns: []*datara.Column{
					{Name: "id", Type: "integer"},
					{Name: "name", Type: "varchar"},
					{Name: "email", Type: "varchar"},
					{Name: "created_at", Type: "timestamp"},
				},
			},
		},
	}

	config := &Config{
		Migration: MigrationConfig{
			Dir:    outputDir,
			Format: "json",
		},
	}

	err = generateJSON(schema, config)
	if err != nil {
		t.Fatalf("generateJSON() error = %v", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 JSON file, got %d", len(files))
	}
}

func TestExecuteSchemaProgram(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestExecuteSchemaProgram")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	goFile := filepath.Join(srcDir, "schema.go")
	content := []byte(`package main

import (
	"encoding/json"
	"fmt"
)

type Schema struct {
	Tables []*Table
}

type Table struct {
	Name    string
	Columns []*Column
}

type Column struct {
	Name string
	Type string
	Tags map[string]string
}

func main() {
	schema := &Schema{
		Tables: []*Table{
			{
				Name: "users",
				Columns: []*Column{
					{Name: "id", Type: "integer", Tags: map[string]string{"primary_key": "", "autoincrement": ""}},
					{Name: "name", Type: "varchar", Tags: map[string]string{"notnull": ""}},
				},
			},
		},
	}
	data, _ := json.Marshal(schema)
	fmt.Print(string(data))
}`)

	if err := os.WriteFile(goFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	modFile := filepath.Join(srcDir, "go.mod")
	modContent := []byte(`module schema

go 1.21
`)
	if err := os.WriteFile(modFile, modContent, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	schema, err := executeSchemaProgram([]string{"go", "run", goFile})
	if err != nil {
		t.Fatalf("executeSchemaProgram() error = %v", err)
	}

	var schemaObj struct {
		Tables []*struct {
			Name    string
			Columns []*struct {
				Name string
				Type string
				Tags map[string]string
			}
		}
	}
	if err := json.Unmarshal([]byte(schema), &schemaObj); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	if len(schemaObj.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(schemaObj.Tables))
	}
	if schemaObj.Tables[0].Name != "users" {
		t.Errorf("Expected table name 'users', got '%s'", schemaObj.Tables[0].Name)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Migration.Dir != "migrations" {
		t.Errorf("Expected default migration dir 'migrations', got '%s'", config.Migration.Dir)
	}
	if config.Migration.Format != "sql" {
		t.Errorf("Expected default migration format 'sql', got '%s'", config.Migration.Format)
	}
	if config.Migration.Charset != "utf8mb4" {
		t.Errorf("Expected default charset 'utf8mb4', got '%s'", config.Migration.Charset)
	}
	if config.Migration.Engine != "InnoDB" {
		t.Errorf("Expected default engine 'InnoDB', got '%s'", config.Migration.Engine)
	}
}

func TestRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestRun")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	goFile := filepath.Join(srcDir, "schema.go")
	content := []byte(`package main

import (
	"encoding/json"
	"fmt"
)

type Schema struct {
	Tables []*Table
}

type Table struct {
	Name    string
	Columns []*Column
}

type Column struct {
	Name string
	Type string
	Tags map[string]string
}

func main() {
	schema := &Schema{
		Tables: []*Table{
			{
				Name: "users",
				Columns: []*Column{
					{Name: "id", Type: "integer", Tags: map[string]string{"primary_key": "", "autoincrement": ""}},
					{Name: "name", Type: "varchar", Tags: map[string]string{"notnull": ""}},
				},
			},
		},
	}
	data, _ := json.Marshal(schema)
	fmt.Print(string(data))
}`)

	if err := os.WriteFile(goFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	modFile := filepath.Join(srcDir, "go.mod")
	modContent := []byte(`module schema

go 1.21
`)
	if err := os.WriteFile(modFile, modContent, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	migrationsDir := filepath.Join(tmpDir, "migrations")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		checkFn func(t *testing.T, tmpDir string)
	}{
		{
			name: "Generate SQL",
			args: []string{"-schema", goFile, "-output", migrationsDir, "-format", "sql"},
			checkFn: func(t *testing.T, tmpDir string) {
				files, err := os.ReadDir(migrationsDir)
				if err != nil {
					t.Fatal(err)
				}
				if len(files) != 1 {
					t.Errorf("Expected 1 migration file, got %d", len(files))
				}
			},
		},
		{
			name: "Generate JSON",
			args: []string{"-schema", goFile, "-output", migrationsDir, "-format", "json"},
			checkFn: func(t *testing.T, tmpDir string) {
				// Clean up migrations directory first
				if err := os.RemoveAll(migrationsDir); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(migrationsDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Run test
				oldArgs := os.Args
				os.Args = append([]string{"datara"}, []string{"-schema", goFile, "-output", migrationsDir, "-format", "json"}...)
				defer func() { os.Args = oldArgs }()

				if err := run(); err != nil {
					t.Fatal(err)
				}

				// Check files
				files, err := os.ReadDir(migrationsDir)
				if err != nil {
					t.Fatal(err)
				}
				if len(files) != 1 {
					t.Errorf("Expected 1 JSON file, got %d", len(files))
				}
			},
		},
		{
			name:    "Invalid format",
			args:    []string{"-schema", goFile, "-output", migrationsDir, "-format", "invalid"},
			wantErr: true,
		},
		{
			name:    "Missing schema",
			args:    []string{"-output", migrationsDir, "-format", "sql"},
			wantErr: true,
		},
		{
			name:    "Missing output",
			args:    []string{"-schema", goFile, "-format", "sql"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up migrations directory before each test
			if err := os.RemoveAll(migrationsDir); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(migrationsDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Set command line arguments
			oldArgs := os.Args
			os.Args = append([]string{"datara"}, tt.args...)
			defer func() { os.Args = oldArgs }()

			// Run main
			err := run()
			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run checks if no error expected
			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, tmpDir)
			}
		})
	}
}
