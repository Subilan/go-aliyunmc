CREATE TABLE IF NOT EXISTS `user_roles`
(
    `user_id` int         NOT NULL,
    `role`    int NOT NULL,
    PRIMARY KEY (`user_id`, `role`)
);