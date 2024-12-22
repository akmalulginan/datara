package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/akmalulginan/datara/internal/schema"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config adalah struktur untuk konfigurasi dari datara.hcl
type Config struct {
	Schema struct {
		Program []string `hcl:"program"`
	} `hcl:"schema,block"`
	Migration struct {
		Dir    string `hcl:"dir"`
		Format string `hcl:"format,optional"`
	} `hcl:"migration,block"`
	Naming struct {
		Table struct {
			Plural    bool `hcl:"plural,optional"`
			SnakeCase bool `hcl:"snake_case,optional"`
		} `hcl:"table,block"`
		Column struct {
			SnakeCase bool `hcl:"snake_case,optional"`
		} `hcl:"column,block"`
	} `hcl:"naming,block"`
}

func main() {
	var cmd string
	flag.StringVar(&cmd, "cmd", "diff", "Command to execute (diff)")
	flag.Parse()

	switch cmd {
	case "diff":
		if err := generateDiff(); err != nil {
			fmt.Printf("Error generating diff: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command. Available commands: diff")
		os.Exit(1)
	}
}

func generateDiff() error {
	// 1. Baca konfigurasi
	config, err := readConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// 2. Execute program untuk mendapatkan schema
	executor := schema.NewExecutor(config.Schema.Program)
	desiredSchema, err := executor.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute schema program: %w", err)
	}

	// Jika tidak ada perubahan, keluar
	if desiredSchema == "" {
		fmt.Println("No changes detected")
		return nil
	}

	// 3. Generate migration file
	if err := generateMigrationFile(desiredSchema, config.Migration.Dir); err != nil {
		return fmt.Errorf("failed to generate migration file: %w", err)
	}

	fmt.Println("Generated new migration")
	return nil
}

func readConfig() (*Config, error) {
	var config Config
	if err := hclsimple.DecodeFile("datara.hcl", nil, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func generateMigrationFile(sql, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := filepath.Join(dir, fmt.Sprintf("%s.sql", timestamp))

	// Tulis file langsung tanpa menambahkan marker
	if err := os.WriteFile(filename, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("Generated migration file: %s\n", filename)
	return nil
}

// generateDownMigration menghasilkan SQL untuk rollback
func generateDownMigration(upSQL string) string {
	var drops []string
	tablePattern := regexp.MustCompile(`CREATE TABLE "([^"]+)"`)

	// Find all table names
	matches := tablePattern.FindAllStringSubmatch(upSQL, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tableName := match[1]
			drops = append(drops, fmt.Sprintf(`DROP TABLE IF EXISTS %q CASCADE;`, tableName))
		}
	}

	// Reverse order untuk drop tables (handle dependencies)
	for i := len(drops)/2 - 1; i >= 0; i-- {
		opp := len(drops) - 1 - i
		drops[i], drops[opp] = drops[opp], drops[i]
	}

	return strings.Join(drops, "\n")
}
