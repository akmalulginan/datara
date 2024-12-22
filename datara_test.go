package datara

import (
	"encoding/json"
	"go/ast"
	"os"
	"reflect"
	"testing"
	"time"
)

// Test struct untuk pengujian
type TestUser struct {
	ID        uint         `db:"primary_key,autoincrement"`
	UUID      string       `db:"type=VARCHAR,length=36"`
	Name      string       `db:"type=VARCHAR,length=100,not null"`
	Email     string       `db:"type=VARCHAR,length=255,unique"`
	Age       int          `db:"nullable"`
	Status    string       `db:"type=ENUM,enum=active|inactive|suspended"`
	Data      any          `db:"type=JSON"`
	CreatedAt time.Time    `db:"not null,default=CURRENT_TIMESTAMP"`
	Profile   *TestProfile `rel:"profiles,id,ondelete=CASCADE"`
}

type TestProfile struct {
	ID     uint   `db:"primary_key,autoincrement"`
	UserID uint   `db:"type=INT"`
	Bio    string `db:"type=TEXT,nullable"`
}

func TestParse(t *testing.T) {
	parser := NewParser()

	type Address struct {
		Street  string `db:"type=varchar(255)"`
		City    string `db:"type=varchar(100)"`
		Country string `db:"type=varchar(100)"`
	}

	type User struct {
		ID        int       `db:"type=int,primary_key"`
		Name      string    `db:"type=varchar(255),not null"`
		Email     string    `db:"type=varchar(255),unique"`
		Age       *int      `db:"type=int,nullable"`
		Address   Address   `db:"type=json"`
		CreatedAt time.Time `db:"type=timestamp"`
	}

	tests := []struct {
		name     string
		model    interface{}
		wantErr  bool
		validate func(*testing.T, *Schema)
	}{
		{
			name:    "Valid Struct",
			model:   User{},
			wantErr: false,
			validate: func(t *testing.T, s *Schema) {
				if len(s.Tables) != 1 {
					t.Errorf("Expected 1 table, got %d", len(s.Tables))
					return
				}

				table := s.Tables[0]
				if table.Name != "User" {
					t.Errorf("Expected table name 'User', got %s", table.Name)
				}

				expectedColumns := 6
				if len(table.Columns) != expectedColumns {
					t.Errorf("Expected %d columns, got %d", expectedColumns, len(table.Columns))
				}

				// Validate specific columns
				for _, col := range table.Columns {
					switch col.Name {
					case "ID":
						if col.Type != "int" || col.Nullable {
							t.Errorf("Invalid ID column configuration")
						}
					case "Email":
						if col.Type != "varchar(255)" {
							t.Errorf("Invalid Email column type")
						}
					case "Age":
						if !col.Nullable {
							t.Errorf("Age should be nullable")
						}
					case "Address":
						if !col.IsJSON {
							t.Errorf("Address should be JSON type")
						}
					}
				}
			},
		},
		{
			name:    "Nil Model",
			model:   nil,
			wantErr: true,
		},
		{
			name:    "Non-Struct Type",
			model:   "not a struct",
			wantErr: true,
		},
		{
			name:    "Pointer to Struct",
			model:   &User{},
			wantErr: false,
			validate: func(t *testing.T, s *Schema) {
				if len(s.Tables) != 1 {
					t.Errorf("Expected 1 table, got %d", len(s.Tables))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := parser.Parse(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, schema)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary file with test content
	content := `package test

type User struct {
	ID        int       ` + "`db:\"type=int,primary_key\"`" + `
	Name      string    ` + "`db:\"type=varchar(255),not null\"`" + `
	Email     string    ` + "`db:\"type=varchar(255),unique\"`" + `
	CreatedAt time.Time ` + "`db:\"type=timestamp\"`" + `
}
`
	tmpfile, err := os.CreateTemp("", "test_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	schema, err := parser.ParseFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if len(schema.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(schema.Tables))
		return
	}

	table := schema.Tables[0]
	if table.Name != "User" {
		t.Errorf("Expected table name 'User', got %s", table.Name)
	}

	expectedColumns := 4
	if len(table.Columns) != expectedColumns {
		t.Errorf("Expected %d columns, got %d", expectedColumns, len(table.Columns))
	}

	// Test with invalid file
	_, err = parser.ParseFile("nonexistent.go")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseSchema(t *testing.T) {
	parser := NewParser()

	validSchema := `{
		"Tables": [
			{
				"Name": "users",
				"Columns": [
					{
						"Name": "id",
						"Type": "int",
						"Nullable": false
					},
					{
						"Name": "name",
						"Type": "varchar(255)",
						"Nullable": false
					}
				]
			}
		]
	}`

	invalidSchema := `{invalid json}`

	tests := []struct {
		name     string
		schema   string
		wantErr  bool
		validate func(*testing.T, *Schema)
	}{
		{
			name:    "Valid Schema",
			schema:  validSchema,
			wantErr: false,
			validate: func(t *testing.T, s *Schema) {
				if len(s.Tables) != 1 {
					t.Errorf("Expected 1 table, got %d", len(s.Tables))
					return
				}
				if s.Tables[0].Name != "users" {
					t.Errorf("Expected table name 'users', got %s", s.Tables[0].Name)
				}
			},
		},
		{
			name:    "Invalid Schema",
			schema:  invalidSchema,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := parser.ParseSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, schema)
			}
		})
	}
}

func TestParseField(t *testing.T) {
	parser := NewParser()

	type TestStruct struct {
		ID        uint      `db:"primary_key,autoincrement"`
		Name      string    `db:"type=VARCHAR,length=100,not null"`
		Email     string    `db:"type=VARCHAR,length=255,unique"`
		Age       int       `db:"nullable"`
		Status    string    `db:"type=ENUM,enum=active|inactive|suspended"`
		Data      any       `db:"type=JSON"`
		CreatedAt time.Time `db:"not null,default=CURRENT_TIMESTAMP"`
		UpdatedAt *time.Time
		DeletedAt *time.Time `db:"nullable"`
	}

	t.Run("All Fields", func(t *testing.T) {
		typ := reflect.TypeOf(TestStruct{})
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			column := parser.(*DefaultParser).parseField(field)
			if column == nil && !field.Anonymous {
				t.Errorf("parseField() returned nil for field %s", field.Name)
			}
		}
	})
}

func TestNewParserWithConfig(t *testing.T) {
	config := ParserConfig{
		Naming: NamingConfig{
			TablePlural:     true,
			TableSnakeCase:  true,
			ColumnSnakeCase: true,
		},
		Types: map[string]string{
			"string": "TEXT",
			"int":    "BIGINT",
		},
		Charset:    "utf8",
		Collation:  "utf8_general_ci",
		Engine:     "MyISAM",
		SoftDelete: true,
	}

	parser := NewParserWithConfig(config)
	dp, ok := parser.(*DefaultParser)
	if !ok {
		t.Fatal("Expected DefaultParser type")
	}

	if !reflect.DeepEqual(dp.config, config) {
		t.Errorf("NewParserWithConfig() config = %v, want %v", dp.config, config)
	}
}

func TestGenerateJSONSchema(t *testing.T) {
	parser := NewParser()

	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type User struct {
		ID        int                    `json:"id"`
		Name      string                 `json:"name"`
		Age       *int                   `json:"age,omitempty"`
		Address   Address                `json:"address"`
		Tags      []string               `json:"tags"`
		Metadata  map[string]interface{} `json:"metadata"`
		CreatedAt time.Time              `json:"created_at"`
	}

	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{
			name:     "Simple Types",
			typ:      reflect.TypeOf(""),
			expected: `{"type":"string"}`,
		},
		{
			name:     "Integer Types",
			typ:      reflect.TypeOf(0),
			expected: `{"type":"integer"}`,
		},
		{
			name:     "Float Types",
			typ:      reflect.TypeOf(0.0),
			expected: `{"type":"number"}`,
		},
		{
			name:     "Boolean Types",
			typ:      reflect.TypeOf(true),
			expected: `{"type":"boolean"}`,
		},
		{
			name: "Complex Struct",
			typ:  reflect.TypeOf(User{}),
			expected: `{
				"type": "object",
				"properties": {
					"id": {"type": "integer"},
					"name": {"type": "string"},
					"age": {"type": "integer"},
					"address": {
						"type": "object",
						"properties": {
							"street": {"type": "string"},
							"city": {"type": "string"},
							"country": {"type": "string"}
						},
						"required": ["street", "city", "country"]
					},
					"tags": {
						"type": "array",
						"items": {"type": "string"}
					},
					"metadata": {
						"type": "object",
						"additionalProperties": {}
					},
					"created_at": {
						"type": "string",
						"format": "date-time"
					}
				},
				"required": ["id", "name", "address", "tags", "metadata", "created_at"]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).generateJSONSchema(tt.typ)

			// Normalize JSON for comparison
			var gotMap, expectedMap map[string]interface{}
			if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
				t.Fatalf("Failed to unmarshal generated schema: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedMap); err != nil {
				t.Fatalf("Failed to unmarshal expected schema: %v", err)
			}

			if !reflect.DeepEqual(gotMap, expectedMap) {
				gotJSON, _ := json.MarshalIndent(gotMap, "", "  ")
				expectedJSON, _ := json.MarshalIndent(expectedMap, "", "  ")
				t.Errorf("generateJSONSchema() =\n%s\nwant\n%s", gotJSON, expectedJSON)
			}
		})
	}
}

func TestParseTags(t *testing.T) {
	parser := NewParser()

	type TestStruct struct {
		Field1 string `db:"type=varchar(255),not null,column=field_1"`
		Field2 string `db:"nullable" rel:"users,id,ondelete=SET NULL"`
		Field3 string `db:"type=enum,enum=a|b|c"`
	}

	field1, _ := reflect.TypeOf(TestStruct{}).FieldByName("Field1")
	field2, _ := reflect.TypeOf(TestStruct{}).FieldByName("Field2")
	field3, _ := reflect.TypeOf(TestStruct{}).FieldByName("Field3")

	tests := []struct {
		name     string
		tag      reflect.StructTag
		expected map[string]string
	}{
		{
			name: "Multiple DB Tags",
			tag:  field1.Tag,
			expected: map[string]string{
				"type":     "varchar(255)",
				"not null": "",
				"column":   "field_1",
			},
		},
		{
			name: "DB and Rel Tags",
			tag:  field2.Tag,
			expected: map[string]string{
				"nullable": "",
				"rel":      "users,id,ondelete=SET NULL",
			},
		},
		{
			name: "Enum Tags",
			tag:  field3.Tag,
			expected: map[string]string{
				"type": "enum",
				"enum": "a|b|c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).parseTags(tt.tag)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseTags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNaming(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSnake  string
		wantPlural string
	}{
		{
			name:       "Simple Word",
			input:      "User",
			wantSnake:  "user",
			wantPlural: "Users",
		},
		{
			name:       "Camel Case",
			input:      "UserProfile",
			wantSnake:  "user_profile",
			wantPlural: "UserProfiles",
		},
		{
			name:       "Already Snake Case",
			input:      "user_profile",
			wantSnake:  "user_profile",
			wantPlural: "user_profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSnake := toSnakeCase(tt.input)
			if gotSnake != tt.wantSnake {
				t.Errorf("toSnakeCase() = %v, want %v", gotSnake, tt.wantSnake)
			}

			gotPlural := pluralize(tt.input)
			if gotPlural != tt.wantPlural {
				t.Errorf("pluralize() = %v, want %v", gotPlural, tt.wantPlural)
			}
		})
	}
}

func TestParseEnumType(t *testing.T) {
	parser := NewParser()

	type Status int
	const (
		Active Status = iota
		Inactive
		Suspended
	)

	tests := []struct {
		name     string
		typ      reflect.Type
		tags     map[string]string
		expected string
	}{
		{
			name: "Enum from Tags",
			typ:  reflect.TypeOf(""),
			tags: map[string]string{
				"enum": "active|inactive|suspended",
			},
			expected: "ENUM('active','inactive','suspended')",
		},
		{
			name:     "String Type without Enum",
			typ:      reflect.TypeOf(""),
			tags:     map[string]string{},
			expected: "",
		},
		{
			name:     "Custom Type",
			typ:      reflect.TypeOf(Status(0)),
			tags:     map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).parseEnumType(tt.typ, tt.tags)
			if got != tt.expected {
				t.Errorf("parseEnumType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSQLType(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{
			name:     "String Type",
			typ:      reflect.TypeOf(""),
			expected: "VARCHAR(255)",
		},
		{
			name:     "Int8 Type",
			typ:      reflect.TypeOf(int8(0)),
			expected: "TINYINT",
		},
		{
			name:     "Int16 Type",
			typ:      reflect.TypeOf(int16(0)),
			expected: "SMALLINT",
		},
		{
			name:     "Int32 Type",
			typ:      reflect.TypeOf(int32(0)),
			expected: "INT",
		},
		{
			name:     "Int64 Type",
			typ:      reflect.TypeOf(int64(0)),
			expected: "BIGINT",
		},
		{
			name:     "Uint8 Type",
			typ:      reflect.TypeOf(uint8(0)),
			expected: "TINYINT UNSIGNED",
		},
		{
			name:     "Uint16 Type",
			typ:      reflect.TypeOf(uint16(0)),
			expected: "SMALLINT UNSIGNED",
		},
		{
			name:     "Uint32 Type",
			typ:      reflect.TypeOf(uint32(0)),
			expected: "INT UNSIGNED",
		},
		{
			name:     "Uint64 Type",
			typ:      reflect.TypeOf(uint64(0)),
			expected: "BIGINT UNSIGNED",
		},
		{
			name:     "Float32 Type",
			typ:      reflect.TypeOf(float32(0)),
			expected: "FLOAT",
		},
		{
			name:     "Float64 Type",
			typ:      reflect.TypeOf(float64(0)),
			expected: "DOUBLE",
		},
		{
			name:     "Bool Type",
			typ:      reflect.TypeOf(false),
			expected: "TINYINT(1)",
		},
		{
			name:     "Time Type",
			typ:      reflect.TypeOf(time.Time{}),
			expected: "DATETIME",
		},
		{
			name:     "Slice Type",
			typ:      reflect.TypeOf([]string{}),
			expected: "JSON",
		},
		{
			name:     "Map Type",
			typ:      reflect.TypeOf(map[string]interface{}{}),
			expected: "JSON",
		},
		{
			name:     "Bytes Type",
			typ:      reflect.TypeOf([]byte{}),
			expected: "BLOB",
		},
		{
			name:     "Pointer Type",
			typ:      reflect.TypeOf(&struct{}{}),
			expected: "JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).getSQLType(tt.typ)
			if got != tt.expected {
				t.Errorf("getSQLType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseASTExpr(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		expr     ast.Expr
		expected string
	}{
		{
			name:     "Basic Identifier",
			expr:     &ast.Ident{Name: "string"},
			expected: "VARCHAR(255)",
		},
		{
			name:     "Array Type",
			expr:     &ast.ArrayType{Elt: &ast.Ident{Name: "string"}},
			expected: "JSON",
		},
		{
			name:     "Map Type",
			expr:     &ast.MapType{Key: &ast.Ident{Name: "string"}, Value: &ast.Ident{Name: "interface"}},
			expected: "JSON",
		},
		{
			name: "Time Type",
			expr: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Time"},
			},
			expected: "DATETIME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).parseASTExpr(tt.expr)
			if got != tt.expected {
				t.Errorf("parseASTExpr() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNullableFromTags(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		tags     map[string]string
		expected bool
	}{
		{
			name: "Explicitly Nullable",
			tags: map[string]string{
				"nullable": "",
			},
			expected: true,
		},
		{
			name: "Explicitly Not Null",
			tags: map[string]string{
				"not null": "",
			},
			expected: false,
		},
		{
			name:     "No Tags",
			tags:     map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).isNullableFromTags(tt.tags)
			if got != tt.expected {
				t.Errorf("isNullableFromTags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseForeignKeyFromTags(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name      string
		fieldName string
		tags      map[string]string
		expected  *ForeignKey
	}{
		{
			name:      "Basic Foreign Key",
			fieldName: "UserID",
			tags: map[string]string{
				"rel": "users,id",
			},
			expected: &ForeignKey{
				Name:            "fk_UserID_users",
				Column:          "UserID",
				ReferenceTable:  "users",
				ReferenceColumn: "id",
				OnDelete:        "CASCADE",
				OnUpdate:        "CASCADE",
			},
		},
		{
			name:      "Foreign Key with Options",
			fieldName: "UserID",
			tags: map[string]string{
				"rel": "users,id,ondelete=SET NULL,onupdate=SET NULL",
			},
			expected: &ForeignKey{
				Name:            "fk_UserID_users",
				Column:          "UserID",
				ReferenceTable:  "users",
				ReferenceColumn: "id",
				OnDelete:        "SET NULL",
				OnUpdate:        "SET NULL",
			},
		},
		{
			name:      "No Relation Tag",
			fieldName: "UserID",
			tags:      map[string]string{},
			expected:  nil,
		},
		{
			name:      "Invalid Relation Format",
			fieldName: "UserID",
			tags: map[string]string{
				"rel": "users",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).parseForeignKeyFromTags(tt.fieldName, tt.tags)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseForeignKeyFromTags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Implementasi EnumType untuk testing
type TestEnum int

const (
	EnumOne TestEnum = iota
	EnumTwo
	EnumThree
)

func (e TestEnum) Values() []string {
	return []string{"one", "two", "three"}
}

func (e TestEnum) String() string {
	return e.Values()[e]
}

func TestParseEnumTypeWithImplementation(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		typ      reflect.Type
		tags     map[string]string
		expected string
	}{
		{
			name:     "Enum Type Implementation",
			typ:      reflect.TypeOf(TestEnum(0)),
			tags:     map[string]string{},
			expected: "ENUM('one','two','three')",
		},
		{
			name: "Enum from Tags Override",
			typ:  reflect.TypeOf(TestEnum(0)),
			tags: map[string]string{
				"enum": "a|b|c",
			},
			expected: "ENUM('a','b','c')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).parseEnumType(tt.typ, tt.tags)
			if got != tt.expected {
				t.Errorf("parseEnumType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic Word",
			input:    "user",
			expected: "users",
		},
		{
			name:     "Word Ending in s",
			input:    "status",
			expected: "statuses",
		},
		{
			name:     "Empty String",
			input:    "",
			expected: "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluralize(tt.input)
			if got != tt.expected {
				t.Errorf("pluralize() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple Camel Case",
			input:    "userName",
			expected: "user_name",
		},
		{
			name:     "Multiple Upper Case",
			input:    "UserAPIKey",
			expected: "user_api_key",
		},
		{
			name:     "Already Snake Case",
			input:    "user_name",
			expected: "user_name",
		},
		{
			name:     "Single Letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "Empty String",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("toSnakeCase() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMapGoTypeToSQL(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{
			name:     "String Type",
			goType:   "string",
			expected: "VARCHAR(255)",
		},
		{
			name:     "Int Type",
			goType:   "int",
			expected: "INT",
		},
		{
			name:     "Int32 Type",
			goType:   "int32",
			expected: "INT",
		},
		{
			name:     "Int64 Type",
			goType:   "int64",
			expected: "BIGINT",
		},
		{
			name:     "Float32 Type",
			goType:   "float32",
			expected: "FLOAT",
		},
		{
			name:     "Float64 Type",
			goType:   "float64",
			expected: "DOUBLE",
		},
		{
			name:     "Bool Type",
			goType:   "bool",
			expected: "TINYINT(1)",
		},
		{
			name:     "Unknown Type",
			goType:   "unknown",
			expected: "TEXT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).mapGoTypeToSQL(tt.goType)
			if got != tt.expected {
				t.Errorf("mapGoTypeToSQL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetJSONFieldSchema(t *testing.T) {
	parser := NewParser()

	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	tests := []struct {
		name     string
		typ      reflect.Type
		expected map[string]interface{}
	}{
		{
			name: "String Type",
			typ:  reflect.TypeOf(""),
			expected: map[string]interface{}{
				"type": "string",
			},
		},
		{
			name: "Int Type",
			typ:  reflect.TypeOf(0),
			expected: map[string]interface{}{
				"type": "integer",
			},
		},
		{
			name: "Float Type",
			typ:  reflect.TypeOf(0.0),
			expected: map[string]interface{}{
				"type": "number",
			},
		},
		{
			name: "Bool Type",
			typ:  reflect.TypeOf(true),
			expected: map[string]interface{}{
				"type": "boolean",
			},
		},
		{
			name: "Time Type",
			typ:  reflect.TypeOf(time.Time{}),
			expected: map[string]interface{}{
				"type":   "string",
				"format": "date-time",
			},
		},
		{
			name: "Slice Type",
			typ:  reflect.TypeOf([]string{}),
			expected: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		{
			name: "Map Type",
			typ:  reflect.TypeOf(map[string]interface{}{}),
			expected: map[string]interface{}{
				"type":                 "object",
				"additionalProperties": map[string]interface{}{},
			},
		},
		{
			name: "Bytes Type",
			typ:  reflect.TypeOf([]byte{}),
			expected: map[string]interface{}{
				"type":            "string",
				"contentEncoding": "base64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.(*DefaultParser).getJSONFieldSchema(tt.typ)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("getJSONFieldSchema() = %v, want %v", got, tt.expected)
			}
		})
	}
}
