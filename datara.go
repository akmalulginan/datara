package datara

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Schema represents a database schema
type Schema struct {
	Tables []*Table
}

// Table represents a database table
type Table struct {
	Name        string
	Columns     []*Column
	Indexes     []*Index
	ForeignKeys []*ForeignKey
	PrimaryKey  *PrimaryKey
}

// Column represents a table column
type Column struct {
	Name          string
	Type          string
	Length        int
	Nullable      bool
	Default       interface{}
	AutoIncrement bool
	IsPrimaryKey  bool
	IsUnique      bool
}

// Index represents a table index
type Index struct {
	Name    string
	Columns []string
	Type    string
	Unique  bool
}

// PrimaryKey represents a primary key constraint
type PrimaryKey struct {
	Name    string
	Columns []string
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name             string
	Columns          []string
	ReferenceTable   string
	ReferenceColumns []string
	OnDelete         string
	OnUpdate         string
}

// SQLType merepresentasikan tipe data SQL
type SQLType struct {
	Name      string
	Length    int
	Precision int
	Scale     int
	Unsigned  bool
}

// EnumType merepresentasikan tipe enum SQL
type EnumType struct {
	Name   string
	Values []string
}

// ForeignKeyReference merepresentasikan referensi foreign key
type ForeignKeyReference struct {
	Table    string
	Columns  []string
	OnDelete string
	OnUpdate string
}

// ValidateSQLType memvalidasi tipe data SQL
func ValidateSQLType(sqlType string) (*SQLType, error) {
	// Normalisasi tipe data
	sqlType = strings.ToUpper(strings.TrimSpace(sqlType))

	// Parse tipe data dasar
	result := &SQLType{}

	// Handle unsigned
	if strings.Contains(sqlType, "UNSIGNED") {
		result.Unsigned = true
		sqlType = strings.ReplaceAll(sqlType, "UNSIGNED", "")
		sqlType = strings.TrimSpace(sqlType)
	}

	// Handle tipe data dengan parameter
	if strings.Contains(sqlType, "(") {
		base := sqlType[:strings.Index(sqlType, "(")]
		params := sqlType[strings.Index(sqlType, "(")+1 : strings.Index(sqlType, ")")]

		result.Name = base

		// Parse parameter
		if strings.Contains(params, ",") {
			// Format: DECIMAL(10,2)
			parts := strings.Split(params, ",")
			if len(parts) == 2 {
				if p, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
					result.Precision = p
				}
				if s, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					result.Scale = s
				}
			}
		} else {
			// Format: VARCHAR(255)
			if l, err := strconv.Atoi(params); err == nil {
				result.Length = l
			}
		}
	} else {
		result.Name = sqlType
	}

	// Validasi dan set nilai default
	switch result.Name {
	case "TINYINT":
		if result.Length == 0 {
			result.Length = 4
		}
	case "SMALLINT":
		if result.Length == 0 {
			result.Length = 6
		}
	case "MEDIUMINT":
		if result.Length == 0 {
			result.Length = 9
		}
	case "INT", "INTEGER":
		result.Name = "INT"
		if result.Length == 0 {
			result.Length = 11
		}
	case "BIGINT":
		if result.Length == 0 {
			result.Length = 20
		}
	case "DECIMAL", "NUMERIC":
		result.Name = "DECIMAL"
		if result.Precision == 0 {
			result.Precision = 10
		}
		if result.Scale == 0 {
			result.Scale = 2
		}
	case "FLOAT":
		if result.Precision == 0 {
			result.Precision = 10
		}
	case "DOUBLE", "REAL":
		result.Name = "DOUBLE"
		if result.Precision == 0 {
			result.Precision = 16
		}
	case "VARCHAR", "CHAR":
		if result.Length == 0 {
			if result.Name == "VARCHAR" {
				result.Length = 255
			} else {
				result.Length = 1
			}
		}
	case "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
		result.Length = 0 // Tidak perlu length
	case "DATETIME", "TIMESTAMP", "DATE", "TIME":
		result.Length = 0 // Tidak perlu length
	case "BOOLEAN", "BOOL":
		result.Name = "TINYINT"
		result.Length = 1
	case "ENUM":
		// Enum dihandle secara khusus
		return result, nil
	case "JSON":
		result.Length = 0 // Tidak perlu length
	default:
		return nil, fmt.Errorf("tipe data tidak didukung: %s", sqlType)
	}

	return result, nil
}

// String menghasilkan representasi string dari tipe data SQL
func (t *SQLType) String() string {
	var result strings.Builder

	result.WriteString(t.Name)

	// Tambahkan parameter jika diperlukan
	switch t.Name {
	case "DECIMAL", "NUMERIC":
		if t.Precision > 0 && t.Scale > 0 {
			result.WriteString(fmt.Sprintf("(%d,%d)", t.Precision, t.Scale))
		}
	case "FLOAT", "DOUBLE":
		if t.Precision > 0 {
			result.WriteString(fmt.Sprintf("(%d)", t.Precision))
		}
	case "VARCHAR", "CHAR":
		if t.Length > 0 {
			result.WriteString(fmt.Sprintf("(%d)", t.Length))
		}
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "BIGINT":
		if t.Length > 0 {
			result.WriteString(fmt.Sprintf("(%d)", t.Length))
		}
	}

	// Tambahkan unsigned jika perlu
	if t.Unsigned {
		result.WriteString(" UNSIGNED")
	}

	return result.String()
}

// ParseSchema mengkonversi struct menjadi skema SQL
func ParseSchema(models ...interface{}) *Schema {
	schema := &Schema{
		Tables: make([]*Table, 0),
	}

	for _, model := range models {
		// Jika model adalah string, lewati
		if _, ok := model.(string); ok {
			continue
		}

		// Jika model adalah pointer, ambil nilai aslinya
		val := reflect.ValueOf(model)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Hanya proses jika tipe adalah struct
		if val.Kind() == reflect.Struct {
			table := parseStruct(val.Type())
			schema.Tables = append(schema.Tables, table)
		}
	}

	// Tambahkan relasi antar tabel
	addRelations(schema)

	return schema
}

// handleSpecialFields menangani field-field khusus seperti id, timestamps
func handleSpecialFields(name string) *Column {
	switch name {
	case "id":
		return &Column{
			Name:          "id",
			Type:          "BIGINT",
			AutoIncrement: true,
			IsPrimaryKey:  true,
			Nullable:      false,
		}
	case "created_at":
		return &Column{
			Name:     "created_at",
			Type:     "TIMESTAMP",
			Default:  "CURRENT_TIMESTAMP",
			Nullable: false,
		}
	case "updated_at":
		return &Column{
			Name:     "updated_at",
			Type:     "TIMESTAMP",
			Default:  "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
			Nullable: false,
		}
	case "deleted_at":
		return &Column{
			Name:     "deleted_at",
			Type:     "TIMESTAMP",
			Nullable: true,
			Default:  nil,
		}
	case "last_login_at":
		return &Column{
			Name:     "last_login_at",
			Type:     "TIMESTAMP",
			Nullable: true,
			Default:  nil,
		}
	}
	return nil
}

// handleNumericType menangani tipe data numerik
func handleNumericType(kind reflect.Kind) (string, interface{}) {
	switch kind {
	case reflect.Bool:
		return "TINYINT(1)", 0
	case reflect.Int8:
		return "TINYINT", nil
	case reflect.Int16:
		return "SMALLINT", nil
	case reflect.Int, reflect.Int32:
		return "INT", nil
	case reflect.Int64:
		return "BIGINT", nil
	case reflect.Uint8:
		return "TINYINT UNSIGNED", nil
	case reflect.Uint16:
		return "SMALLINT UNSIGNED", nil
	case reflect.Uint, reflect.Uint32:
		return "INT UNSIGNED", nil
	case reflect.Uint64:
		return "BIGINT UNSIGNED", nil
	case reflect.Float32:
		return "FLOAT", nil
	case reflect.Float64:
		return "DOUBLE", nil
	}
	return "", nil
}

// handleStringType menangani tipe data string dan menentukan karakteristik kolomnya
func handleStringType(name string) (string, bool, bool, interface{}) {
	switch name {
	case "email", "username":
		return "VARCHAR(255)", true, false, nil
	case "password":
		return "VARCHAR(255)", false, false, nil
	case "phone", "phone_number":
		return "VARCHAR(20)", false, true, nil
	case "status", "role":
		return "VARCHAR(50)", false, false, "user"
	case "address", "bio":
		return "TEXT", false, true, nil
	case "avatar", "website", "location":
		return "VARCHAR(255)", false, true, nil
	case "is_active", "is_verified":
		return "TINYINT(1)", false, false, 1
	default:
		if strings.HasSuffix(name, "_id") {
			return "BIGINT UNSIGNED", false, false, nil
		}
		return "VARCHAR(255)", false, false, nil
	}
}

// createColumn membuat instance Column baru dengan konfigurasi dasar
func createColumn(name string, fieldType reflect.Type) *Column {
	column := &Column{
		Name:     toSnakeCase(name),
		Nullable: false,
	}

	// Handle pointer type (nullable)
	if fieldType.Kind() == reflect.Ptr {
		column.Nullable = true
		fieldType = fieldType.Elem()
	}

	return column
}

// setColumnType mengatur tipe data kolom berdasarkan tipe Go
func setColumnType(column *Column, fieldType reflect.Type) {
	// Cek special fields dahulu
	if specialColumn := handleSpecialFields(column.Name); specialColumn != nil {
		*column = *specialColumn
		return
	}

	// Handle numeric types
	if sqlType, defaultVal := handleNumericType(fieldType.Kind()); sqlType != "" {
		column.Type = sqlType
		if defaultVal != nil {
			column.Default = defaultVal
		}
		return
	}

	// Handle string types
	if fieldType.Kind() == reflect.String {
		sqlType, isUnique, isNullable, defaultVal := handleStringType(column.Name)
		column.Type = sqlType
		column.IsUnique = isUnique
		if isNullable {
			column.Nullable = true
		}
		if defaultVal != nil {
			column.Default = defaultVal
		}
		return
	}

	// Handle time.Time
	if fieldType == reflect.TypeOf(time.Time{}) {
		column.Type = "DATETIME"
		column.Nullable = true
		return
	}

	// Default fallback
	column.Type = "VARCHAR(255)"
}

// newColumn membuat kolom baru dari field struct
func newColumn(field reflect.StructField) *Column {
	// Skip jika field private
	if !field.IsExported() {
		return nil
	}

	// Buat kolom dasar
	column := createColumn(field.Name, field.Type)

	// Set tipe kolom
	setColumnType(column, field.Type)

	return column
}

func parseStruct(t reflect.Type) *Table {
	table := &Table{
		Name:        toSnakeCase(t.Name()) + "s",
		Columns:     make([]*Column, 0),
		Indexes:     make([]*Index, 0),
		ForeignKeys: make([]*ForeignKey, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip jika field adalah struct embedded
		if field.Anonymous {
			continue
		}

		column := newColumn(field)
		if column != nil {
			table.Columns = append(table.Columns, column)

			// Tambahkan index untuk kolom yang membutuhkannya
			if column.IsPrimaryKey {
				table.PrimaryKey = &PrimaryKey{
					Name:    fmt.Sprintf("pk_%s", table.Name),
					Columns: []string{column.Name},
				}
			}

			// Tambahkan unique index jika diperlukan
			if column.IsUnique {
				table.Indexes = append(table.Indexes, &Index{
					Name:    fmt.Sprintf("idx_%s_%s_unique", table.Name, column.Name),
					Columns: []string{column.Name},
					Type:    "BTREE",
					Unique:  true,
				})
			}

			// Handle foreign key berdasarkan nama kolom
			if strings.HasSuffix(column.Name, "_id") {
				refTableName := strings.TrimSuffix(column.Name, "_id") + "s"
				fk := &ForeignKey{
					Name:             fmt.Sprintf("fk_%s_%s", table.Name, column.Name),
					Columns:          []string{column.Name},
					ReferenceTable:   refTableName,
					ReferenceColumns: []string{"id"},
					OnDelete:         "RESTRICT",
					OnUpdate:         "RESTRICT",
				}
				table.ForeignKeys = append(table.ForeignKeys, fk)

				// Tambahkan index untuk foreign key
				table.Indexes = append(table.Indexes, &Index{
					Name:    fmt.Sprintf("idx_%s_%s", table.Name, column.Name),
					Columns: []string{column.Name},
					Type:    "BTREE",
				})
			}
		}
	}

	return table
}

// ToSQL menghasilkan SQL untuk membuat tabel (up migration)
func (s *Schema) ToSQL() string {
	var sql strings.Builder

	sql.WriteString("-- migrate:up\n\n")

	for i, table := range s.Tables {
		if i > 0 {
			sql.WriteString("\n\n")
		}

		// Create table
		sql.WriteString(s.createTableSQL(table))
	}

	sql.WriteString("\n\n-- migrate:down\n\n")
	sql.WriteString(s.ToDownSQL())

	return sql.String()
}

// ToDownSQL menghasilkan SQL untuk menghapus tabel (down migration)
func (s *Schema) ToDownSQL() string {
	var sql strings.Builder

	// Drop tabel dalam urutan terbalik untuk menghindari masalah foreign key
	for i := len(s.Tables) - 1; i >= 0; i-- {
		if i < len(s.Tables)-1 {
			sql.WriteString("\n")
		}
		sql.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", s.Tables[i].Name))
	}

	return sql.String()
}

// Helper functions
func toSnakeCase(s string) string {
	// Kasus khusus untuk ID
	if s == "ID" {
		return "id"
	}
	if strings.HasSuffix(s, "ID") {
		return toSnakeCase(strings.TrimSuffix(s, "ID")) + "_id"
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func addRelations(schema *Schema) {
	// Implementasi relasi antar tabel
	for _, table := range schema.Tables {
		for _, fk := range table.ForeignKeys {
			// Tambahkan index untuk foreign key
			table.Indexes = append(table.Indexes, &Index{
				Name:    fmt.Sprintf("idx_%s_%s", table.Name, strings.Join(fk.Columns, "_")),
				Columns: fk.Columns,
				Type:    "BTREE",
			})
		}
	}
}

// CompareSchema membandingkan dua skema dan menghasilkan query ALTER TABLE
func (s *Schema) CompareSchema(old *Schema) string {
	var sql strings.Builder

	// Jika skema lama kosong atau tidak memiliki tabel, buat semua tabel
	if old == nil || len(old.Tables) == 0 {
		return s.ToSQL()
	}

	// Bandingkan setiap tabel
	for _, newTable := range s.Tables {
		// Cari tabel yang sama di skema lama
		var oldTable *Table
		for _, t := range old.Tables {
			if t.Name == newTable.Name {
				oldTable = t
				break
			}
		}

		if oldTable == nil {
			// Tabel baru, buat query CREATE TABLE
			if sql.Len() > 0 {
				sql.WriteString("\n\n")
			}
			sql.WriteString(s.createTableSQL(newTable))
		} else {
			// Bandingkan kolom
			for _, newColumn := range newTable.Columns {
				var found bool
				for _, oldColumn := range oldTable.Columns {
					if oldColumn.Name == newColumn.Name {
						found = true
						break
					}
				}

				if !found {
					// Kolom baru, tambahkan dengan ALTER TABLE
					if sql.Len() > 0 {
						sql.WriteString("\n\n")
					}
					sql.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s",
						newTable.Name, newColumn.Name, newColumn.Type))

					// Add length for string types
					if newColumn.Length > 0 && (strings.Contains(newColumn.Type, "VARCHAR") || strings.Contains(newColumn.Type, "CHAR")) {
						sql.WriteString(fmt.Sprintf("(%d)", newColumn.Length))
					}

					// Add nullable
					if newColumn.Nullable {
						sql.WriteString(" NULL")
					} else {
						sql.WriteString(" NOT NULL")
					}

					// Add default
					if newColumn.Default != nil {
						if str, ok := newColumn.Default.(string); ok {
							if str == "CURRENT_TIMESTAMP" || str == "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" {
								sql.WriteString(fmt.Sprintf(" DEFAULT %s", str))
							} else if str == "true" {
								sql.WriteString(" DEFAULT 1")
							} else if str == "false" {
								sql.WriteString(" DEFAULT 0")
							} else {
								sql.WriteString(fmt.Sprintf(" DEFAULT '%s'", str))
							}
						} else if b, ok := newColumn.Default.(bool); ok {
							if b {
								sql.WriteString(" DEFAULT 1")
							} else {
								sql.WriteString(" DEFAULT 0")
							}
						} else if n, ok := newColumn.Default.(int); ok {
							sql.WriteString(fmt.Sprintf(" DEFAULT %d", n))
						} else if f, ok := newColumn.Default.(float64); ok {
							sql.WriteString(fmt.Sprintf(" DEFAULT %f", f))
						} else {
							sql.WriteString(fmt.Sprintf(" DEFAULT %v", newColumn.Default))
						}
					}

					sql.WriteString(";")

					// Add down migration
					sql.WriteString("\n\n-- migrate:down\n\n")
					sql.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`;", newTable.Name, newColumn.Name))
				}
			}
		}
	}

	return sql.String()
}

// formatColumnDefinition memformat definisi kolom SQL
func formatColumnDefinition(column *Column) string {
	var sql strings.Builder

	// Nama kolom
	sql.WriteString(fmt.Sprintf("`%s` ", column.Name))

	// Tipe data
	sql.WriteString(column.Type)

	// Nullable
	if column.Nullable {
		sql.WriteString(" NULL")
	} else {
		sql.WriteString(" NOT NULL")
	}

	// Auto increment
	if column.AutoIncrement {
		sql.WriteString(" AUTO_INCREMENT")
	}

	// Default value
	if column.Default != nil {
		switch v := column.Default.(type) {
		case string:
			if v == "CURRENT_TIMESTAMP" || strings.Contains(v, "CURRENT_TIMESTAMP ON UPDATE") {
				sql.WriteString(fmt.Sprintf(" DEFAULT %s", v))
			} else {
				sql.WriteString(fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(v, "'", "''")))
			}
		case bool:
			if v {
				sql.WriteString(" DEFAULT 1")
			} else {
				sql.WriteString(" DEFAULT 0")
			}
		case int:
			sql.WriteString(fmt.Sprintf(" DEFAULT %d", v))
		case float64:
			sql.WriteString(fmt.Sprintf(" DEFAULT %f", v))
		}
	}

	return sql.String()
}

// formatConstraints memformat constraint tabel SQL
func formatConstraints(table *Table) string {
	var sql strings.Builder

	// Primary Key
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) > 0 {
		sql.WriteString(",\n  ")
		sql.WriteString(fmt.Sprintf("PRIMARY KEY (`%s`)", strings.Join(table.PrimaryKey.Columns, "`,`")))
	}

	// Indexes
	for _, index := range table.Indexes {
		if len(index.Columns) > 0 {
			sql.WriteString(",\n  ")
			if index.Unique {
				sql.WriteString("UNIQUE ")
			}
			sql.WriteString(fmt.Sprintf("KEY `%s` (`%s`)", index.Name, strings.Join(index.Columns, "`,`")))
		}
	}

	// Foreign Keys
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) > 0 && len(fk.ReferenceColumns) > 0 {
			sql.WriteString(",\n  ")
			sql.WriteString(fmt.Sprintf("CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
				fk.Name,
				strings.Join(fk.Columns, "`,`"),
				fk.ReferenceTable,
				strings.Join(fk.ReferenceColumns, "`,`")))

			if fk.OnDelete != "" {
				sql.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
			}
			if fk.OnUpdate != "" {
				sql.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
			}
		}
	}

	return sql.String()
}

// createTableSQL menghasilkan query CREATE TABLE untuk tabel baru
func (s *Schema) createTableSQL(table *Table) string {
	var sql strings.Builder

	// Header CREATE TABLE
	sql.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", table.Name))

	// Columns
	for i, column := range table.Columns {
		sql.WriteString("  ")
		sql.WriteString(fmt.Sprintf("`%s` ", column.Name))

		// Tipe data
		if strings.HasSuffix(column.Type, "UNSIGNED") {
			sql.WriteString(strings.TrimSuffix(column.Type, " UNSIGNED"))
			sql.WriteString(" UNSIGNED")
		} else {
			sql.WriteString(column.Type)
		}

		// Nullable
		if column.Nullable {
			sql.WriteString(" NULL")
		} else {
			sql.WriteString(" NOT NULL")
		}

		// Auto increment
		if column.AutoIncrement {
			sql.WriteString(" AUTO_INCREMENT")
		}

		// Default value
		if column.Default != nil {
			switch v := column.Default.(type) {
			case string:
				if v == "CURRENT_TIMESTAMP" || strings.Contains(v, "CURRENT_TIMESTAMP ON UPDATE") {
					sql.WriteString(fmt.Sprintf(" DEFAULT %s", v))
				} else {
					sql.WriteString(fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(v, "'", "''")))
				}
			case bool:
				if v {
					sql.WriteString(" DEFAULT 1")
				} else {
					sql.WriteString(" DEFAULT 0")
				}
			case int:
				sql.WriteString(fmt.Sprintf(" DEFAULT %d", v))
			case float64:
				sql.WriteString(fmt.Sprintf(" DEFAULT %f", v))
			}
		}

		if i < len(table.Columns)-1 {
			sql.WriteString(",\n")
		}
	}

	// Primary Key
	if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) > 0 {
		sql.WriteString(",\n  ")
		sql.WriteString(fmt.Sprintf("PRIMARY KEY (`%s`)", strings.Join(table.PrimaryKey.Columns, "`,`")))
	}

	// Unique Indexes
	uniqueIndexes := make([]*Index, 0)
	normalIndexes := make([]*Index, 0)
	for _, index := range table.Indexes {
		if index.Unique {
			uniqueIndexes = append(uniqueIndexes, index)
		} else {
			normalIndexes = append(normalIndexes, index)
		}
	}

	// Add unique indexes first
	addedIndexes := make(map[string]bool)
	for _, index := range uniqueIndexes {
		indexKey := strings.Join(index.Columns, "_")
		if !addedIndexes[indexKey] {
			sql.WriteString(",\n  ")
			sql.WriteString(fmt.Sprintf("UNIQUE KEY `%s` (`%s`)", index.Name, strings.Join(index.Columns, "`,`")))
			addedIndexes[indexKey] = true
		}
	}

	// Add normal indexes
	for _, index := range normalIndexes {
		indexKey := strings.Join(index.Columns, "_")
		if !addedIndexes[indexKey] {
			sql.WriteString(",\n  ")
			sql.WriteString(fmt.Sprintf("KEY `%s` (`%s`)", index.Name, strings.Join(index.Columns, "`,`")))
			addedIndexes[indexKey] = true
		}
	}

	// Foreign Keys
	addedFKs := make(map[string]bool)
	for _, fk := range table.ForeignKeys {
		fkKey := fmt.Sprintf("%s_%s", fk.ReferenceTable, strings.Join(fk.Columns, "_"))
		if !addedFKs[fkKey] {
			sql.WriteString(",\n  ")
			sql.WriteString(fmt.Sprintf("CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
				fk.Name,
				strings.Join(fk.Columns, "`,`"),
				fk.ReferenceTable,
				strings.Join(fk.ReferenceColumns, "`,`")))

			if fk.OnDelete != "" {
				sql.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
			}
			if fk.OnUpdate != "" {
				sql.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
			}
			addedFKs[fkKey] = true
		}
	}

	// Footer
	sql.WriteString("\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;")

	return sql.String()
}

// FromSQL mengkonversi SQL menjadi skema
func FromSQL(sql string) *Schema {
	// Jika SQL kosong, kembalikan skema kosong
	if sql == "" {
		return &Schema{
			Tables: make([]*Table, 0),
		}
	}

	// Parse SQL untuk mendapatkan skema
	schema := &Schema{
		Tables: make([]*Table, 0),
	}

	// Split SQL berdasarkan CREATE TABLE
	tables := strings.Split(sql, "CREATE TABLE")
	for _, tableSQL := range tables {
		if strings.TrimSpace(tableSQL) == "" {
			continue
		}

		// Parse nama tabel
		tableName := ""
		if start := strings.Index(tableSQL, "`"); start != -1 {
			if end := strings.Index(tableSQL[start+1:], "`"); end != -1 {
				tableName = tableSQL[start+1 : start+1+end]
			}
		}
		if tableName == "" {
			continue
		}

		// Buat tabel baru
		table := &Table{
			Name:        tableName,
			Columns:     make([]*Column, 0),
			Indexes:     make([]*Index, 0),
			ForeignKeys: make([]*ForeignKey, 0),
		}

		// Parse kolom
		columns := strings.Split(tableSQL, "\n")
		for _, line := range columns {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "`") {
				continue
			}

			// Parse nama kolom
			columnName := ""
			if start := strings.Index(line, "`"); start != -1 {
				if end := strings.Index(line[start+1:], "`"); end != -1 {
					columnName = line[start+1 : start+1+end]
				}
			}
			if columnName == "" {
				continue
			}

			// Parse tipe data
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}

			// Buat kolom baru
			column := &Column{
				Name: columnName,
				Type: strings.ToUpper(parts[2]),
			}

			// Parse opsi kolom
			if strings.Contains(line, "NOT NULL") {
				column.Nullable = false
			} else {
				column.Nullable = true
			}

			if strings.Contains(line, "AUTO_INCREMENT") {
				column.AutoIncrement = true
			}

			if strings.Contains(line, "DEFAULT") {
				if idx := strings.Index(line, "DEFAULT"); idx != -1 {
					rest := line[idx+7:]
					if end := strings.Index(rest, " "); end != -1 {
						column.Default = strings.TrimSpace(rest[:end])
					} else {
						column.Default = strings.TrimSpace(rest)
					}
				}
			}

			// Parse length untuk VARCHAR/CHAR
			if strings.Contains(column.Type, "VARCHAR") || strings.Contains(column.Type, "CHAR") {
				if start := strings.Index(line, "("); start != -1 {
					if end := strings.Index(line[start:], ")"); end != -1 {
						fmt.Sscanf(line[start+1:start+end], "%d", &column.Length)
					}
				}
			}

			table.Columns = append(table.Columns, column)
		}

		// Parse primary key
		if strings.Contains(tableSQL, "PRIMARY KEY") {
			if start := strings.Index(tableSQL, "PRIMARY KEY"); start != -1 {
				if keyStart := strings.Index(tableSQL[start:], "("); keyStart != -1 {
					if keyEnd := strings.Index(tableSQL[start+keyStart:], ")"); keyEnd != -1 {
						keyStr := tableSQL[start+keyStart+1 : start+keyStart+keyEnd]
						keyStr = strings.ReplaceAll(keyStr, "`", "")
						table.PrimaryKey = &PrimaryKey{
							Name:    fmt.Sprintf("pk_%s", table.Name),
							Columns: strings.Split(keyStr, ", "),
						}
					}
				}
			}
		}

		// Parse indexes
		if strings.Contains(tableSQL, "UNIQUE KEY") || strings.Contains(tableSQL, "KEY") {
			lines := strings.Split(tableSQL, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "UNIQUE KEY") || strings.HasPrefix(line, "KEY") {
					index := &Index{
						Type:    "BTREE",
						Unique:  strings.HasPrefix(line, "UNIQUE"),
						Columns: make([]string, 0),
					}

					// Parse nama index
					if start := strings.Index(line, "`"); start != -1 {
						if end := strings.Index(line[start+1:], "`"); end != -1 {
							index.Name = line[start+1 : start+1+end]
						}
					}

					// Parse kolom index
					if start := strings.Index(line, "("); start != -1 {
						if end := strings.Index(line[start:], ")"); end != -1 {
							colStr := line[start+1 : start+end]
							colStr = strings.ReplaceAll(colStr, "`", "")
							index.Columns = strings.Split(colStr, ", ")
						}
					}

					table.Indexes = append(table.Indexes, index)
				}
			}
		}

		// Parse foreign keys
		if strings.Contains(tableSQL, "FOREIGN KEY") {
			lines := strings.Split(tableSQL, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "FOREIGN KEY") {
					fk := &ForeignKey{
						Columns:          make([]string, 0),
						ReferenceColumns: make([]string, 0),
					}

					// Parse nama foreign key
					if start := strings.Index(line, "`"); start != -1 {
						if end := strings.Index(line[start+1:], "`"); end != -1 {
							fk.Name = line[start+1 : start+1+end]
						}
					}

					// Parse kolom foreign key
					if start := strings.Index(line, "FOREIGN KEY"); start != -1 {
						if keyStart := strings.Index(line[start:], "("); keyStart != -1 {
							if keyEnd := strings.Index(line[start+keyStart:], ")"); keyEnd != -1 {
								colStr := line[start+keyStart+1 : start+keyStart+keyEnd]
								colStr = strings.ReplaceAll(colStr, "`", "")
								fk.Columns = strings.Split(colStr, ", ")
							}
						}
					}

					// Parse tabel dan kolom referensi
					if start := strings.Index(line, "REFERENCES"); start != -1 {
						rest := line[start+10:]
						if tableStart := strings.Index(rest, "`"); tableStart != -1 {
							if tableEnd := strings.Index(rest[tableStart+1:], "`"); tableEnd != -1 {
								fk.ReferenceTable = rest[tableStart+1 : tableStart+1+tableEnd]
							}
						}
						if colStart := strings.Index(rest, "("); colStart != -1 {
							if colEnd := strings.Index(rest[colStart:], ")"); colEnd != -1 {
								colStr := rest[colStart+1 : colStart+colEnd]
								colStr = strings.ReplaceAll(colStr, "`", "")
								fk.ReferenceColumns = strings.Split(colStr, ", ")
							}
						}
					}

					// Parse ON DELETE dan ON UPDATE
					if strings.Contains(line, "ON DELETE") {
						if start := strings.Index(line, "ON DELETE"); start != -1 {
							rest := line[start+9:]
							if end := strings.Index(rest, " "); end != -1 {
								fk.OnDelete = strings.TrimSpace(rest[:end])
							} else {
								fk.OnDelete = strings.TrimSpace(rest)
							}
						}
					}
					if strings.Contains(line, "ON UPDATE") {
						if start := strings.Index(line, "ON UPDATE"); start != -1 {
							rest := line[start+9:]
							if end := strings.Index(rest, " "); end != -1 {
								fk.OnUpdate = strings.TrimSpace(rest[:end])
							} else {
								fk.OnUpdate = strings.TrimSpace(rest)
							}
						}
					}

					table.ForeignKeys = append(table.ForeignKeys, fk)
				}
			}
		}

		schema.Tables = append(schema.Tables, table)
	}

	return schema
}

// String menghasilkan representasi string dari tipe enum
func (e *EnumType) String() string {
	// Escape nilai enum
	escapedValues := make([]string, len(e.Values))
	for i, v := range e.Values {
		escapedValues[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	}

	return fmt.Sprintf("ENUM(%s)", strings.Join(escapedValues, ","))
}
