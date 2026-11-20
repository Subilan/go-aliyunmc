-- create "forum_profiles" table
CREATE TABLE `forum_profiles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `user_id` bigint unsigned NOT NULL,
  `nickname` varchar(32) NULL,
  `bio` varchar(200) NOT NULL DEFAULT '',
  `avatar_upload_id` bigint unsigned NULL,
  `background_upload_id` bigint unsigned NULL,
  `avatar_shape` varchar(8) NOT NULL DEFAULT 'circle',
  `permissions_initialized` bool NOT NULL DEFAULT false,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_profiles_user_id` (`user_id`),
  INDEX `idx_forum_profiles_deleted_at` (`deleted_at`),
  INDEX `idx_forum_profiles_avatar_upload_id` (`avatar_upload_id`),
  INDEX `idx_forum_profiles_background_upload_id` (`background_upload_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_boards" table
CREATE TABLE `forum_boards` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `slug` varchar(64) NOT NULL,
  `name` varchar(64) NOT NULL,
  `description` varchar(255) NOT NULL DEFAULT '',
  `icon` varchar(32) NOT NULL DEFAULT 'discussion',
  `color` varchar(7) NOT NULL DEFAULT '#6366f1',
  `sort_order` bigint NOT NULL DEFAULT 0,
  `enabled` bool NOT NULL DEFAULT true,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_boards_slug` (`slug`),
  INDEX `idx_forum_boards_deleted_at` (`deleted_at`),
  INDEX `idx_forum_boards_enabled_sort_order` (`enabled`, `sort_order`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_threads" table
CREATE TABLE `forum_threads` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `board_id` bigint unsigned NOT NULL,
  `author_id` bigint unsigned NOT NULL,
  `title` varchar(160) NOT NULL,
  `content_html` longtext NOT NULL,
  `content_text` mediumtext NOT NULL,
  `pinned` bool NOT NULL DEFAULT false,
  `pinned_at` datetime(3) NULL,
  `locked` bool NOT NULL DEFAULT false,
  `last_reply_at` datetime(3) NULL,
  `deleted_by` bigint unsigned NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_forum_threads_board_created` (`board_id`, `created_at`),
  INDEX `idx_forum_threads_author_created` (`author_id`, `created_at`),
  INDEX `idx_forum_threads_pinned_created` (`pinned`, `created_at`),
  INDEX `idx_forum_threads_board_pinned_created` (`board_id`, `pinned`, `created_at`),
  INDEX `idx_forum_threads_last_reply_at` (`last_reply_at`),
  INDEX `idx_forum_threads_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_replies" table
CREATE TABLE `forum_replies` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `thread_id` bigint unsigned NOT NULL,
  `author_id` bigint unsigned NOT NULL,
  `floor` int unsigned NOT NULL,
  `content_html` longtext NOT NULL,
  `content_text` mediumtext NOT NULL,
  `reply_to_floor` int unsigned NULL,
  `reply_to_reply_id` bigint unsigned NULL,
  `deleted_by` bigint unsigned NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_replies_thread_floor` (`thread_id`, `floor`),
  INDEX `idx_forum_replies_author_created` (`author_id`, `created_at`),
  INDEX `idx_forum_replies_reply_to_reply_id` (`reply_to_reply_id`),
  INDEX `idx_forum_replies_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_tags" table
CREATE TABLE `forum_tags` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL,
  `normalized` varchar(32) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_tags_normalized` (`normalized`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_thread_tags" table
CREATE TABLE `forum_thread_tags` (
  `thread_id` bigint unsigned NOT NULL,
  `tag_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`thread_id`, `tag_id`),
  INDEX `idx_forum_thread_tags_tag_id` (`tag_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_thread_likes" table
CREATE TABLE `forum_thread_likes` (
  `thread_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`thread_id`, `user_id`),
  INDEX `idx_forum_thread_likes_user_id` (`user_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_reply_likes" table
CREATE TABLE `forum_reply_likes` (
  `reply_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`reply_id`, `user_id`),
  INDEX `idx_forum_reply_likes_user_id` (`user_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_user_likes" table
CREATE TABLE `forum_user_likes` (
  `target_user_id` bigint unsigned NOT NULL,
  `liker_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`target_user_id`, `liker_id`),
  INDEX `idx_forum_user_likes_liker_id` (`liker_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_board_follows" table
CREATE TABLE `forum_board_follows` (
  `board_id` bigint unsigned NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`board_id`, `user_id`),
  INDEX `idx_forum_board_follows_user_id` (`user_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_notifications" table
CREATE TABLE `forum_notifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `recipient_id` bigint unsigned NOT NULL,
  `actor_id` bigint unsigned NOT NULL,
  `type` varchar(16) NOT NULL,
  `thread_id` bigint unsigned NOT NULL,
  `reply_id` bigint unsigned NOT NULL DEFAULT 0,
  `quoted_floor` int unsigned NOT NULL DEFAULT 0,
  `excerpt` varchar(500) NOT NULL DEFAULT '',
  `read_at` datetime(3) NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_notifications_unique` (`type`, `recipient_id`, `actor_id`, `thread_id`, `reply_id`),
  INDEX `idx_forum_notifications_recipient_created` (`recipient_id`, `created_at`),
  INDEX `idx_forum_notifications_recipient_read_created` (`recipient_id`, `read_at`, `created_at`),
  INDEX `idx_forum_notifications_actor_id` (`actor_id`),
  INDEX `idx_forum_notifications_thread_id` (`thread_id`),
  INDEX `idx_forum_notifications_reply_id` (`reply_id`),
  INDEX `idx_forum_notifications_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_uploads" table
CREATE TABLE `forum_uploads` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `owner_id` bigint unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `key` varchar(255) NOT NULL,
  `original_name` varchar(255) NOT NULL DEFAULT '',
  `mime` varchar(64) NOT NULL,
  `size_bytes` bigint NOT NULL,
  `width` bigint NOT NULL,
  `height` bigint NOT NULL,
  `sha256` char(64) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_uploads_key` (`key`),
  INDEX `idx_forum_uploads_owner_id` (`owner_id`),
  INDEX `idx_forum_uploads_sha256` (`sha256`),
  INDEX `idx_forum_uploads_deleted_at` (`deleted_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_user_permissions" table
CREATE TABLE `forum_user_permissions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `permission_code` varchar(128) NOT NULL,
  `scope_board_id` bigint unsigned NOT NULL DEFAULT 0,
  `granted_by` bigint unsigned NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_forum_user_permissions_unique` (`user_id`, `permission_code`, `scope_board_id`),
  INDEX `idx_forum_user_permissions_user_id` (`user_id`),
  INDEX `idx_forum_user_permissions_code` (`permission_code`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_settings" table
CREATE TABLE `forum_settings` (
  `key` varchar(64) NOT NULL,
  `value` text NOT NULL,
  `updated_by` bigint unsigned NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`key`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_audit_logs" table
CREATE TABLE `forum_audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `actor_id` bigint unsigned NOT NULL,
  `action` varchar(64) NOT NULL,
  `target_type` varchar(32) NOT NULL,
  `target_id` bigint unsigned NOT NULL DEFAULT 0,
  `detail` text NOT NULL,
  `ip` varchar(45) NOT NULL DEFAULT '',
  `user_agent` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_forum_audit_logs_actor_id` (`actor_id`),
  INDEX `idx_forum_audit_logs_action` (`action`),
  INDEX `idx_forum_audit_logs_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

-- create "forum_login_records" table
CREATE TABLE `forum_login_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `channel` varchar(16) NOT NULL,
  `ip` varchar(45) NOT NULL DEFAULT '',
  `user_agent` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_forum_login_records_user_id` (`user_id`),
  INDEX `idx_forum_login_records_created_at` (`created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;