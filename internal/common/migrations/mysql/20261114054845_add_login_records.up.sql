-- create "login_records" table
CREATE TABLE `login_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_login_records_user_created` (`user_id`, `created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
