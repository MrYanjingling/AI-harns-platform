-- Bootstrap super administrator role and account.
-- Default account: admin / harns@iot (please change it after first login).
-- Role code "admin" is the reserved super administrator role code.

INSERT IGNORE INTO `roles`
  (`id`, `version`, `name`, `code`, `description`, `status`, `created_user`, `updated_user`, `created_time`, `updated_time`)
VALUES
  ('01M6KGSCACPFET7BPJG8TQRSP3', 0, '超级管理员', 'admin', '内置超级管理员', 'enabled', 'system', 'system', NOW(), NOW());

INSERT IGNORE INTO `users`
  (`id`, `version`, `user_name`, `password`, `user_no`, `nick_name`, `birth_date`, `gender`, `mail`, `phone`, `note`,
   `auth_key`, `auth_mode`, `last_login_time`, `login_failure_times`, `locked_until`, `expire_times`, `status`,
   `created_user`, `updated_user`, `created_time`, `updated_time`)
VALUES
  ('01M6KGSCACPFET7BPJGBQ15WMF', 0, 'admin',
   '$2y$10$yKFB6HeMaIzpkk6Lbl9HZOlFlJxais077EVLxYiso21Yvf30diNy6',
   'admin', '超级管理员', NULL, '', '', '', '', '', 'local', NULL, 0, NULL, 0, 'enabled',
   'system', 'system', NOW(), NOW());

INSERT IGNORE INTO `user_roles` (`user_id`, `role_id`, `created_time`)
VALUES ('01M6KGSCACPFET7BPJGBQ15WMF', '01M6KGSCACPFET7BPJG8TQRSP3', NOW());
