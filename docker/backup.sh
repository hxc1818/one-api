#!/bin/bash

# =========================================
# One-API 数据备份脚本
# =========================================

set -e

BACKUP_DIR="./backup"
DATE=$(date +%Y%m%d_%H%M%S)

echo "======================================"
echo "One-API 数据备份脚本"
echo "======================================"
echo ""

# 创建备份目录
mkdir -p "$BACKUP_DIR"

echo "备份目录: $BACKUP_DIR"
echo "时间戳: $DATE"
echo ""

# 检测使用的数据库类型
if [ -f "./data/one-api.db" ]; then
    echo "检测到 SQLite 数据库"
    echo "正在备份..."
    
    cp "./data/one-api.db" "$BACKUP_DIR/one-api_${DATE}.db"
    
    echo "备份完成: $BACKUP_DIR/one-api_${DATE}.db"
    echo "文件大小: $(du -h $BACKUP_DIR/one-api_${DATE}.db | cut -f1)"
    
elif docker-compose ps | grep -q mysql; then
    echo "检测到 MySQL 数据库"
    echo "正在备份..."
    
    docker-compose exec -T mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD:-password} oneapi > "$BACKUP_DIR/oneapi_mysql_${DATE}.sql"
    
    echo "备份完成: $BACKUP_DIR/oneapi_mysql_${DATE}.sql"
    echo "文件大小: $(du -h $BACKUP_DIR/oneapi_mysql_${DATE}.sql | cut -f1)"
    
elif docker-compose ps | grep -q postgres; then
    echo "检测到 PostgreSQL 数据库"
    echo "正在备份..."
    
    docker-compose exec -T postgres pg_dump -U ${POSTGRES_USER:-user} oneapi > "$BACKUP_DIR/oneapi_postgres_${DATE}.sql"
    
    echo "备份完成: $BACKUP_DIR/oneapi_postgres_${DATE}.sql"
    echo "文件大小: $(du -h $BACKUP_DIR/oneapi_postgres_${DATE}.sql | cut -f1)"
    
else
    echo "错误：未检测到数据库"
    exit 1
fi

# 备份配置文件
if [ -f ".env" ]; then
    echo ""
    echo "正在备份配置文件..."
    cp .env "$BACKUP_DIR/env_${DATE}.txt"
    echo "配置备份完成: $BACKUP_DIR/env_${DATE}.txt"
fi

# 清理旧备份（保留最近7天）
echo ""
echo "正在清理旧备份（保留7天）..."
find "$BACKUP_DIR" -name "*.db" -mtime +7 -delete
find "$BACKUP_DIR" -name "*.sql" -mtime +7 -delete
find "$BACKUP_DIR" -name "env_*.txt" -mtime +7 -delete

echo ""
echo "======================================"
echo "备份完成!"
echo "======================================"
echo ""
echo "备份文件位置: $BACKUP_DIR"
echo ""
echo "查看所有备份:"
ls -lh "$BACKUP_DIR" | tail -n +2
echo ""
