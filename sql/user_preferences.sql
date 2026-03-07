CREATE TABLE IF NOT EXISTS `user_preferences`
(
    `username`          VARCHAR(20) NOT NULL,
    `preference_key`    VARCHAR(50) NOT NULL,
    `preference_value`  TEXT        NOT NULL,
    `updated_at`        TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`username`, `preference_key`),
    FOREIGN KEY (`username`) REFERENCES `users` (`username`) ON UPDATE CASCADE ON DELETE CASCADE
)
