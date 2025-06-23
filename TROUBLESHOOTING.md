# トラブルシューティング記録

## 環境構築時に発生した問題と解決策

### 1. Docker Compose のcommand構文エラー

**問題**: 
```
postgres-standby  | bash: -c: line 6: syntax error near unexpected token `postgres'
```

**原因**: 
- YAML の `command: >` 記法で複数行コマンドの引用符処理が不適切
- 改行とインデントの組み合わせで構文エラーが発生

**解決策**: 
```yaml
# 修正前
command: >
  bash -c "
  # コマンド
  "

# 修正後  
command: |
  bash -c "
  # コマンド
  "
```

### 2. PostgreSQL実行ユーザー権限エラー

**問題**:
```
"root" execution of the PostgreSQL server is not permitted.
The server must be started under an unprivileged user ID
```

**原因**:
- Dockerコンテナ内でrootユーザーとしてPostgreSQLを実行しようとした
- PostgreSQLはセキュリティ上、特権ユーザーでの実行を禁止

**解決策**:
```bash
# 修正前
postgres -c config_file=/etc/postgresql/postgresql.conf

# 修正後
exec gosu postgres postgres -c config_file=/etc/postgresql/postgresql.conf
```

### 3. ユーザー切り替えコマンドの違い

**問題**:
```
bash: line 6: exec: su-exec: not found
```

**原因**:
- PostgreSQL 14 Docker イメージには `su-exec` がインストールされていない
- Alpine Linux ベースのイメージでは `su-exec`、Debian ベースでは `gosu` を使用

**解決策**:
- `su-exec` → `gosu` に変更

### 4. pg_basebackup の実行ユーザー

**問題**: 
初期データ複製時の権限問題

**解決策**:
```bash
# postgres ユーザーとして pg_basebackup を実行
gosu postgres bash -c 'PGPASSWORD=repl_password pg_basebackup -h postgres-primary -D /var/lib/postgresql/data -U replicator -R -W -X stream'
```

## 成功要因

### 1. ヘルスチェックの適切な設定
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres -d testdb"]
  interval: 10s
  timeout: 5s
  retries: 5
```
- プライマリサーバーの完全起動を待ってからスタンバイを初期化

### 2. depends_on の condition 指定
```yaml
depends_on:
  postgres-primary:
    condition: service_healthy
```
- サービス間の依存関係を明確に定義

### 3. ネットワーク設定
- カスタムネットワーク `postgres-net` でコンテナ間通信を確立
- サービス名での名前解決が有効

### 4. 段階的なデバッグアプローチ
1. コンテナ起動状態確認 (`docker compose ps`)
2. ログ確認 (`docker compose logs`)
3. 問題の特定と修正
4. 再起動とテスト

## 学習ポイント

1. **Docker Compose YAML構文**: `>` vs `|` の使い分け
2. **PostgreSQL セキュリティ**: 特権ユーザー実行の禁止
3. **Docker イメージの違い**: Alpine vs Debian でのツール差異
4. **権限管理**: `gosu` を使った安全なユーザー切り替え
5. **サービス依存関係**: ヘルスチェックベースの起動順制御