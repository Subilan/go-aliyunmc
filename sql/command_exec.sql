CREATE TABLE IF NOT EXISTS command_exec
(
    `id`         INT AUTO_INCREMENT PRIMARY KEY,
    `type`       VARCHAR(20) NOT NULL COMMENT '指令类型',
    `by`         INT COMMENT '执行者',
    `auto`       TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '是否为自动执行',
    `created_at` TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `status`     VARCHAR(20) NOT NULL COMMENT '执行状态',
    `comment`    TEXT COMMENT '注释',
    FOREIGN KEY (`by`) REFERENCES `users` (`id`)
)