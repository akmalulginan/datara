package schema

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/akmalulginan/datara/internal/state"
)

// Generator menangani konversi dari struct Go ke schema database
type Generator struct {
	config *Config
}

// Config menyimpan konfigurasi untuk generator
type Config struct {
	TablePrefix  string
	TableSuffix  string
	UseSnakeCase bool
	UsePlural    bool
}

// NewGenerator membuat instance baru dari Generator
func NewGenerator(config *Config) *Generator {
	if config == nil {
		config = &Config{
			UseSnakeCase: true,
			UsePlural:    true,
		}
	}
	return &Generator{config: config}
}

// GenerateSchema mengkonversi struct Go ke SchemaState
func (g *Generator) GenerateSchema(models ...interface{}) (*state.SchemaState, error) {
	schema := state.NewSchemaState()

	for _, model := range models {
		table, err := g.generateTable(model)
		if err != nil {
			return nil, fmt.Errorf("failed to generate table for model: %w", err)
		}
		schema.AddTable(table)
	}

	return schema, nil
}

// generateTable mengkonversi struct ke Table
func (g *Generator) generateTable(model interface{}) (state.Table, error) {
	modelInfo, ok := model.(*struct {
		Name   string
		Fields map[string]interface{}
	})
	if !ok {
		return state.Table{}, fmt.Errorf("invalid model format")
	}

	tableName := g.formatTableName(modelInfo.Name)
	table := state.Table{
		Name:        tableName,
		Columns:     make(map[string]state.Column),
		Indexes:     make(map[string]state.Index),
		Constraints: make([]state.Constraint, 0),
	}

	for fieldName, fieldInfo := range modelInfo.Fields {
		info, ok := fieldInfo.(map[string]interface{})
		if !ok {
			continue
		}

		// Generate column
		column := g.generateColumnFromInfo(fieldName, info)
		table.Columns[column.Name] = column

		// Check untuk index dan constraints dari db_tag
		if dbTag, ok := info["db_tag"].(string); ok {
			if idx := g.generateIndexFromTag(fieldName, dbTag); idx != nil {
				table.Indexes[idx.Name] = *idx
			}

			if constraint := g.generateConstraintFromTag(fieldName, dbTag); constraint != nil {
				table.Constraints = append(table.Constraints, *constraint)
			}
		}
	}

	return table, nil
}

// generateColumnFromInfo membuat Column dari informasi field
func (g *Generator) generateColumnFromInfo(fieldName string, info map[string]interface{}) state.Column {
	fieldType, _ := info["type"].(string)

	column := state.Column{
		Name:     g.getColumnName(fieldName),
		Type:     g.getSQLTypeFromGoType(fieldType),
		Nullable: g.isNullableType(fieldType),
	}

	// Parse db_tag untuk opsi tambahan
	if dbTag, ok := info["db_tag"].(string); ok {
		parts := strings.Split(dbTag, ",")
		for _, part := range parts {
			switch {
			case part == "auto_increment":
				column.AutoIncrement = true
			case strings.HasPrefix(part, "default="):
				column.DefaultValue = strings.TrimPrefix(part, "default=")
			}
		}
	}

	return column
}

// getSQLTypeFromGoType mengkonversi tipe Go ke tipe SQL
func (g *Generator) getSQLTypeFromGoType(goType string) string {
	switch goType {
	case "bool":
		return "BOOLEAN"
	case "int", "int32":
		return "INTEGER"
	case "int64", "uint", "uint32", "uint64":
		return "BIGINT"
	case "float32":
		return "FLOAT"
	case "float64":
		return "DOUBLE"
	case "string":
		return "VARCHAR(255)"
	case "*time.Time", "time.Time":
		return "DATETIME"
	default:
		return "TEXT"
	}
}

// isNullableType menentukan apakah tipe bisa null
func (g *Generator) isNullableType(goType string) bool {
	return strings.HasPrefix(goType, "*")
}

// formatTableName memformat nama tabel sesuai konfigurasi
func (g *Generator) formatTableName(name string) string {
	if g.config.UseSnakeCase {
		name = toSnakeCase(name)
	}
	if g.config.UsePlural {
		name = pluralize(name)
	}
	return g.config.TablePrefix + name + g.config.TableSuffix
}

// generateIndexFromTag membuat Index dari tag
func (g *Generator) generateIndexFromTag(fieldName, tag string) *state.Index {
	if strings.Contains(tag, "index") || strings.Contains(tag, "unique") {
		name := fmt.Sprintf("idx_%s", g.getColumnName(fieldName))
		return &state.Index{
			Name:    name,
			Columns: []string{g.getColumnName(fieldName)},
			Unique:  strings.Contains(tag, "unique"),
		}
	}
	return nil
}

// generateConstraintFromTag membuat Constraint dari tag
func (g *Generator) generateConstraintFromTag(fieldName, tag string) *state.Constraint {
	if strings.Contains(tag, "primary_key") {
		return &state.Constraint{
			Name: fmt.Sprintf("pk_%s", g.getColumnName(fieldName)),
			Type: "PRIMARY KEY",
			Def:  fmt.Sprintf("PRIMARY KEY (`%s`)", g.getColumnName(fieldName)),
		}
	}
	return nil
}

// getColumnName mengkonversi nama field ke nama kolom
func (g *Generator) getColumnName(name string) string {
	if g.config.UseSnakeCase {
		name = toSnakeCase(name)
	}
	return name
}

// toSnakeCase mengkonversi string ke snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// pluralize menambahkan 's' di akhir string
// TODO: Implementasi pluralization yang lebih baik
func pluralize(s string) string {
	return s + "s"
}
