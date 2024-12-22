# Datara

Library Go untuk mengkonversi struct Go menjadi skema migrasi database.

## Instalasi

```bash
go get -u github.com/gin/datara
```

## Penggunaan

1. Buat file `register.go` di project Anda:

```go
package main

import (
    "encoding/json"
    "os"
    "your/models"
)

func main() {
    schema := // Convert your models to schema
    json.NewEncoder(os.Stdout).Encode(schema)
}
```

2. Buat file `datara.hcl` untuk konfigurasi:

```hcl
// Schema configuration
schema {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./register",
  ]
}

// Migration settings
migration {
  dir = "migrations"
  format = "sql"
  charset = "utf8mb4"
  collation = "utf8mb4_unicode_ci"
  engine = "InnoDB"
}

// Table naming strategy
naming {
  table {
    plural = true      // Users instead of User
    snake_case = true  // user_profiles instead of UserProfiles
  }
  column {
    snake_case = true  // created_at instead of CreatedAt
  }
}

// Custom type mappings
types {
  "time.Time" = "TIMESTAMP"
  "uuid.UUID" = "VARCHAR(36)"
  "*string" = "TEXT"
  "*int" = "INT"
}

// Versioning strategy
versioning {
  strategy = "timestamp"
  format = "20060102150405"
  prefix = "v"
  suffix = "_migration"
}

// Indexes configuration
indexes {
  naming_template = "idx_{{.TableName}}_{{.ColumnName}}"
  foreign_key_template = "fk_{{.TableName}}_{{.RefTable}}"
}

// Default column values
defaults {
  created_at = "CURRENT_TIMESTAMP"
  updated_at = "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
  deleted_at = "NULL"
}

// Soft delete configuration
soft_delete {
  enabled = true
  column = "deleted_at"
  type = "TIMESTAMP"
}
```

3. Generate migrasi:

```bash
datara -config datara.hcl
```

## Fitur

- Konversi otomatis dari struct Go ke skema database
- Dukungan untuk berbagai tipe data SQL
- Penanganan field nullable
- Dukungan untuk nilai default
- Dukungan untuk tipe enum
- Dukungan untuk foreign key dan relasi
- Dukungan untuk indeks tabel
- Validasi JSON schema
- Konfigurasi fleksibel melalui HCL
- Format output SQL dan JSON

## Contoh Model

```go
type User struct {
    ID        uint      `db:"primary_key,autoincrement"`
    Username  string    `db:"type=VARCHAR(100),unique"`
    Email     string    `db:"type=VARCHAR(255),unique"`
    Password  string    `db:"type=VARCHAR(255)"`
    Profile   *Profile  `rel:"profiles,id,ondelete=CASCADE"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time
}

type Profile struct {
    ID     uint   `db:"primary_key,autoincrement"`
    UserID uint   `db:"type=INT"`
    Bio    string `db:"type=TEXT,nullable"`
}
```

## Lisensi

MIT License