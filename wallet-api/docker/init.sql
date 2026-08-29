-- 初始化脚本（可选）
-- 如果你已经有 one-api 数据库，可以忽略此文件

-- 此文件仅用于全新部署时创建必要的表结构
-- wallet-api 会自动迁移 transactions 和 subscriptions 表

USE oneapi;

-- 设置字符集
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;
