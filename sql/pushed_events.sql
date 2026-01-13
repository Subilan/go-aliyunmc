CREATE TABLE IF NOT EXISTS `pushed_events`
(
    `task_id`    VARCHAR(36) COMMENT '该事件关联的任务ID',
    `ord`        INT COMMENT '事件的顺序',
    `type`       TINYINT   NOT NULL COMMENT '事件来源或事件的任务类型',
    `is_error`   SMALLINT(1)        DEFAULT 0 COMMENT '是否为反映错误的事件',
    `is_public`  SMALLINT(1)        DEFAULT 0 COMMENT '是否可被未登录用户读取',
    `content`    TEXT      NOT NULL COMMENT '推送内容',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks (task_id)
)