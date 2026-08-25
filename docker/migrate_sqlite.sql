-- =========================================
-- One-API 数据库迁移脚本 - SQLite
-- =========================================
-- 用途：从旧版本迁移到新版本，添加必要的字段
-- 执行方式：sqlite3 one-api.db < migrate_sqlite.sql

-- 开始事务
BEGIN TRANSACTION;

-- 检查并添加 users 表的 rpm 字段
-- SQLite 不支持 IF NOT EXISTS，所以我们使用 PRAGMA 来检查
-- 如果字段已存在，ALTER TABLE 会失败但不会影响其他操作

-- 添加用户速率限制字段
-- 注意：如果字段已存在，这些语句会失败，但可以忽略错误
ALTER TABLE users ADD COLUMN rpm INTEGER DEFAULT 0; -- 每分钟请求数限制
ALTER TABLE users ADD COLUMN rpd INTEGER DEFAULT 0; -- 每天请求数限制
ALTER TABLE users ADD COLUMN rpw INTEGER DEFAULT 0; -- 每周请求数限制

-- 确保 tokens 表有 used_quota 字段
ALTER TABLE tokens ADD COLUMN used_quota INTEGER DEFAULT 0; -- 已使用配额

-- 提交事务
COMMIT;

-- 查看表结构
PRAGMA table_info(users);
PRAGMA table_info(tokens);

-- 显示完成信息
SELECT 'Migration completed! Note: If you see errors about existing columns, that is normal.' AS status;
