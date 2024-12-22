package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/akmalulginan/datara"
)

type MigrationConfig struct {
	Dir     string `hcl:"dir"`
	Format  string `hcl:"format"`
	Charset string `hcl:"charset"`
	Engine  string `hcl:"engine"`
}

type Config struct {
	Migration MigrationConfig `hcl:"migration"`
}

func DefaultConfig() *Config {
	return &Config{
		Migration: MigrationConfig{
			Dir:     "migrations",
			Format:  "sql",
			Charset: "utf8mb4",
			Engine:  "InnoDB",
		},
	}
}

func generateSQL(schema *datara.Schema, config *Config) error {
	// Create output directory if not exists
	if err := os.MkdirAll(config.Migration.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate migration file name
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("v%s_migration.sql", timestamp)
	outputFile := filepath.Join(config.Migration.Dir, filename)

	// Generate SQL content
	sql := generateSQLContent(schema)

	// Write to file
	if err := os.WriteFile(outputFile, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %v", err)
	}

	fmt.Printf("Generated migration file: %s\n", outputFile)
	return nil
}

func generateJSON(schema *datara.Schema, config *Config) error {
	// Create output directory if not exists
	if err := os.MkdirAll(config.Migration.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate JSON file
	outputFile := filepath.Join(config.Migration.Dir, "schema.json")
	jsonSchema := generateJSONContent(schema)

	// Write to file
	if err := os.WriteFile(outputFile, []byte(jsonSchema), 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	fmt.Printf("Generated JSON file: %s\n", outputFile)
	return nil
}

func executeSchemaProgram(args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute schema program: %v", err)
	}
	return string(output), nil
}

func generateSQLContent(schema *datara.Schema) string {
	var sql string
	for _, table := range schema.Tables {
		sql += fmt.Sprintf("CREATE TABLE %s (\n", table.Name)
		for i, col := range table.Columns {
			sql += fmt.Sprintf("  %s %s", col.Name, col.Type)
			if i < len(table.Columns)-1 {
				sql += ",\n"
			}
		}
		sql += "\n);\n\n"
	}
	return sql
}

func generateJSONContent(schema *datara.Schema) string {
	data, _ := json.MarshalIndent(schema, "", "  ")
	return string(data)
}

func run() error {
	// Create new flag set
	fs := flag.NewFlagSet("datara", flag.ExitOnError)

	// Define flags
	schemaFile := fs.String("schema", "", "Path to schema Go file")
	outputDir := fs.String("output", "", "Output directory for generated files")
	format := fs.String("format", "sql", "Output format (sql or json)")

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	// Validate flags
	if *schemaFile == "" {
		return fmt.Errorf("schema file is required")
	}
	if *outputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	// Execute schema program
	schema, err := executeSchemaProgram([]string{"go", "run", *schemaFile})
	if err != nil {
		return err
	}

	// Parse schema JSON
	var schemaObj datara.Schema
	if err := json.Unmarshal([]byte(schema), &schemaObj); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %v", err)
	}

	// Create config
	config := &Config{
		Migration: MigrationConfig{
			Dir:    *outputDir,
			Format: *format,
		},
	}

	// Generate output based on format
	switch *format {
	case "sql":
		return generateSQL(&schemaObj, config)
	case "json":
		return generateJSON(&schemaObj, config)
	default:
		return fmt.Errorf("unsupported format: %s", *format)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
