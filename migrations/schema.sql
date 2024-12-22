CREATE TABLE "users" (
  "id" bigserial,
  "username" varchar(100) NOT NULL,
  "email" varchar(255) NOT NULL,
  "password" varchar(255) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "last_login_at" timestamp with time zone,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" timestamp with time zone,
  "last_location" varchar(255),
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uni_users_email" ON "users" ("email");

CREATE UNIQUE INDEX IF NOT EXISTS "uni_users_username" ON "users" ("username");

CREATE TABLE "profiles" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "bio" varchar(500),
  "phone_number" varchar(20),
  "is_verified" boolean NOT NULL DEFAULT false,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "avatar" varchar(255),
  "address" varchar(1000),
  "website" varchar(255),
  "notes" text,
  "level" integer DEFAULT 1,
  "experience" bigint DEFAULT 0,
  "title" varchar(100),
  "badges" text[],
  "settings" jsonb,
  "metadata" jsonb,
  "tags" text[],
  "status" varchar(50) DEFAULT 'active',
  "score" decimal(10,
  2) DEFAULT 0,
  "rating" decimal(5,
  2) DEFAULT 0,
  "points" bigint DEFAULT 100,
  "balance" decimal(15,
  4) DEFAULT 0,
  "weight" decimal(8,
  3) DEFAULT 0,
  "height" decimal(6,
  2) DEFAULT 0,
  "age" smallint DEFAULT 0,
  "price" decimal(12,
  2) DEFAULT 0,
  "tax" decimal(8,
  4) DEFAULT 0,
  "discount" decimal(5,
  2) DEFAULT 0,
  "quantity" integer DEFAULT 1,
  PRIMARY KEY ("id")
);