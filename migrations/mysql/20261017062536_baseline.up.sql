-- create "balance_samples" table
CREATE TABLE `balance_samples` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL,
  `amount` double NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_balance_samples_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "changelogs" table
CREATE TABLE `changelogs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `title` longtext NOT NULL,
  `body` longtext NOT NULL,
  `category` varchar(191) NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_changelogs_category` (`category`),
  INDEX `idx_changelogs_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "instances" table
CREATE TABLE `instances` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `instance_id` longtext NOT NULL,
  `instance_type` longtext NOT NULL,
  `region_id` longtext NOT NULL,
  `zone_id` longtext NOT NULL,
  `v_switch_id` longtext NOT NULL,
  `ip` varchar(191) NULL,
  `is_deployed` bool NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_instances_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "player_list_samples" table
CREATE TABLE `player_list_samples` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL,
  `player_names` longtext NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_player_list_samples_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "user_preferences" table
CREATE TABLE `user_preferences` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `user_id` bigint unsigned NOT NULL,
  `data` longtext NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_user_preferences_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_user_preferences_user_id` (`user_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "users" table
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `username` varchar(191) NULL,
  `password_hash` longtext NOT NULL,
  `role` varchar(191) NULL DEFAULT "",
  `banned` bool NULL DEFAULT 0,
  `whitelist_uuid` varchar(191) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_username_deleted_at` (`deleted_at`, `username`),
  INDEX `idx_users_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_whitelist_uuid` (`whitelist_uuid`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "changelog_likes" table
CREATE TABLE `changelog_likes` (
  `changelog_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  PRIMARY KEY (`changelog_id`, `user_id`),
  INDEX `fk_changelog_likes_user` (`user_id`),
  CONSTRAINT `fk_changelog_likes_changelog` FOREIGN KEY (`changelog_id`) REFERENCES `changelogs` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_changelog_likes_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- create "tasks" table
CREATE TABLE `tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `type` longtext NOT NULL,
  `start_at` datetime(3) NULL,
  `end_at` datetime(3) NULL,
  `status` longtext NOT NULL,
  `step` bigint unsigned NULL,
  `output` longtext NOT NULL,
  `error` varchar(191) NULL,
  `by` bigint unsigned NULL,
  PRIMARY KEY (`id`),
  INDEX `fk_tasks_user` (`by`),
  INDEX `idx_tasks_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_tasks_user` FOREIGN KEY (`by`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
