# セキュリティガイド

## 概要

このプロジェクトはPostgreSQLストリーミングレプリケーションの学習用デモです。GitHubに公開する際のセキュリティ対策を実装しています。

## セキュリティ対策

### 1. 環境変数による認証情報管理

**実装済み:**
- すべてのパスワードとユーザー名を環境変数化
- `.env.example`ファイルでテンプレート提供
- `.env`ファイルを`.gitignore`で除外

**対象ファイル:**
- `docker-compose.yml`: 全コンテナの環境変数化
- `scripts/setup-primary.sh`: レプリケーションユーザー作成の環境変数化
- `app/`: 全Goアプリケーションの接続情報環境変数化

### 2. ネットワークアクセス制限

**実装済み:**
- `pg_hba.conf`で0.0.0.0/0の全開放をコメントアウト
- Dockerネットワーク内のプライベートIP範囲のみ許可
- 本番環境でのさらなる制限を推奨

### 3. デフォルト値とフォールバック

**設計方針:**
- デモ環境での動作確保のため、適切なデフォルト値を提供
- 環境変数が未設定でも基本動作可能
- セキュリティ警告を適切に表示

## 使用前の必須設定

### 1. 環境変数ファイル作成

```bash
cp .env.example .env
```

### 2. パスワード変更

`.env`ファイルを編集し、以下の値を変更してください：

```bash
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_REPLICATION_PASSWORD=your_replication_password_here
PGADMIN_DEFAULT_EMAIL=your_email@example.com
PGADMIN_DEFAULT_PASSWORD=your_admin_password_here
```

### 3. パスワード要件

- **最低8文字以上**
- **英数字と記号を組み合わせ**
- **推測されやすい単語は避ける**
- **レプリケーションパスワードとメインパスワードは異なる値に**

## 本番環境での追加対策

### 1. SSL/TLS暗号化

```postgresql
# postgresql.conf
ssl = on
ssl_cert_file = 'server.crt'
ssl_key_file = 'server.key'
```

### 2. 証明書ベース認証

```postgresql
# pg_hba.conf
hostssl replication replicator 0.0.0.0/0 cert
```

### 3. ネットワーク分離

```yaml
# docker-compose.yml での例
networks:
  postgres-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

## 監査とログ

### 1. 接続ログ

```postgresql
# postgresql.conf
log_connections = on
log_disconnections = on
log_statement = 'all'
```

### 2. 定期的なパスワード変更

```bash
# 3-6ヶ月ごとのパスワード変更を推奨
ALTER USER postgres PASSWORD 'new_secure_password';
ALTER USER replicator PASSWORD 'new_replication_password';
```

## トラブルシューティング

### 環境変数が読み込まれない場合

1. `.env`ファイルの存在確認
2. `docker-compose.yml`での変数名確認
3. コンテナ再起動: `docker compose down && docker compose up -d`

### 接続エラーの場合

1. パスワード設定の確認
2. ネットワーク設定の確認
3. `pg_hba.conf`の設定確認

## セキュリティチェックリスト

- [ ] `.env`ファイルを作成し、適切なパスワードを設定
- [ ] デフォルトパスワードを変更
- [ ] `.env`ファイルがGitにコミットされていないことを確認
- [ ] 本番環境ではSSL/TLS暗号化を有効化
- [ ] ネットワークアクセスを適切に制限
- [ ] 定期的なパスワード変更計画を策定
- [ ] 監査ログの設定と確認

## 注意事項

⚠️ **このプロジェクトは学習・デモ用途です**

- 本番環境での使用には追加のセキュリティ対策が必要
- 定期的なセキュリティ更新とパッチ適用が重要
- 機密データは絶対に使用しないでください