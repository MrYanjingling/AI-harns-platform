CREATE TABLE IF NOT EXISTS `thing_models` (
  `id`              CHAR(32)     NOT NULL COMMENT 'ULID',
  `version`         BIGINT       NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',
  `thing_model_key` VARCHAR(64)  COLLATE utf8mb4_bin NOT NULL COMMENT '物模型 Key，全局唯一，区分大小写',
  `name`            VARCHAR(64)  NOT NULL COMMENT '物模型名称',
  `description`     VARCHAR(255) NOT NULL DEFAULT '' COMMENT '物模型描述',
  `features`        JSON         NULL COMMENT '特征定义列表',
  `created_user`    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_user`    VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '更新人',
  `created_time`    DATETIME     NOT NULL COMMENT '创建时间',
  `updated_time`    DATETIME     NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_thing_models_key` (`thing_model_key`),
  UNIQUE KEY `uk_thing_models_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='物模型';

CREATE TABLE IF NOT EXISTS `thing_model_properties` (
  `id`             CHAR(32)     NOT NULL COMMENT 'ULID',
  `version`        BIGINT       NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',
  `thing_model_id` CHAR(32)     NOT NULL COMMENT '所属物模型 ID',
  `identifier`     VARCHAR(64)  NOT NULL COMMENT '属性唯一标识',
  `name`           VARCHAR(64)  NOT NULL COMMENT '属性名称',
  `data_type`      JSON         NOT NULL COMMENT '数据类型 DataTypeDetail',
  `rw_flag`        VARCHAR(8)   NOT NULL DEFAULT 'r' COMMENT '读写标志 r/w/rw',
  `required`       TINYINT      NOT NULL DEFAULT 0 COMMENT '是否必填',
  `description`    VARCHAR(255) NOT NULL DEFAULT '' COMMENT '属性描述',
  `custom`         JSON         NULL COMMENT '自定义信息',
  `sort`           INT          NOT NULL DEFAULT 0 COMMENT '展示顺序',
  `created_user`   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '创建人',
  `updated_user`   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '更新人',
  `created_time`   DATETIME     NOT NULL COMMENT '创建时间',
  `updated_time`   DATETIME     NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tmp_model_identifier` (`thing_model_id`, `identifier`),
  UNIQUE KEY `uk_tmp_model_name` (`thing_model_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='物模型属性';
