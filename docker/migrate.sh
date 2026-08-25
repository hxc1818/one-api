#!/bin/bash

# =========================================
# One-API 数据库迁移脚本
# =========================================

set -e

echo "======================================"
echo "One-API 数据库迁移向导"
echo "======================================"
echo ""

# 检测数据库类型
echo "请选择您的数据库类型："
echo "1) SQLite (默认)"
echo "2) MySQL/MariaDB"
echo "3) PostgreSQL"
echo ""
read -p "请输入选项 [1-3]: " DB_TYPE
DB_TYPE=${DB_TYPE:-1}

case $DB_TYPE in
    1)
        echo ""
        echo "选择: SQLite"
        echo "======================================"
        echo ""
        
        # 查找 SQLite 数据库文件
        if [ -f "./data/one-api.db" ]; then
            DB_FILE="./data/one-api.db"
        elif [ -f "one-api.db" ]; then
            DB_FILE="one-api.db"
        else
            read -p "请输入 SQLite 数据库文件路径: " DB_FILE
        fi
        
        if [ ! -f "$DB_FILE" ]; then
            echo "错误：数据库文件不存在: $DB_FILE"
            exit 1
        fi
        
        echo "正在备份数据库..."
        cp "$DB_FILE" "${DB_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
        echo "备份完成: ${DB_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
        echo ""
        
        echo "正在执行迁移..."
        sqlite3 "$DB_FILE" < migrate_sqlite.sql 2>&1 | grep -v "duplicate column name" || true
        echo ""
        echo "迁移完成!"
        ;;
        
    2)
        echo ""
        echo "选择: MySQL/MariaDB"
        echo "======================================"
        echo ""
        
        read -p "MySQL 主机 [localhost]: " MYSQL_HOST
        MYSQL_HOST=${MYSQL_HOST:-localhost}
        
        read -p "MySQL 端口 [3306]: " MYSQL_PORT
        MYSQL_PORT=${MYSQL_PORT:-3306}
        
        read -p "数据库名 [oneapi]: " MYSQL_DB
        MYSQL_DB=${MYSQL_DB:-oneapi}
        
        read -p "MySQL 用户名 [root]: " MYSQL_USER
        MYSQL_USER=${MYSQL_USER:-root}
        
        read -sp "MySQL 密码: " MYSQL_PASS
        echo ""
        echo ""
        
        echo "正在执行迁移..."
        mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" "$MYSQL_DB" < migrate_mysql.sql
        echo ""
        echo "迁移完成!"
        ;;
        
    3)
        echo ""
        echo "选择: PostgreSQL"
        echo "======================================"
        echo ""
        
        read -p "PostgreSQL 主机 [localhost]: " PG_HOST
        PG_HOST=${PG_HOST:-localhost}
        
        read -p "PostgreSQL 端口 [5432]: " PG_PORT
        PG_PORT=${PG_PORT:-5432}
        
        read -p "数据库名 [oneapi]: " PG_DB
        PG_DB=${PG_DB:-oneapi}
        
        read -p "PostgreSQL 用户名 [postgres]: " PG_USER
        PG_USER=${PG_USER:-postgres}
        
        read -sp "PostgreSQL 密码: " PG_PASS
        echo ""
        echo ""
        
        export PGPASSWORD="$PG_PASS"
        
        echo "正在执行迁移..."
        psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" -f migrate_postgres.sql
        echo ""
        echo "迁移完成!"
        
        unset PGPASSWORD
        ;;
        
    *)
        echo "错误：无效的选项"
        exit 1
        ;;
esac

echo ""
echo "======================================"
echo "迁移成功完成!"
echo "======================================"
echo ""
echo "新增字段："
echo "  users 表："
echo "    - rpm: 每分钟请求数限制"
echo "    - rpd: 每天请求数限制"
echo "    - rpw: 每周请求数限制"
echo "  tokens 表："
echo "    - used_quota: 已使用配额"
echo ""
echo "现在可以启动新版本的 One-API 了！"
echo ""
