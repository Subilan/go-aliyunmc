CREATE TABLE IF NOT EXISTS instance_statuses
(
    -- 实例标识符
    `instance_id` VARCHAR(50) NOT NULL,

    -- 实例状态
    `status`      VARCHAR(15) NOT NULL,

    -- 状态更新时间，初始值为该记录的创建时间
    `updated_at`  TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    CONSTRAINT `fk_instance_statuses_instance_id` FOREIGN KEY (instance_id) REFERENCES instances (instance_id) ON UPDATE CASCADE ON DELETE CASCADE
);