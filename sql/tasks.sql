CREATE TABLE IF NOT EXISTS `tasks`
(
    `task_id`    VARCHAR(36) PRIMARY KEY,
    `type`       VARCHAR(20) NOT NULL COMMENT '任务类型',
    `user_id`    INT         NOT NULL,
    `status`     VARCHAR(20) NOT NULL DEFAULT 'running' COMMENT '任务状态',
    `created_at` TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users` (id)
);