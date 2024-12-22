package datara

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"unicode"
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
}

// Column represents a table column
type Column struct {
	Name        string
	Type        string
	Nullable    bool
	Default     interface{}
	Tags        map[string]string
	Length      int          // untuk VARCHAR, CHAR, dll
	Precision   int          // untuk DECIMAL, NUMERIC
	Scale       int          // untuk DECIMAL, NUMERIC
	Unsigned    bool         // untuk tipe numerik
	Charset     string       // untuk tipe string
	Collation   string       // untuk tipe string
	JSONOptions *JSONOptions // opsi untuk kolom JSON
	IsJSON      bool         // menandai kolom sebagai JSON
	JSONSchema  string       // skema JSON untuk validasi
}

// Index represents a table index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name            string
	Column          string
	ReferenceTable  string
	ReferenceColumn string
	OnDelete        string
	OnUpdate        string
}

// RelationType menentukan jenis relasi
type RelationType string

const (
	OneToOne   RelationType = "one_to_one"
	OneToMany  RelationType = "one_to_many"
	ManyToOne  RelationType = "many_to_one"
	ManyToMany RelationType = "many_to_many"
)

// SQLType menentukan tipe data SQL
type SQLType string

const (
	// Tipe String
	Char       SQLType = "CHAR"
	Varchar    SQLType = "VARCHAR"
	Text       SQLType = "TEXT"
	Tinytext   SQLType = "TINYTEXT"
	Mediumtext SQLType = "MEDIUMTEXT"
	Longtext   SQLType = "LONGTEXT"

	// Tipe Numerik
	Tinyint   SQLType = "TINYINT"
	Smallint  SQLType = "SMALLINT"
	Mediumint SQLType = "MEDIUMINT"
	Int       SQLType = "INT"
	Bigint    SQLType = "BIGINT"
	Decimal   SQLType = "DECIMAL"
	Float     SQLType = "FLOAT"
	Double    SQLType = "DOUBLE"

	// Tipe Date/Time
	Date      SQLType = "DATE"
	Time      SQLType = "TIME"
	DateTime  SQLType = "DATETIME"
	Timestamp SQLType = "TIMESTAMP"
	Year      SQLType = "YEAR"

	// Tipe Binary
	Binary     SQLType = "BINARY"
	Varbinary  SQLType = "VARBINARY"
	Blob       SQLType = "BLOB"
	Tinyblob   SQLType = "TINYBLOB"
	Mediumblob SQLType = "MEDIUMBLOB"
	Longblob   SQLType = "LONGBLOB"

	// Tipe Khusus
	Enum SQLType = "ENUM"
	Set  SQLType = "SET"
	Json SQLType = "JSON"
	Uuid SQLType = "UUID"
)

// ParserConfig adalah konfigurasi untuk parser
type ParserConfig struct {
	Naming     NamingConfig
	Types      map[string]string
	Charset    string
	Collation  string
	Engine     string
	SoftDelete bool
}

// NamingConfig adalah konfigurasi untuk penamaan
type NamingConfig struct {
	TablePlural     bool
	TableSnakeCase  bool
	ColumnSnakeCase bool
}

// Parser adalah interface utama untuk mengkonversi struct menjadi skema migrasi
type Parser interface {
	Parse(interface{}) (*Schema, error)
	ParseFile(string) (*Schema, error)
	ParseSchema(string) (*Schema, error)
}

// DefaultParser adalah implementasi default dari Parser
type DefaultParser struct {
	config ParserConfig
}

// NewParser membuat instance baru dari DefaultParser dengan konfigurasi default
func NewParser() Parser {
	return &DefaultParser{
		config: ParserConfig{
			Charset:   "utf8mb4",
			Collation: "utf8mb4_unicode_ci",
			Engine:    "InnoDB",
		},
	}
}

// NewParserWithConfig membuat instance baru dari DefaultParser dengan konfigurasi kustom
func NewParserWithConfig(config ParserConfig) Parser {
	return &DefaultParser{
		config: config,
	}
}

// ParseSchema mengkonversi output dari program schema menjadi Schema
func (p *DefaultParser) ParseSchema(schemaStr string) (*Schema, error) {
	// Parse schema string
	// Ini hanya contoh sederhana, Anda perlu mengimplementasikan parsing yang sesuai
	// dengan format output dari program register.go
	var schema Schema
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		return nil, fmt.Errorf("error parsing schema: %v", err)
	}

	// Apply naming conventions
	if p.config.Naming.TablePlural {
		for _, table := range schema.Tables {
			table.Name = pluralize(table.Name)
		}
	}
	if p.config.Naming.TableSnakeCase {
		for _, table := range schema.Tables {
			table.Name = toSnakeCase(table.Name)
		}
	}
	if p.config.Naming.ColumnSnakeCase {
		for _, table := range schema.Tables {
			for _, column := range table.Columns {
				column.Name = toSnakeCase(column.Name)
			}
		}
	}

	// Apply custom type mappings
	if len(p.config.Types) > 0 {
		for _, table := range schema.Tables {
			for _, column := range table.Columns {
				if customType, ok := p.config.Types[column.Type]; ok {
					column.Type = customType
				}
			}
		}
	}

	return &schema, nil
}

// Helper functions for naming conventions
func pluralize(s string) string {
	if s == "" {
		return "s"
	}

	// Aturan khusus
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}

	return s + "s"
}

func toSnakeCase(s string) string {
	var result strings.Builder
	var prev rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			// Jika huruf sebelumnya adalah huruf kecil atau
			// jika huruf setelahnya adalah huruf kecil (dalam kasus seperti "API")
			// tambahkan underscore
			if unicode.IsLower(prev) || (i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
				result.WriteRune('_')
			}
		}
		prev = r
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// Parse mengkonversi struct Go menjadi skema database
func (p *DefaultParser) Parse(model interface{}) (*Schema, error) {
	if model == nil {
		return nil, fmt.Errorf("model tidak boleh nil")
	}

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model harus berupa struct, ditemukan %s", t.Kind())
	}

	table := &Table{
		Name:        t.Name(),
		Columns:     make([]*Column, 0),
		ForeignKeys: make([]*ForeignKey, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		column := p.parseField(field)
		if column != nil {
			table.Columns = append(table.Columns, column)

			// Parse foreign key jika ada
			if fk := p.parseForeignKey(field, column); fk != nil {
				table.ForeignKeys = append(table.ForeignKeys, fk)
			}
		}
	}

	schema := &Schema{
		Tables: []*Table{table},
	}

	return schema, nil
}

// ParseFile membaca file Go dan mengkonversi struct di dalamnya menjadi skema database
func (p *DefaultParser) ParseFile(filename string) (*Schema, error) {
	// Baca file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing Go file: %v", err)
	}

	// Cari struct dalam file
	schema := &Schema{
		Tables: make([]*Table, 0),
	}

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Parse struct
		table := &Table{
			Name:        typeSpec.Name.Name,
			Columns:     make([]*Column, 0),
			ForeignKeys: make([]*ForeignKey, 0),
		}

		// Parse fields
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue
			}

			// Get field type
			fieldType := p.parseASTExpr(field.Type)
			if fieldType == "" {
				continue
			}

			// Parse field tags
			var tags map[string]string
			if field.Tag != nil {
				tag := strings.Trim(field.Tag.Value, "`")
				tags = p.parseTags(reflect.StructTag(tag))
			}

			// Create column
			column := &Column{
				Name:     field.Names[0].Name,
				Type:     fieldType,
				Tags:     tags,
				Nullable: p.isNullableFromTags(tags),
			}

			// Parse additional type information
			if typeTag, ok := tags["type"]; ok {
				parts := strings.Split(typeTag, ",")
				column.Type = parts[0]

				for _, part := range parts[1:] {
					if strings.HasPrefix(part, "length=") {
						fmt.Sscanf(strings.TrimPrefix(part, "length="), "%d", &column.Length)
					} else if strings.HasPrefix(part, "precision=") {
						fmt.Sscanf(strings.TrimPrefix(part, "precision="), "%d", &column.Precision)
					} else if strings.HasPrefix(part, "scale=") {
						fmt.Sscanf(strings.TrimPrefix(part, "scale="), "%d", &column.Scale)
					} else if part == "unsigned" {
						column.Unsigned = true
					}
				}
			}

			// Handle JSON options
			if strings.HasPrefix(column.Type, "JSON") {
				column.IsJSON = true
			}

			// Parse charset and collation
			if charset, ok := tags["charset"]; ok {
				column.Charset = charset
			}
			if collation, ok := tags["collation"]; ok {
				column.Collation = collation
			}

			// Check untuk default value dari tag
			if defaultVal, ok := tags["default"]; ok {
				column.Default = defaultVal
			}

			table.Columns = append(table.Columns, column)

			// Parse foreign key jika ada
			if fk := p.parseForeignKeyFromTags(field.Names[0].Name, tags); fk != nil {
				table.ForeignKeys = append(table.ForeignKeys, fk)
			}
		}

		schema.Tables = append(schema.Tables, table)
		return true
	})

	return schema, nil
}

// parseASTExpr mengkonversi AST expression ke string tipe SQL
func (p *DefaultParser) parseASTExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return p.mapGoTypeToSQL(t.Name)
	case *ast.ArrayType:
		return "JSON"
	case *ast.MapType:
		return "JSON"
	case *ast.StarExpr:
		return p.parseASTExpr(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			if ident.Name == "time" && t.Sel.Name == "Time" {
				return "DATETIME"
			}
		}
		return "TEXT"
	default:
		return "TEXT"
	}
}

// mapGoTypeToSQL mengkonversi tipe Go ke tipe SQL
func (p *DefaultParser) mapGoTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR(255)"
	case "int", "int32":
		return "INT"
	case "int64":
		return "BIGINT"
	case "float32":
		return "FLOAT"
	case "float64":
		return "DOUBLE"
	case "bool":
		return "TINYINT(1)"
	default:
		return "TEXT"
	}
}

// isNullableFromTags menentukan apakah field bisa null berdasarkan tags
func (p *DefaultParser) isNullableFromTags(tags map[string]string) bool {
	if _, ok := tags["nullable"]; ok {
		return true
	}
	if _, ok := tags["not null"]; ok {
		return false
	}
	return true
}

// parseForeignKeyFromTags mengekstrak foreign key dari tags
func (p *DefaultParser) parseForeignKeyFromTags(fieldName string, tags map[string]string) *ForeignKey {
	relTag, ok := tags["rel"]
	if !ok {
		return nil
	}

	parts := strings.Split(relTag, ",")
	if len(parts) < 2 {
		return nil
	}

	fk := &ForeignKey{
		Name:            fmt.Sprintf("fk_%s_%s", fieldName, parts[0]),
		Column:          fieldName,
		ReferenceTable:  parts[0],
		ReferenceColumn: parts[1],
		OnDelete:        "CASCADE", // default
		OnUpdate:        "CASCADE", // default
	}

	// Parse opsi tambahan
	for _, part := range parts[2:] {
		if strings.HasPrefix(part, "ondelete=") {
			fk.OnDelete = strings.TrimPrefix(part, "ondelete=")
		} else if strings.HasPrefix(part, "onupdate=") {
			fk.OnUpdate = strings.TrimPrefix(part, "onupdate=")
		}
	}

	return fk
}

// EnumType adalah interface untuk tipe enum custom
type EnumType interface {
	Values() []string
	String() string
}

// EnumValuer adalah interface untuk mendapatkan nilai enum
type EnumValuer interface {
	EnumValue() string
}

// JSONOptions menentukan opsi untuk kolom JSON
type JSONOptions struct {
	Validate bool   // Validasi JSON saat insert/update
	Schema   string // JSON Schema untuk validasi
}

// getSQLType mengembalikan tipe SQL yang sesuai untuk tipe Go
func (p *DefaultParser) getSQLType(t reflect.Type) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check custom type definitions first
	switch t.String() {
	case "time.Time":
		return string(DateTime)
	case "uuid.UUID":
		return string(Uuid)
	}

	switch t.Kind() {
	case reflect.String:
		return string(Varchar) + "(255)"
	case reflect.Int8:
		return string(Tinyint)
	case reflect.Int16:
		return string(Smallint)
	case reflect.Int32, reflect.Int:
		return string(Int)
	case reflect.Int64:
		return string(Bigint)
	case reflect.Uint8:
		return string(Tinyint) + " UNSIGNED"
	case reflect.Uint16:
		return string(Smallint) + " UNSIGNED"
	case reflect.Uint32:
		return string(Int) + " UNSIGNED"
	case reflect.Uint, reflect.Uint64:
		return string(Bigint) + " UNSIGNED"
	case reflect.Float32:
		return string(Float)
	case reflect.Float64:
		return string(Double)
	case reflect.Bool:
		return string(Tinyint) + "(1)"
	case reflect.Struct:
		return string(Json)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return string(Blob)
		}
		return string(Json)
	case reflect.Map:
		return string(Json)
	default:
		return string(Text)
	}
}

// generateJSONSchema menghasilkan skema JSON untuk tipe Go
func (p *DefaultParser) generateJSONSchema(t reflect.Type) string {
	schema := make(map[string]interface{})

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.String:
		schema["type"] = "string"
	case reflect.Struct:
		if t.String() == "time.Time" {
			schema["type"] = "string"
			schema["format"] = "date-time"
		} else {
			schema["type"] = "object"
			properties := make(map[string]interface{})
			required := make([]string, 0)

			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)

				// Skip unexported fields
				if field.PkgPath != "" {
					continue
				}

				// Get JSON field name
				jsonTag := field.Tag.Get("json")
				fieldName := field.Name
				omitempty := false
				if jsonTag != "" {
					parts := strings.Split(jsonTag, ",")
					if parts[0] != "-" {
						fieldName = parts[0]
					} else {
						continue // Skip fields with json:"-"
					}
					for _, opt := range parts[1:] {
						if opt == "omitempty" {
							omitempty = true
							break
						}
					}
				}

				// Get field schema
				fieldSchema := p.getJSONFieldSchema(field.Type)
				properties[fieldName] = fieldSchema

				// Check if field is required
				if !omitempty && field.Type.Kind() != reflect.Ptr {
					required = append(required, fieldName)
				}
			}

			schema["properties"] = properties
			if len(required) > 0 {
				schema["required"] = required
			}
		}
	case reflect.Slice:
		schema["type"] = "array"
		schema["items"] = p.getJSONFieldSchema(t.Elem())
	case reflect.Map:
		schema["type"] = "object"
		if t.Key().Kind() == reflect.String {
			schema["additionalProperties"] = p.getJSONFieldSchema(t.Elem())
		}
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

// getJSONFieldSchema mengembalikan skema JSON untuk tipe field
func (p *DefaultParser) getJSONFieldSchema(t reflect.Type) map[string]interface{} {
	schema := make(map[string]interface{})

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.String:
		schema["type"] = "string"
	case reflect.Struct:
		if t.String() == "time.Time" {
			schema["type"] = "string"
			schema["format"] = "date-time"
		} else {
			schema["type"] = "object"
			properties := make(map[string]interface{})
			required := make([]string, 0)

			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)

				// Skip unexported fields
				if field.PkgPath != "" {
					continue
				}

				// Get JSON field name
				jsonTag := field.Tag.Get("json")
				fieldName := field.Name
				omitempty := false
				if jsonTag != "" {
					parts := strings.Split(jsonTag, ",")
					if parts[0] != "-" {
						fieldName = parts[0]
					} else {
						continue // Skip fields with json:"-"
					}
					for _, opt := range parts[1:] {
						if opt == "omitempty" {
							omitempty = true
							break
						}
					}
				}

				// Get field schema
				fieldSchema := p.getJSONFieldSchema(field.Type)
				properties[fieldName] = fieldSchema

				// Check if field is required
				if !omitempty && field.Type.Kind() != reflect.Ptr {
					required = append(required, fieldName)
				}
			}

			schema["properties"] = properties
			if len(required) > 0 {
				schema["required"] = required
			}
		}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			schema["type"] = "string"
			schema["contentEncoding"] = "base64"
		} else {
			schema["type"] = "array"
			schema["items"] = p.getJSONFieldSchema(t.Elem())
		}
	case reflect.Map:
		schema["type"] = "object"
		if t.Key().Kind() == reflect.String {
			schema["additionalProperties"] = p.getJSONFieldSchema(t.Elem())
		}
	}

	return schema
}

// parseField mengkonversi struct field menjadi definisi kolom
func (p *DefaultParser) parseField(field reflect.StructField) *Column {
	// Skip jika field unexported
	if field.PkgPath != "" {
		return nil
	}

	tags := p.parseTags(field.Tag)

	// Parse tipe data
	sqlType := p.getSQLType(field.Type)

	// Jika ada type di tag, gunakan itu
	if typeTag, ok := tags["type"]; ok {
		sqlType = typeTag
		// Tambahkan length jika ada
		if length, ok := tags["length"]; ok {
			sqlType = fmt.Sprintf("%s(%s)", sqlType, length)
		}
	}

	column := &Column{
		Name:     field.Name,
		Type:     sqlType,
		Tags:     tags,
		Nullable: p.isNullable(field),
	}

	// Handle JSON options
	if strings.HasPrefix(strings.ToUpper(column.Type), "JSON") {
		column.IsJSON = true
		if schema := p.generateJSONSchema(field.Type); schema != "" {
			column.JSONSchema = schema
			column.JSONOptions = &JSONOptions{
				Validate: true,
				Schema:   schema,
			}
		}
	}

	// Parse charset and collation
	if charset, ok := tags["charset"]; ok {
		column.Charset = charset
	}
	if collation, ok := tags["collation"]; ok {
		column.Collation = collation
	}

	// Check untuk default value dari tag
	if defaultVal, ok := tags["default"]; ok {
		column.Default = defaultVal
	}

	return column
}

// parseTags mengekstrak informasi dari struct tag
func (p *DefaultParser) parseTags(tag reflect.StructTag) map[string]string {
	tags := make(map[string]string)

	// Parse `db` tag
	if dbTag := tag.Get("db"); dbTag != "" {
		parts := strings.Split(dbTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "=") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.TrimSpace(kv[1])
					if key != "" {
						tags[key] = value
					}
				}
			} else if part != "" {
				tags[strings.TrimSpace(part)] = ""
			}
		}
	}

	// Parse `rel` tag untuk relasi
	if relTag := tag.Get("rel"); relTag != "" {
		tags["rel"] = strings.TrimSpace(relTag)
	}

	return tags
}

// parseForeignKey mengekstrak informasi foreign key dari field
func (p *DefaultParser) parseForeignKey(field reflect.StructField, column *Column) *ForeignKey {
	relTag := field.Tag.Get("rel")
	if relTag == "" {
		return nil
	}

	parts := strings.Split(relTag, ",")
	if len(parts) < 2 {
		return nil
	}

	fk := &ForeignKey{
		Name:            fmt.Sprintf("fk_%s_%s", field.Name, parts[0]),
		Column:          column.Name,
		ReferenceTable:  parts[0],
		ReferenceColumn: parts[1],
		OnDelete:        "CASCADE", // default
		OnUpdate:        "CASCADE", // default
	}

	// Parse opsi tambahan
	for _, part := range parts[2:] {
		if strings.HasPrefix(part, "ondelete=") {
			fk.OnDelete = strings.TrimPrefix(part, "ondelete=")
		} else if strings.HasPrefix(part, "onupdate=") {
			fk.OnUpdate = strings.TrimPrefix(part, "onupdate=")
		}
	}

	return fk
}

// isNullable menentukan apakah field bisa null
func (p *DefaultParser) isNullable(field reflect.StructField) bool {
	// Pointer types are nullable
	if field.Type.Kind() == reflect.Ptr {
		return true
	}

	// Check tags
	if tag := field.Tag.Get("db"); tag != "" {
		// Explicitly marked as not null
		if strings.Contains(tag, "not null") {
			return false
		}
		// Explicitly marked as nullable
		if strings.Contains(tag, "nullable") {
			return true
		}
	}

	// Default to true unless it's a primary key
	if tag := field.Tag.Get("db"); strings.Contains(tag, "primary_key") {
		return false
	}

	return true
}

// parseEnumType mengekstrak nilai enum dari tipe Go
func (p *DefaultParser) parseEnumType(t reflect.Type, tags map[string]string) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Cek apakah ada tag enum values
	if values, ok := tags["enum"]; ok {
		enumValues := strings.Split(values, "|")
		if len(enumValues) > 0 {
			return fmt.Sprintf("ENUM('%s')", strings.Join(enumValues, "','"))
		}
	}

	// Cek apakah tipe mengimplementasi EnumType
	if t.Implements(reflect.TypeOf((*EnumType)(nil)).Elem()) {
		// Buat instance dari tipe enum
		enumValue := reflect.New(t).Elem().Interface().(EnumType)
		values := enumValue.Values()

		if len(values) > 0 {
			return fmt.Sprintf("ENUM('%s')", strings.Join(values, "','"))
		}
	}

	// Cek apakah tipe mengimplementasi EnumValuer
	if t.Implements(reflect.TypeOf((*EnumValuer)(nil)).Elem()) {
		// Kita akan menggunakan tipe string untuk enum ini
		return "VARCHAR(50)"
	}

	return ""
}
