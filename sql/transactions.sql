CREATE TABLE IF NOT EXISTS transactions
(
    `amount`        FLOAT       NOT NULL,
    `balance`       FLOAT       NOT NULL,
    `time`          TIMESTAMP   NOT NULL,
    `flow`          VARCHAR(12) NOT NULL,
    `type`          VARCHAR(12) NOT NULL,
    `remarks`       VARCHAR(32),
    `billing_cycle` VARCHAR(10) NOT NULL,
    UNIQUE INDEX `unique_transaction` (`amount`, `balance`, `time`, `flow`, `type`, `remarks`)
)