CREATE TABLE IF NOT EXISTS bills
(
    `usage_end`    TIMESTAMP   NOT NULL,
    `usage_start`  TIMESTAMP   NOT NULL,
    `product_code` VARCHAR(10) NOT NULL,
    `product_name` VARCHAR(32) NOT NULL,
    `pay`          FLOAT       NOT NULL,
    `status`       VARCHAR(20) NOT NULL
)