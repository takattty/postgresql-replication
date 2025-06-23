# 読み書き分離デモアプリケーション（Go版）

PostgreSQLストリーミングレプリケーション環境での読み書き分離を実演するGoアプリケーションです。

## 機能

### 1. 基本的な読み書き分離
- **書き込み**: プライマリサーバー（ポート5432）への書き込み操作（Docker exec経由）
- **読み取り**: スタンバイサーバー（ポート5433）からの読み取り操作（直接接続）
- **データ同期**: 書き込み後の自動レプリケーション確認

### 2. パフォーマンス測定
- 書き込み・読み取り操作の実行時間測定
- レプリケーション遅延の定量的評価
- 統計処理による平均値計算

### 3. レプリケーション監視
- `pg_stat_replication`を使用したリアルタイム監視
- LSN（Log Sequence Number）の進行状況確認
- データ整合性チェック

### 4. データ整合性テスト
- 連続データ書き込みによる同期テスト
- レプリケーション遅延の影響確認

## セットアップ

### 1. 依存関係のインストール
```bash
cd app
go mod tidy
```

### 2. PostgreSQL環境の確認
レプリケーション環境が起動していることを確認：
```bash
docker compose ps
```

### 3. アプリケーション実行

#### 接続テスト
```bash
go run connection_test.go
```

#### シンプルテスト
```bash
go run simple_test.go
```

#### フルデモ
```bash
go run replication_demo_final.go
```

## プログラム構造

### 主要な型

#### ReplicationDatabase
```go
type ReplicationDatabase struct {
    StandbyDB *sql.DB  // スタンバイサーバーへの接続
}
```

#### ReplicationData
```go
type ReplicationData struct {
    ID        int
    Data      string
    CreatedAt time.Time
}
```

#### ReplicationDemo
```go
type ReplicationDemo struct {
    DB *ReplicationDatabase
}
```

### 主要なメソッド

#### データ操作
- `WriteToPrimary(dataText string) bool`: プライマリへの書き込み
- `ReadFromStandby(limit int) ([]ReplicationData, error)`: スタンバイからの読み取り
- `GetDataCount() (int, error)`: データ件数取得

#### 監視・テスト
- `GetReplicationStatus() *float64`: レプリケーション状態取得
- `RunBasicDemo() bool`: 基本デモ実行
- `RunPerformanceTest(iterations int)`: パフォーマンステスト
- `RunDataConsistencyCheck() bool`: データ整合性チェック

## 接続設定

### スタンバイサーバー（Go直接接続）
```go
connStr := "host=localhost port=5433 user=postgres password=password dbname=testdb sslmode=disable"
```

### プライマリサーバー（Docker exec経由）
```bash
docker exec postgres-primary psql -U postgres -d testdb -c "INSERT ..."
```

## エラーハンドリング

- データベース接続エラーの適切な処理
- SQL実行エラーのキャッチ
- Docker コマンド実行エラーの対応
- リソースの確実なクリーンアップ（defer文使用）

## パフォーマンス特性

### Python版との比較
- **起動時間**: Goの方が高速（コンパイル済みバイナリ）
- **メモリ使用量**: Goの方が軽量
- **並行処理**: Goのgoroutineによる効率的な並行処理
- **型安全性**: コンパイル時の型チェック

### 測定項目
- 書き込み操作時間
- 読み取り操作時間
- レプリケーション遅延
- データ整合性確認

## 学習ポイント

1. **Go database/sql**: 標準的なデータベース操作
2. **PostgreSQL driver**: lib/pq ドライバーの使用
3. **構造体とメソッド**: オブジェクト指向的な設計
4. **エラーハンドリング**: Goのエラー処理パターン
5. **Docker統合**: 外部コマンド実行による書き込み操作
6. **時間測定**: time.Now()とtime.Since()による性能測定

## トラブルシューティング

### 依存関係エラー
```
go: cannot find module for path github.com/lib/pq
```
- `go mod tidy`でモジュールをダウンロード

### 接続エラー
```
connection refused
```
- PostgreSQLコンテナが起動しているか確認
- ポート番号が正しいか確認（スタンバイ:5433）

### Docker実行エラー
```
docker command not found
```
- Dockerがインストールされているか確認
- コンテナが実行中か確認（`docker compose ps`）

## ビルドと実行

### 開発モード
```bash
go run replication_demo_final.go
```

### 本番ビルド
```bash
go build -o replication-demo replication_demo_final.go
./replication-demo
```

### クロスコンパイル例
```bash
# Linux用
GOOS=linux GOARCH=amd64 go build -o replication-demo-linux replication_demo_final.go

# Windows用
GOOS=windows GOARCH=amd64 go build -o replication-demo.exe replication_demo_final.go
```