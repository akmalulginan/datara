package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/akmalulginan/datara"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config adalah struktur untuk konfigurasi dari datara.hcl
type Config struct {
	Schema struct {
		Program []string `hcl:"program"`
	} `hcl:"schema,block"`
	Migration struct {
		Dir       string `hcl:"dir"`
		Format    string `hcl:"format,optional"`
		Charset   string `hcl:"charset,optional"`
		Collation string `hcl:"collation,optional"`
		Engine    string `hcl:"engine,optional"`
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

// DefaultConfig mengembalikan konfigurasi default
func DefaultConfig() *Config {
	config := &Config{}
	config.Migration.Dir = "migrations"
	config.Migration.Format = "sql"
	config.Migration.Charset = "utf8mb4"
	config.Migration.Collation = "utf8mb4_unicode_ci"
	config.Migration.Engine = "InnoDB"
	config.Naming.Table.Plural = true
	config.Naming.Table.SnakeCase = true
	config.Naming.Column.SnakeCase = true
	return config
}

func loadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		// Cari datara.hcl di direktori saat ini
		if _, err := os.Stat("datara.hcl"); err == nil {
			configPath = "datara.hcl"
		} else {
			return config, nil // Gunakan default jika tidak ada file konfigurasi
		}
	}

	if err := hclsimple.DecodeFile(configPath, nil, config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return config, nil
}

// calculateFileHash menghitung hash SHA-256 dari file
func calculateFileHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("h1:%s", base64.StdEncoding.EncodeToString(hash[:]))
}

// readChecksumFile membaca file datara.sum
func readChecksumFile(config *Config) (map[string]string, error) {
	checksums := make(map[string]string)

	checksumPath := filepath.Join(config.Migration.Dir, "datara.sum")
	file, err := os.Open(checksumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return checksums, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	firstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if firstLine {
			// Skip global hash
			firstLine = false
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksums[parts[0]] = parts[1]
		}
	}

	return checksums, scanner.Err()
}

// calculateGlobalHash menghitung hash dari semua file migrasi
func calculateGlobalHash(config *Config) (string, error) {
	// Baca semua file migrasi
	files, err := os.ReadDir(config.Migration.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return calculateFileHash([]byte("")), nil
		}
		return "", err
	}

	// Urutkan file untuk konsistensi
	var filenames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			filenames = append(filenames, file.Name())
		}
	}
	sort.Strings(filenames)

	// Gabungkan semua konten file
	var allContent []byte
	for _, filename := range filenames {
		content, err := os.ReadFile(filepath.Join(config.Migration.Dir, filename))
		if err != nil {
			return "", err
		}
		allContent = append(allContent, content...)
	}

	return calculateFileHash(allContent), nil
}

// writeChecksumFile menulis file datara.sum
func writeChecksumFile(checksums map[string]string, globalHash string, config *Config) error {
	// Sort filenames untuk konsistensi
	var filenames []string
	for filename := range checksums {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	// Buat atau update file datara.sum
	checksumPath := filepath.Join(config.Migration.Dir, "datara.sum")
	file, err := os.Create(checksumPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Tulis global hash
	if _, err := fmt.Fprintf(file, "%s\n", globalHash); err != nil {
		return err
	}

	// Tulis checksums dalam format yang diurutkan
	for _, filename := range filenames {
		if _, err := fmt.Fprintf(file, "%s %s\n", filename, checksums[filename]); err != nil {
			return err
		}
	}

	return nil
}

// updateChecksums memperbarui datara.sum dengan file migrasi baru
func updateChecksums(newFile string, content []byte, config *Config) error {
	// Baca checksums yang ada
	checksums, err := readChecksumFile(config)
	if err != nil {
		return fmt.Errorf("failed to read checksum file: %v", err)
	}

	// Hitung dan tambahkan hash baru
	filename := filepath.Base(newFile)
	checksums[filename] = calculateFileHash(content)

	// Hitung global hash
	globalHash, err := calculateGlobalHash(config)
	if err != nil {
		return fmt.Errorf("failed to calculate global hash: %v", err)
	}

	// Tulis kembali file checksum
	return writeChecksumFile(checksums, globalHash, config)
}

// calculateSQLHash menghitung hash dari SQL yang akan dibuat
func calculateSQLHash(sql string) string {
	return calculateFileHash([]byte(sql))
}

// getLastMigrationHash mendapatkan hash dari migrasi terakhir
func getLastMigrationHash(config *Config) (string, error) {
	checksums, err := readChecksumFile(config)
	if err != nil {
		return "", err
	}

	// Cari file migrasi terakhir
	var lastFile string
	for filename := range checksums {
		if strings.HasSuffix(filename, ".sql") {
			if lastFile == "" || filename > lastFile {
				lastFile = filename
			}
		}
	}

	if lastFile == "" {
		return "", nil
	}

	// Ambil hash dari datara.sum
	if hash, ok := checksums[lastFile]; ok {
		return hash, nil
	}

	return "", fmt.Errorf("hash not found for last migration")
}

// saveLastSchema menyimpan skema terakhir ke file
func saveLastSchema(schema *datara.Schema, config *Config) error {
	// Buat direktori migrations jika belum ada
	if err := os.MkdirAll(config.Migration.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %v", err)
	}

	// Simpan skema ke file
	schemaPath := filepath.Join(config.Migration.Dir, "datara.schema")
	sql := schema.ToSQL()
	if err := os.WriteFile(schemaPath, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %v", err)
	}

	return nil
}

// loadLastSchema membaca skema terakhir dari file
func loadLastSchema(config *Config) (*datara.Schema, error) {
	schemaPath := filepath.Join(config.Migration.Dir, "datara.schema")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Jika file belum ada, kembalikan skema kosong
			return &datara.Schema{
				Tables: make([]*datara.Table, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to read schema file: %v", err)
	}

	// Jika file kosong, kembalikan skema kosong
	if len(content) == 0 {
		return &datara.Schema{
			Tables: make([]*datara.Table, 0),
		}, nil
	}

	// Parse SQL menjadi skema
	return datara.FromSQL(string(content)), nil
}

// executeSchemaProgram menjalankan program untuk mendapatkan skema
func executeSchemaProgram(program []string) (string, error) {
	fmt.Println("=== executeSchemaProgram ===")
	fmt.Printf("Program: %v\n", program)

	if len(program) == 0 {
		return "", fmt.Errorf("no schema program specified")
	}

	// Simpan current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %v", err)
	}
	fmt.Printf("Current directory: %s\n", currentDir)

	// Pastikan path ke register.go relatif terhadap lokasi datara.hcl
	registerPath := program[len(program)-1]
	if !filepath.IsAbs(registerPath) {
		registerPath = filepath.Join(currentDir, registerPath)
	}
	program[len(program)-1] = registerPath
	fmt.Printf("Register path: %s\n", registerPath)

	// Execute program
	cmd := exec.Command(program[0], program[1:]...)
	cmd.Env = os.Environ()               // Pass environment variables
	cmd.Dir = filepath.Dir(registerPath) // Set working directory ke lokasi register.go

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("schema program failed: %s\n%s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("failed to execute schema program: %v", err)
	}

	fmt.Println("\nOutput from register.go:")
	fmt.Println(string(output))
	fmt.Println("=== End executeSchemaProgram ===\n")

	return string(output), nil
}

// generateMigration membuat file migrasi baru
func generateMigration(sql string, config *Config) error {
	fmt.Println("=== generateMigration ===")
	fmt.Println("Input SQL:")
	fmt.Println(sql)

	// Buat direktori migrations jika belum ada
	if err := os.MkdirAll(config.Migration.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %v", err)
	}

	// Generate nama file migrasi dengan timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := filepath.Join(config.Migration.Dir, timestamp+".sql")
	fmt.Printf("Migration file: %s\n", filename)

	// Tulis file migrasi
	if err := os.WriteFile(filename, []byte(sql), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %v", err)
	}

	// Update checksums
	if err := updateChecksums(filename, []byte(sql), config); err != nil {
		return fmt.Errorf("failed to update checksums: %v", err)
	}

	fmt.Printf("Generated migration file: %s\n", filename)
	fmt.Println("=== End generateMigration ===\n")
	return nil
}

// run adalah fungsi utama yang menjalankan program
func run() error {
	fmt.Println("=== Starting datara ===")
	// Parse command line flags
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load config
	config, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	fmt.Printf("Config loaded: %+v\n", config)

	// Jalankan program untuk mendapatkan skema
	sql, err := executeSchemaProgram(config.Schema.Program)
	if err != nil {
		return err
	}

	// Generate file migrasi
	if err := generateMigration(sql, config); err != nil {
		return err
	}

	fmt.Println("=== datara completed successfully ===")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
