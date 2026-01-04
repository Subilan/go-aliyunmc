CREATE TABLE IF NOT EXISTS instances
(
    -- 实例的标识符，由阿里云返回，在这里也用作主键
    instance_id   VARCHAR(50) PRIMARY KEY,

    -- 实例类型
    instance_type VARCHAR(20) NOT NULL,

    -- 实例所在的地域
    region_id     VARCHAR(15) NOT NULL,

    -- 实例所在的可用区
    zone_id       VARCHAR(20) NOT NULL,

    -- 实例被分配的公网 IP 地址
    ip            VARCHAR(50)          DEFAULT NULL,

    -- 实例被删除的时间。如果实例没有被删除，为空
    deleted_at    TIMESTAMP            DEFAULT NULL,

    -- 实例被创建的时间，以该记录被插入的时间来替代表示
    created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 是否已部署
    deployed      TINYINT(1)  NOT NULL DEFAULT 0
);