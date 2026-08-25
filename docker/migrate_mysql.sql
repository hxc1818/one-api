-- =========================================
-- One-API 数据库迁移脚本 - MySQL/MariaDB
-- =========================================
-- 用途：从旧版本迁移到新版本，添加必要的字段
-- 执行方式：mysql -u root -p oneapi < migrate_mysql.sql

-- 添加用户速率限制字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS rpm INT DEFAULT 0 COMMENT '每分钟请求数限制，0表示无限制';
ALTER TABLE users ADD COLUMN IF NOT EXISTS rpd INT DEFAULT 0 COMMENT '每天请求数限制，0表示无限制';
ALTER TABLE users ADD COLUMN IF NOT EXISTS rpw INT DEFAULT 0 COMMENT '每周请求数限制，0表示无限制';

-- 确保 tokens 表有 used_quota 字段（如果已存在会跳过）
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS used_quota BIGINT DEFAULT 0 COMMENT '已使用配额';

-- 查看修改结果
DESCRIBE users;
DESCRIBE tokens;

-- 显示迁移完成信息
SELECT 'Migration completed successfully!' AS status;
