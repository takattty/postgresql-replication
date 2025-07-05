// Simple read-write separation demo
package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// getEnv 環境変数を取得、存在しない場合はデフォルト値を返す
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// simpleDemo シンプルな読み書き分離テスト
func simpleDemo() {
	fmt.Println("🎯 シンプル読み書き分離テスト")

	// スタンバイ接続（読み取り専用）
	fmt.Println("\n📖 スタンバイサーバーから読み取り...")
	// 環境変数から接続情報を取得（デモ用デフォルト値付き）
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "password")
	dbName := getEnv("POSTGRES_DB", "testdb")

	// Docker環境では異なるホスト名とポートを使用
	standbyHost := getEnv("POSTGRES_STANDBY_HOST", "localhost")
	standbyPort := getEnv("POSTGRES_STANDBY_PORT", "5433")
	
	// IPv4を強制するためにlocalhostを2127.0.0.1に変換
	if standbyHost == "localhost" {
		standbyHost = "127.0.0.1"
	}
	
	standbyConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		standbyHost, standbyPort, dbUser, dbPassword, dbName)
	standbyDB, err := sql.Open("postgres", standbyConnStr)
	if err != nil {
		fmt.Printf("❌ スタンバイ接続エラー: %v\n", err)
		return
	}
	defer func() { _ = standbyDB.Close() }()

	// 読み取り前のデータ件数確認
	var countBefore int
	err = standbyDB.QueryRow("SELECT count(*) FROM test_replication").Scan(&countBefore)
	if err != nil {
		fmt.Printf("❌ データ件数取得エラー: %v\n", err)
		return
	}
	fmt.Printf("   読み取り前のデータ件数: %d\n", countBefore)

	// 最新データ表示
	rows, err := standbyDB.Query("SELECT id, data, created_at FROM test_replication ORDER BY created_at DESC LIMIT 3")
	if err != nil {
		fmt.Printf("❌ データ取得エラー: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()

	fmt.Println("   最新データ:")
	for rows.Next() {
		var id int
		var data string
		var createdAt time.Time
		err := rows.Scan(&id, &data, &createdAt)
		if err != nil {
			fmt.Printf("❌ データスキャンエラー: %v\n", err)
			continue
		}
		fmt.Printf("     ID:%d | %s | %s\n", id, data, createdAt.Format("2006-01-02 15:04:05"))
	}

	// プライマリ接続テスト（Dockerコンテナ経由）
	fmt.Println("\n📝 プライマリサーバーに書き込み...")
	fmt.Println("   注意: ローカルホスト接続に問題があるため、dockerコマンドを使用")

	// Dockerコマンドで直接書き込み
	testData := fmt.Sprintf("Simple test at %s", time.Now().Format("2006-01-02T15:04:05"))
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb",
		"-c", fmt.Sprintf("INSERT INTO test_replication (data) VALUES ('%s');", testData))

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("   ❌ 書き込み失敗: %v\n", err)
		fmt.Printf("   出力: %s\n", string(output))
		return
	}

	if strings.Contains(string(output), "INSERT 0 1") {
		fmt.Printf("   ✅ 書き込み成功: '%s'\n", testData)
	} else {
		fmt.Printf("   ❌ 書き込み結果が不明: %s\n", string(output))
		return
	}

	// レプリケーション待機
	fmt.Println("\n⏱️  レプリケーション待機中...")
	time.Sleep(2 * time.Second)

	// スタンバイで再確認
	fmt.Println("\n📖 スタンバイサーバーで同期確認...")
	var countAfter int
	err = standbyDB.QueryRow("SELECT count(*) FROM test_replication").Scan(&countAfter)
	if err != nil {
		fmt.Printf("❌ データ件数取得エラー: %v\n", err)
		return
	}
	fmt.Printf("   読み取り後のデータ件数: %d\n", countAfter)

	if countAfter > countBefore {
		fmt.Println("   ✅ データが正常に同期されました！")

		// 最新データ確認
		var latestID int
		var latestData string
		var latestTime time.Time
		err = standbyDB.QueryRow(
			"SELECT id, data, created_at FROM test_replication ORDER BY created_at DESC LIMIT 1").Scan(
			&latestID, &latestData, &latestTime)
		if err != nil {
			fmt.Printf("❌ 最新データ取得エラー: %v\n", err)
		} else {
			fmt.Printf("   最新データ: ID:%d | %s | %s\n",
				latestID, latestData, latestTime.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("   ⚠️  データ同期に問題があります")
	}

	fmt.Printf("\n🎉 読み書き分離テスト完了!\n")
	fmt.Printf("   書き込み前: %d件\n", countBefore)
	fmt.Printf("   書き込み後: %d件\n", countAfter)
	fmt.Printf("   増加分: %d件\n", countAfter-countBefore)
}

func main() {
	simpleDemo()
}
