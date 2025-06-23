#!/bin/bash
set -e

# プライマリサーバーの初期設定スクリプト

echo "Setting up PostgreSQL Primary Server for Streaming Replication"

# レプリケーション用ユーザーを作成
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- レプリケーション用ユーザー作成
    CREATE USER ${POSTGRES_REPLICATION_USER:-replicator} WITH REPLICATION ENCRYPTED PASSWORD '${POSTGRES_REPLICATION_PASSWORD:-repl_password}';
    
    -- レプリケーションスロット作成
    SELECT pg_create_physical_replication_slot('standby_slot');
    
    -- 権限設定
    GRANT CONNECT ON DATABASE $POSTGRES_DB TO replicator;
    
    -- テスト用テーブル作成
    CREATE TABLE IF NOT EXISTS test_replication (
        id SERIAL PRIMARY KEY,
        data TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    -- サンプルデータ挿入
    INSERT INTO test_replication (data) VALUES 
        ('Primary server initial data'),
        ('Replication test data 1'),
        ('Replication test data 2');
    
    -- 現在の状態を表示
    SELECT 'Primary server setup completed' as status;
EOSQL

echo "Primary server setup completed successfully!"
echo "Replication user: replicator"
echo "Replication slot: standby_slot"