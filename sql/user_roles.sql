CREATE TABLE IF NOT EXISTS `user_roles`
(
    `user_id` int         NOT NULL,
    `role`    varchar(20) NOT NULL,
    PRIMARY KEY (`user_id`, `role`)
);