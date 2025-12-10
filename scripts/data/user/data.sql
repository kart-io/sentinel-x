

CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(64) NOT NULL COMMENT '用户名',
  `email` VARCHAR(128) DEFAULT NULL COMMENT '邮箱',
  `password` VARCHAR(255) NOT NULL COMMENT '密码Hash',
  `avatar` VARCHAR(255) DEFAULT NULL COMMENT '头像URL',
  `mobile` VARCHAR(20) DEFAULT NULL COMMENT '手机号',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1启用 0禁用',
  `created_at` BIGINT NOT NULL COMMENT '创建时间(时间戳)',
  `updated_at` BIGINT NOT NULL COMMENT '更新时间(时间戳)',
  `created_by` BIGINT UNSIGNED DEFAULT 0 COMMENT '创建人',
  `updated_by` BIGINT UNSIGNED DEFAULT 0 COMMENT '更新人',
  `deleted_at` BIGINT DEFAULT NULL COMMENT '软删除时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  UNIQUE KEY `uk_email` (`email`),
  KEY `idx_mobile` (`mobile`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
