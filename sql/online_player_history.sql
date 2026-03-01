CREATE TABLE online_player_history
(
    `id`           INT PRIMARY KEY AUTO_INCREMENT,
    `created_at`   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `player_count` INT       NOT NULL,
    `players`      TEXT      NOT NULL
)