-- migrate:up

CREATE TABLE `users` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(255) NOT NULL,
  `email` VARCHAR(255) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `phone_number` VARCHAR(20) NULL,
  `address` TEXT NULL,
  `status` VARCHAR(50) NOT NULL DEFAULT 'user',
  `role` VARCHAR(50) NOT NULL DEFAULT 'user',
  `is_active` TINYINT(1) NOT NULL DEFAULT 0,
  `last_login_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` TIMESTAMP NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username_unique` (`username`),
  UNIQUE KEY `idx_users_email_unique` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `profiles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` INT UNSIGNED NOT NULL,
  `bio` TEXT NULL,
  `avatar` VARCHAR(255) NULL,
  `location` VARCHAR(255) NULL,
  `website` VARCHAR(255) NULL,
  `phone` VARCHAR(20) NULL,
  `is_verified` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_profiles_user_id` (`user_id`),
  CONSTRAINT `fk_profiles_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- migrate:down

DROP TABLE IF EXISTS `profiles`;
DROP TABLE IF EXISTS `users`;