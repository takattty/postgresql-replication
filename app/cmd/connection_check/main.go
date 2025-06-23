// Connection check tool for PostgreSQL replication setup
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// getEnv 環境変数を取得、存在しない場合はデフォルト値を返す
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// testConnection 指定されたホスト・ポートでの接続をテスト
func testConnection(host string, port int, description string) bool {
	fmt.Printf("🔗 %sへの接続をテスト中...\n", description)
	fmt.Printf("   ホスト: %s:%d\n", host, port)

	// 接続文字列を構築
	// デモ用のデフォルト値を使用。本番環境では環境変数を使用してください。
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "password")
	dbName := getEnv("POSTGRES_DB", "testdb")
	
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
		host, port, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("   ❌ 接続失敗: %v\n", err)
		return false
	}
	defer db.Close()

	// 接続テスト
	err = db.Ping()
	if err != nil {
		fmt.Printf("   ❌ 接続失敗: %v\n", err)
		return false
	}

	// PostgreSQLバージョン確認
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		fmt.Printf("   ❌ バージョン取得失敗: %v\n", err)
		return false
	}
	fmt.Printf("   ✅ 接続成功: %s\n", version[:60]+"...")

	// テーブル存在確認
	var count int
	err = db.QueryRow("SELECT count(*) FROM test_replication").Scan(&count)
	if err != nil {
		fmt.Printf("   ❌ テーブル確認失敗: %v\n", err)
		return false
	}
	fmt.Printf("   📊 test_replicationテーブル: %d件のデータ\n", count)

	// サーバーの種別確認
	var isRecovery bool
	err = db.QueryRow("SELECT pg_is_in_recovery()").Scan(&isRecovery)
	if err != nil {
		fmt.Printf("   ❌ サーバータイプ確認失敗: %v\n", err)
		return false
	}

	serverType := "プライマリ"
	if isRecovery {
		serverType = "スタンバイ"
	}
	fmt.Printf("   🏷️  サーバータイプ: %s\n", serverType)

	return true
}

func main() {
	fmt.Println("🎯 PostgreSQL接続テスト")
	fmt.Println(strings.Repeat("=", 50))

	// プライマリサーバーテスト
	primaryOK := testConnection("localhost", 5432, "プライマリサーバー")
	fmt.Println()

	// スタンバイサーバーテスト
	standbyOK := testConnection("localhost", 5433, "スタンバイサーバー")
	fmt.Println()

	// 結果サマリー
	fmt.Println("📋 テスト結果サマリー:")
	if primaryOK {
		fmt.Println("   プライマリ: ✅ OK")
	} else {
		fmt.Println("   プライマリ: ❌ NG")
	}

	if standbyOK {
		fmt.Println("   スタンバイ: ✅ OK")
	} else {
		fmt.Println("   スタンバイ: ❌ NG")
	}

	if primaryOK && standbyOK {
		fmt.Println("\n🎉 全ての接続テストが成功しました！")
		fmt.Println("   デモアプリケーションを実行できます。")
		os.Exit(0)
	} else {
		fmt.Println("\n⚠️  接続に問題があります。")
		fmt.Println("   Docker Composeコンテナの状態を確認してください。")
		os.Exit(1)
	}
}