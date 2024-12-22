// Schema configuration
schema {
  program = [
    "go",
    "run",
    "./main/register.go"
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