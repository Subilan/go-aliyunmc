-- create "task_output_lines" table
CREATE TABLE `task_output_lines` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_id` bigint unsigned NOT NULL,
  `seq` bigint unsigned NOT NULL,
  `step` bigint unsigned NOT NULL DEFAULT 0,
  `output` longtext NOT NULL,
  `created_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_task_output_line_task_seq` (`task_id`, `seq`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
