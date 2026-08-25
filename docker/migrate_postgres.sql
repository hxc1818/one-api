-- =========================================
-- One-API 数据库迁移脚本 - PostgreSQL
-- =========================================
-- 用途：从旧版本迁移到新版本，添加必要的字段
-- 执行方式：psql -U user -d oneapi -f migrate_postgres.sql

-- 添加用户速率限制字段
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='rpm') THEN
        ALTER TABLE users ADD COLUMN rpm INT DEFAULT 0;
        COMMENT ON COLUMN users.rpm IS '每分钟请求数限制，0表示无限制';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='rpd') THEN
        ALTER TABLE users ADD COLUMN rpd INT DEFAULT 0;
        COMMENT ON COLUMN users.rpd IS '每天请求数限制，0表示无限制';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='rpw') THEN
        ALTER TABLE users ADD COLUMN rpw INT DEFAULT 0;
        COMMENT ON COLUMN users.rpw IS '每周请求数限制，0表示无限制';
    END IF;
END $$;

-- 确保 tokens 表有 used_quota 字段
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='tokens' AND column_name='used_quota') THEN
        ALTER TABLE tokens ADD COLUMN used_quota BIGINT DEFAULT 0;
        COMMENT ON COLUMN tokens.used_quota IS '已使用配额';
    END IF;
END $$;

-- 查看修改结果
\d users
\d tokens

-- 显示迁移完成信息
SELECT 'Migration completed successfully!' AS status;
