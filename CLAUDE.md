# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# PostgreSQL Streaming Replication Learning Project

## Architecture Overview

This project demonstrates PostgreSQL streaming replication using Docker Compose with three containers:
- **postgres-primary**: Master database server with WAL streaming enabled
- **postgres-standby**: Read-only replica using pg_basebackup initialization
- **pgadmin**: Web-based database administration tool

The standby server automatically initializes from the primary using `pg_basebackup` with streaming replication (`-R -X stream`). Configuration files in `primary/` and `standby/` directories contain PostgreSQL and authentication settings optimized for replication.

Key files:
- `docker-compose.yml`: Container orchestration with health checks and dependencies
- `scripts/setup-primary.sh`: Creates replication user, slot, and test table
- `scripts/setup-standby.sh`: Minimal setup (initialization handled by pg_basebackup)
- `primary/postgresql.conf`: WAL settings, replication slots, archiving configuration
- `standby/postgresql.conf`: Hot standby settings, primary connection info
- `primary/pg_hba.conf` & `standby/pg_hba.conf`: Authentication for replication connections

## 概要
Docker ComposeでPostgreSQLストリーミングレプリケーション環境を構築し、実際の動作を確認する学習プロジェクトです。

## 環境構成
- **Primary Server**: postgres:14 (ポート5432)
- **Standby Server**: postgres:14 (ポート5433)
- **PgAdmin**: 管理ツール (ポート8080)

## セットアップ手順

### 1. 環境変数設定（重要）
```bash
# .envファイルを作成してデフォルトパスワードを変更
cp .env.example .env
# .envファイルを編集して適切なパスワードを設定
```

### 2. 環境起動
```bash
cd ~/software/postgresql-replication
docker compose up -d
```

### 3. 環境確認
```bash
# コンテナ状態確認
docker compose ps

# ログ確認
docker compose logs postgres-primary
docker compose logs postgres-standby
```

### 4. レプリケーション状態確認
```bash
# プライマリに接続してレプリケーション状態確認
docker exec -it postgres-primary psql -U postgres -d testdb -c "SELECT * FROM pg_stat_replication;"

# スタンバイに接続してレプリケーション状態確認
docker exec -it postgres-standby psql -U postgres -d testdb -c "SELECT * FROM pg_stat_wal_receiver;"
```

### 5. データ同期テスト
```bash
# プライマリでデータ挿入
docker exec -it postgres-primary psql -U postgres -d testdb -c "INSERT INTO test_replication (data) VALUES ('Test from primary at $(date)');"

# スタンバイでデータ確認
docker exec -it postgres-standby psql -U postgres -d testdb -c "SELECT * FROM test_replication ORDER BY created_at DESC LIMIT 5;"
```

## 接続情報

### プライマリサーバー
- **Host**: localhost
- **Port**: 5432
- **Database**: testdb (環境変数: POSTGRES_DB)
- **User**: postgres (環境変数: POSTGRES_USER)
- **Password**: .envファイルで設定 (環境変数: POSTGRES_PASSWORD)

### スタンバイサーバー
- **Host**: localhost
- **Port**: 5433
- **Database**: testdb (環境変数: POSTGRES_DB)
- **User**: postgres (環境変数: POSTGRES_USER)
- **Password**: .envファイルで設定 (環境変数: POSTGRES_PASSWORD)

### PgAdmin
- **URL**: http://localhost:8080
- **Email**: .envファイルで設定 (環境変数: PGADMIN_DEFAULT_EMAIL)
- **Password**: .envファイルで設定 (環境変数: PGADMIN_DEFAULT_PASSWORD)

### セキュリティ注意事項
⚠️ **本プロジェクトはデモ・学習用です**
- デフォルトパスワードは必ず変更してください
- 本番環境では強力なパスワードを使用し、SSL接続を有効にしてください
- ネットワークアクセスを適切に制限してください

## 監視・テストコマンド

### レプリケーション状態監視
```bash
# レプリケーション遅延確認
docker exec -it postgres-primary psql -U postgres -d testdb -c "
SELECT 
    client_addr,
    state,
    sent_lsn,
    write_lsn,
    flush_lsn,
    replay_lsn,
    CASE WHEN replay_lsn IS NOT NULL THEN 
        EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))
    END AS lag_seconds
FROM pg_stat_replication;"
```

### WAL統計確認
```bash
# WAL生成量確認
docker exec -it postgres-primary psql -U postgres -d testdb -c "
SELECT 
    pg_current_wal_lsn() as current_wal_lsn,
    pg_walfile_name(pg_current_wal_lsn()) as current_wal_file;"
```

### スタンバイ情報確認
```bash
# リカバリ状態確認
docker exec -it postgres-standby psql -U postgres -d testdb -c "
SELECT 
    pg_is_in_recovery() as in_recovery,
    pg_last_wal_receive_lsn() as received_lsn,
    pg_last_wal_replay_lsn() as replayed_lsn,
    pg_last_xact_replay_timestamp() as last_replay_time;"
```

## セキュリティ設定

### 環境変数による設定
本プロジェクトでは機密情報の保護のため、環境変数による設定を採用しています：

```bash
# .env ファイルの例
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_REPLICATION_PASSWORD=your_replication_password_here
PGADMIN_DEFAULT_EMAIL=your_email@example.com
PGADMIN_DEFAULT_PASSWORD=your_admin_password_here
```

### パスワードセキュリティ
- **デフォルトパスワードは使用しない**: 必ず独自のパスワードに変更
- **強力なパスワード**: 8文字以上、英数字記号を組み合わせ
- **パスワード管理**: .envファイルはgitignoreに含まれており、リポジトリにコミットされません

### ネットワークセキュリティ
- pg_hba.confでアクセス元IPを制限
- 本番環境では0.0.0.0/0のような全開放は避ける
- SSL/TLS暗号化を有効にすることを推奨

## 環境停止
```bash
# 環境停止
docker compose down

# データも削除する場合
docker compose down -v
```

## トラブルシューティング

### よくある問題
1. **スタンバイが起動しない**: プライマリの起動完了を確認
2. **レプリケーションが進まない**: ネットワーク設定とユーザー権限を確認
3. **接続エラー**: pg_hba.confの設定を確認
4. **Docker Compose構文エラー**: `command: |` vs `command: >` の使い分け
5. **PostgreSQL権限エラー**: `gosu postgres` でユーザー切り替えが必要

### 詳細なトラブルシューティング
構築時に発生した問題と解決策の詳細は `TROUBLESHOOTING.md` を参照してください。

### ログ確認方法
```bash
# 詳細ログ確認
docker compose logs -f postgres-primary
docker compose logs -f postgres-standby
```

## Go読み書き分離デモアプリケーション

### アプリケーション実行
```bash
# 依存関係インストール
cd app && go mod tidy

# 標準Goテスト実行（推奨）
go test -v

# 個別コマンド実行
go run cmd/connection_check/main.go
go run cmd/simple_demo/main.go
go run cmd/replication_demo/main.go

# バイナリビルド
go build -o bin/connection_check ./cmd/connection_check
go build -o bin/simple_demo ./cmd/simple_demo
go build -o bin/replication_demo ./cmd/replication_demo

# ビルド済みバイナリ実行
./bin/connection_check
./bin/simple_demo
./bin/replication_demo
```

### Go版機能
- **読み書き分離**: プライマリ書き込み、スタンバイ読み取り
- **パフォーマンス測定**: 実行時間の定量的評価  
- **レプリケーション監視**: 遅延・状態のリアルタイム確認
- **データ整合性確認**: 連続書き込みでの同期検証
- **標準Goテスト**: `go test`による自動テスト実行
- **コマンド分離**: 独立したバイナリでの機能提供

### テスト実行結果例
```
=== RUN   TestDatabaseConnection
    ✅ データベース接続テスト成功
=== RUN   TestBasicReplication  
    ✅ 基本レプリケーションテスト成功: 初期=29, 最終=30
=== RUN   TestReplicationLag
    ✅ レプリケーション遅延テスト成功: 0.000秒
=== RUN   TestReadWriteSeparation
    ✅ 読み書き分離テスト成功: 書き込み=0.081秒, 読み取り=0.006秒
=== RUN   TestDataConsistency
    ✅ データ整合性テスト成功: 書き込み成功=3, 同期確認=3
=== RUN   TestPerformanceBenchmark
    ✅ パフォーマンステスト成功: 平均書き込み=0.114秒, 平均読み取り=0.005秒
PASS
```

### アプリケーション構成
```
app/
├── cmd/                          # コマンドライン実行ファイル
│   ├── connection_check/main.go  # データベース接続確認ツール
│   ├── simple_demo/main.go       # シンプルな読み書き分離デモ
│   └── replication_demo/main.go  # 包括的なレプリケーションデモ
├── bin/                          # ビルド済みバイナリ
├── replication_test.go           # 標準Goテストスイート
├── go.mod                        # Go modules設定
└── go.sum                        # 依存関係チェックサム
```

## 学習ポイント
- WAL（Write-Ahead Logging）の仕組み
- ストリーミングレプリケーションの動作原理
- 同期・非同期レプリケーションの違い
- レプリケーション監視方法
- フェイルオーバーの基本概念
- 読み書き分離アーキテクチャの実装
- Docker環境での制約と対応策