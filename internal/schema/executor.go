package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	migrationsDir = "migrations"
	schemaFile    = "migrations/schema.sql"
	hashFile      = "migrations/schema_hash"
)

// Executor menangani eksekusi program schema
type Executor struct {
	program []string
}

// NewExecutor membuat instance baru dari Executor
func NewExecutor(program []string) *Executor {
	return &Executor{
		program: program,
	}
}

// Execute menjalankan program schema dan mengembalikan SQL statements
func (e *Executor) Execute() (string, error) {
	log.Printf("Starting schema execution with program: %v", e.program)

	// Pastikan direktori migrations ada
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory: %w", err)
	}
	log.Printf("Migrations directory ensured: %s", migrationsDir)

	// Simpan current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Pastikan path ke register.go relatif terhadap lokasi datara.hcl
	registerPath := e.program[len(e.program)-1]
	if !filepath.IsAbs(registerPath) {
		registerPath = filepath.Join(currentDir, registerPath)
	}
	e.program[len(e.program)-1] = registerPath
	log.Printf("Using register file: %s", registerPath)

	// Execute program
	cmd := exec.Command(e.program[0], e.program[1:]...)
	cmd.Env = os.Environ()               // Pass environment variables
	cmd.Dir = filepath.Dir(registerPath) // Set working directory ke lokasi register.go

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("schema program failed: %s\n%s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("failed to execute schema program: %w", err)
	}
	log.Printf("Successfully executed schema program")

	// Format output untuk konsistensi
	newSchema := strings.TrimSpace(string(output))
	if newSchema == "" {
		log.Printf("No schema output received")
		return "", nil
	}

	// Bersihkan output dari karakter tidak perlu
	newSchema = cleanOutput(newSchema)

	// Format SQL untuk readability
	newSchema = formatSQL(newSchema)
	log.Printf("Formatted new schema (length: %d chars)", len(newSchema))

	// Baca schema lama
	oldSchema, err := os.ReadFile(schemaFile)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}

	// Jika tidak ada schema lama, ini adalah migration pertama
	if os.IsNotExist(err) {
		log.Printf("No previous schema found, this is the first migration")
		// Simpan schema baru
		if err := saveSchemaState(newSchema); err != nil {
			return "", fmt.Errorf("failed to save schema state: %w", err)
		}
		return formatMigration(
			newSchema,
			"DROP TABLE IF EXISTS \"profiles\" CASCADE;\nDROP TABLE IF EXISTS \"users\" CASCADE;",
		), nil
	}

	log.Printf("Found existing schema (length: %d chars)", len(oldSchema))

	// Generate diff antara schema lama dan baru
	upSQL, downSQL, err := generateSchemaDiff(string(oldSchema), newSchema)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema diff: %w", err)
	}

	// Jika tidak ada perubahan, return empty
	if upSQL == "" {
		return "", nil
	}

	// Format migration dengan up dan down
	migration := formatMigration(upSQL, downSQL)

	// Simpan schema baru
	if err := saveSchemaState(newSchema); err != nil {
		return "", fmt.Errorf("failed to save schema state: %w", err)
	}

	return migration, nil
}

// formatMigration memformat migration dengan up dan down statements
func formatMigration(upSQL, downSQL string) string {
	return fmt.Sprintf("-- migrate:up\n\n%s\n\n-- migrate:down\n\n%s", upSQL, downSQL)
}

// generateSchemaDiff membandingkan dua schema dan menghasilkan ALTER statements
func generateSchemaDiff(oldSchema, newSchema string) (string, string, error) {
	log.Printf("Generating schema diff")

	// Parse schema lama dan baru
	oldTables := parseTables(oldSchema)
	newTables := parseTables(newSchema)

	log.Printf("Found tables - Old: %d, New: %d", len(oldTables), len(newTables))

	var upStatements, downStatements []string

	// 1. Handle dropped tables
	for tableName := range oldTables {
		if _, exists := newTables[tableName]; !exists {
			log.Printf("Table dropped: %s", tableName)
			// Down: Create table
			downStatements = append(downStatements, oldTables[tableName])

			// Up: Drop table
			upStatements = append(upStatements, fmt.Sprintf("DROP TABLE IF EXISTS %q CASCADE", tableName))
		}
	}

	// 2. Handle new tables
	for tableName, newTable := range newTables {
		if _, exists := oldTables[tableName]; !exists {
			log.Printf("New table added: %s", tableName)
			// Down: Drop table
			downStatements = append(downStatements, fmt.Sprintf("DROP TABLE IF EXISTS %q CASCADE", tableName))

			// Up: Create table
			upStatements = append(upStatements, newTable)
		}
	}

	// 3. Handle modified tables
	for tableName, newTable := range newTables {
		oldTable, exists := oldTables[tableName]
		if !exists {
			continue // New table, already handled
		}

		// Compare and generate ALTER TABLE statements
		upStmts, downStmts := compareTableDefinitions(tableName, oldTable, newTable)
		if len(upStmts) > 0 {
			log.Printf("Table modified: %s (%d changes)", tableName, len(upStmts))
			upStatements = append(upStatements, upStmts...)
			downStatements = append(downStatements, downStmts...)
		}
	}

	if len(upStatements) == 0 {
		log.Printf("No changes detected in schema diff")
		return "", "", nil
	}

	// Format statements
	upSQL := strings.Join(upStatements, ";\n") + ";"
	downSQL := strings.Join(downStatements, ";\n") + ";"

	log.Printf("[TRACE] Generated upSQL in diff: %s", upSQL)
	log.Printf("[TRACE] Generated downSQL in diff: %s", downSQL)

	return upSQL, downSQL, nil
}

// parseTables mengekstrak definisi tabel dari schema SQL
func parseTables(schema string) map[string]string {
	tables := make(map[string]string)
	statements := strings.Split(schema, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if strings.HasPrefix(stmt, "CREATE TABLE") {
			// Extract table name
			tableName := extractTableName(stmt)
			if tableName != "" {
				tables[tableName] = stmt
			}
		}
	}

	return tables
}

// extractTableName mengekstrak nama tabel dari CREATE TABLE statement
func extractTableName(stmt string) string {
	parts := strings.Split(stmt, " ")
	for i, part := range parts {
		if part == "TABLE" && i+1 < len(parts) {
			// Remove quotes and any trailing characters
			name := strings.Trim(parts[i+1], `"() `)
			return name
		}
	}
	return ""
}

// compareTableDefinitions membandingkan dua definisi tabel dan menghasilkan ALTER statements
func compareTableDefinitions(tableName, oldDef, newDef string) ([]string, []string) {
	var upStatements, downStatements []string

	// Parse column definitions
	oldColumns := parseColumns(oldDef)
	newColumns := parseColumns(newDef)

	log.Printf("Comparing table %q - Old columns: %d, New columns: %d",
		tableName, len(oldColumns), len(newColumns))

	// 1. Handle dropped columns
	for colName := range oldColumns {
		if _, exists := newColumns[colName]; !exists {
			log.Printf("Column dropped from %q: %s", tableName, colName)
			// Down: Add column back
			stmt := fmt.Sprintf("ALTER TABLE %q ADD COLUMN %s", tableName, oldColumns[colName])
			downStatements = append(downStatements, stmt)

			// Up: Drop column
			stmt = fmt.Sprintf("ALTER TABLE %q DROP COLUMN %q", tableName, colName)
			upStatements = append(upStatements, stmt)
		}
	}

	// 2. Handle new columns
	for colName, colDef := range newColumns {
		if _, exists := oldColumns[colName]; !exists {
			log.Printf("New column added to %q: %s", tableName, colName)
			// Down: Drop column
			stmt := fmt.Sprintf("ALTER TABLE %q DROP COLUMN %q", tableName, colName)
			downStatements = append(downStatements, stmt)

			// Up: Add column
			// Bersihkan definisi kolom dari karakter yang tidak perlu
			colDef = cleanColumnDef(colDef)
			stmt = fmt.Sprintf("ALTER TABLE %q ADD COLUMN %s", tableName, colDef)
			upStatements = append(upStatements, stmt)
		}
	}

	// 3. Handle modified columns
	for colName, newColDef := range newColumns {
		oldColDef, exists := oldColumns[colName]
		if !exists {
			continue // New column, already handled
		}

		if oldColDef != newColDef {
			log.Printf("Column modified in %q: %s", tableName, colName)
			// Extract type from column definition
			newType := extractColumnType(newColDef)
			oldType := extractColumnType(oldColDef)

			// Down: Restore old type
			downStmt := fmt.Sprintf("ALTER TABLE %q ALTER COLUMN %q TYPE %s", tableName, colName, oldType)
			downStatements = append(downStatements, downStmt)

			// Up: Apply new type
			upStmt := fmt.Sprintf("ALTER TABLE %q ALTER COLUMN %q TYPE %s", tableName, colName, newType)
			upStatements = append(upStatements, upStmt)
		}
	}

	return upStatements, downStatements
}

// cleanColumnDef membersihkan definisi kolom dari karakter yang tidak perlu
func cleanColumnDef(def string) string {
	// Hapus karakter yang tidak perlu
	def = strings.TrimSpace(def)
	def = strings.TrimRight(def, ";")

	// Perbaiki format decimal
	if strings.Contains(def, "decimal(") {
		// Ekstrak nama kolom
		parts := strings.SplitN(def, " ", 2)
		if len(parts) == 2 {
			colName := parts[0]
			rest := parts[1]

			// Perbaiki format decimal
			rest = strings.Replace(rest, "decimal(", "decimal (", -1)
			rest = strings.Replace(rest, ",", ", ", -1)

			def = colName + " " + rest
		}
	}

	return def
}

// parseColumns mengekstrak definisi kolom dari CREATE TABLE statement
func parseColumns(tableDef string) map[string]string {
	columns := make(map[string]string)

	// Extract content between parentheses
	start := strings.Index(tableDef, "(")
	end := strings.LastIndex(tableDef, ")")
	if start == -1 || end == -1 {
		return columns
	}

	// Split dengan mempertahankan tanda kurung
	columnDefs := splitKeepingParentheses(tableDef[start+1 : end])
	for _, colDef := range columnDefs {
		colDef = strings.TrimSpace(colDef)
		if colDef == "" || strings.HasPrefix(colDef, "PRIMARY KEY") {
			continue
		}

		// Handle decimal type dengan precision
		if strings.Contains(colDef, "decimal(") {
			// Ekstrak nama kolom dan tipe data
			parts := strings.SplitN(colDef, " ", 2)
			if len(parts) == 2 {
				colName := strings.Trim(parts[0], `"`)
				rest := parts[1]

				// Ekstrak precision dan scale
				precisionStart := strings.Index(rest, "(")
				precisionEnd := strings.LastIndex(rest, ")")
				if precisionStart != -1 && precisionEnd != -1 {
					precision := rest[precisionStart+1 : precisionEnd]
					precisionParts := strings.Split(precision, ",")

					if len(precisionParts) == 2 {
						p := strings.TrimSpace(precisionParts[0])
						s := strings.TrimSpace(precisionParts[1])

						// Tambahkan default value jika ada
						defaultValue := ""
						if precisionEnd+1 < len(rest) {
							defaultValue = strings.TrimSpace(rest[precisionEnd+1:])
						}

						rest = fmt.Sprintf("decimal(%s,%s) %s", p, s, defaultValue)
					}
				}

				columns[colName] = colName + " " + rest
				continue
			}
		}

		parts := strings.SplitN(colDef, " ", 2)
		if len(parts) == 2 {
			name := strings.Trim(parts[0], `"`)
			columns[name] = colDef
		}
	}

	return columns
}

// splitKeepingParentheses memisahkan string dengan koma tapi mempertahankan tanda kurung
func splitKeepingParentheses(s string) []string {
	var result []string
	var current strings.Builder
	parenCount := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			parenCount++
			current.WriteByte(s[i])
		case ')':
			parenCount--
			current.WriteByte(s[i])
		case ',':
			if parenCount == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteByte(s[i])
			}
		default:
			current.WriteByte(s[i])
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// extractColumnType mengekstrak tipe data dari definisi kolom
func extractColumnType(colDef string) string {
	parts := strings.SplitN(colDef, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	// Ambil bagian setelah nama kolom dan hapus constraint
	typeParts := strings.Split(parts[1], " ")
	return typeParts[0]
}

// cleanOutput membersihkan output dari karakter tidak perlu
func cleanOutput(sql string) string {
	// Hapus karakter % di akhir dan whitespace
	sql = strings.TrimRight(sql, "% \t\n\r")

	// Hapus whitespace berlebih di setiap baris
	lines := strings.Split(sql, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

// calculateHash menghitung hash SHA-256 dari string
func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// saveSchemaState menyimpan state schema ke file
func saveSchemaState(schema string) error {
	// Simpan schema
	if err := os.WriteFile(schemaFile, []byte(schema), 0644); err != nil {
		return fmt.Errorf("failed to save schema file: %w", err)
	}

	// Hitung dan simpan hash
	hash := calculateHash(normalizeSchema(schema))
	if err := os.WriteFile(hashFile, []byte(hash), 0644); err != nil {
		return fmt.Errorf("failed to save hash file: %w", err)
	}

	return nil
}

// formatSQL memformat SQL untuk readability
func formatSQL(sql string) string {
	// Split statements
	statements := strings.Split(sql, ";")
	var formatted []string

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Format berdasarkan tipe statement
		if strings.HasPrefix(stmt, "CREATE TABLE") {
			stmt = formatCreateTable(stmt)
		} else if strings.HasPrefix(stmt, "CREATE") {
			stmt = formatCreateIndex(stmt)
		}

		formatted = append(formatted, stmt)
	}

	return strings.Join(formatted, ";\n\n") + ";"
}

// formatCreateTable memformat CREATE TABLE statement
func formatCreateTable(sql string) string {
	// Extract table name dan column definitions
	parts := strings.SplitN(sql, "(", 2)
	if len(parts) != 2 {
		return sql
	}

	tableDef := strings.TrimSpace(parts[0])
	columnDefs := strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))

	// Format column definitions
	columns := strings.Split(columnDefs, ",")
	var formattedColumns []string
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if strings.HasPrefix(col, "PRIMARY KEY") {
			formattedColumns = append(formattedColumns, "  "+col)
		} else {
			formattedColumns = append(formattedColumns, "  "+col)
		}
	}

	return fmt.Sprintf("%s (\n%s\n)", tableDef, strings.Join(formattedColumns, ",\n"))
}

// formatCreateIndex memformat CREATE INDEX statement
func formatCreateIndex(sql string) string {
	return strings.TrimSpace(sql)
}

// normalizeSchema menormalkan schema untuk perbandingan yang konsisten
func normalizeSchema(schema string) string {
	// Split menjadi statements individual
	statements := strings.Split(schema, ";")
	var normalized []string

	// Group statements berdasarkan tipe
	var creates, indexes, others []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		switch {
		case strings.HasPrefix(stmt, "CREATE TABLE"):
			creates = append(creates, stmt)
		case strings.HasPrefix(stmt, "CREATE UNIQUE INDEX") || strings.HasPrefix(stmt, "CREATE INDEX"):
			indexes = append(indexes, stmt)
		default:
			others = append(others, stmt)
		}
	}

	// Sort setiap group
	sort.Strings(creates)
	sort.Strings(indexes)
	sort.Strings(others)

	// Gabungkan kembali dalam urutan yang konsisten
	normalized = append(normalized, creates...)
	normalized = append(normalized, indexes...)
	normalized = append(normalized, others...)

	return strings.Join(normalized, ";\n") + ";"
}
