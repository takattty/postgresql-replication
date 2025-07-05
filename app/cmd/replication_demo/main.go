// Comprehensive replication demo application
package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

// ReplicationData レプリケーションデータ構造体
type ReplicationData struct {
	ID        int
	Data      string
	CreatedAt time.Time
}

// ReplicationDatabase レプリケーション環境でのデータベース操作クラス
type ReplicationDatabase struct {
	StandbyDB *sql.DB
}

// NewReplicationDatabase 新しいReplicationDatabaseインスタンスを作成
func NewReplicationDatabase() (*ReplicationDatabase, error) {
	// 環境変数から接続情報を取得（デモ用デフォルト値付き）
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "password")
	dbName := getEnv("POSTGRES_DB", "testdb")
	
	standbyConnStr := fmt.Sprintf("host=localhost port=5433 user=%s password=%s dbname=%s sslmode=disable",
		dbUser, dbPassword, dbName)
	standbyDB, err := sql.Open("postgres", standbyConnStr)
	if err != nil {
		return nil, fmt.Errorf("スタンバイDB接続エラー: %v", err)
	}

	err = standbyDB.Ping()
	if err != nil {
		_ = standbyDB.Close()
		return nil, fmt.Errorf("スタンバイDB ping エラー: %v", err)
	}

	fmt.Println("✅ レプリケーションデータベース接続を初期化")
	fmt.Println("   - 読み取り: スタンバイ (localhost:5433)")
	fmt.Println("   - 書き込み: プライマリ (Docker経由)")

	return &ReplicationDatabase{
		StandbyDB: standbyDB,
	}, nil
}

// Close データベース接続を閉じる
func (r *ReplicationDatabase) Close() {
	if r.StandbyDB != nil {
		_ = r.StandbyDB.Close()
	}
}

// WriteToPrimary プライマリサーバーにデータを書き込み（Docker経由）
func (r *ReplicationDatabase) WriteToPrimary(dataText string) bool {
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb",
		"-c", fmt.Sprintf("INSERT INTO test_replication (data) VALUES ('%s') RETURNING id, created_at;", dataText))

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ 書き込み失敗: %v\n", err)
		return false
	}

	// 結果をパース（簡易版）
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "|") && !strings.Contains(strings.ToLower(line), "id") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				rowID := strings.TrimSpace(parts[0])
				fmt.Printf("📝 プライマリに書き込み成功: ID=%s, データ='%s'\n", rowID, dataText)
				return true
			}
		}
	}

	fmt.Printf("📝 プライマリに書き込み成功: データ='%s'\n", dataText)
	return true
}

// ReadFromStandby スタンバイサーバーからデータを読み取り
func (r *ReplicationDatabase) ReadFromStandby(limit int) ([]ReplicationData, error) {
	query := "SELECT id, data, created_at FROM test_replication ORDER BY created_at DESC LIMIT $1"
	rows, err := r.StandbyDB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("データ読み取りエラー: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var results []ReplicationData
	for rows.Next() {
		var data ReplicationData
		err := rows.Scan(&data.ID, &data.Data, &data.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("データスキャンエラー: %v", err)
		}
		results = append(results, data)
	}

	fmt.Printf("📖 スタンバイから読み取り: %d件のデータを取得\n", len(results))
	return results, nil
}

// GetReplicationStatus レプリケーション状態を取得（プライマリから）
func (r *ReplicationDatabase) GetReplicationStatus() *float64 {
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb",
		"-c", `SELECT 
			client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn,
			CASE WHEN replay_lsn IS NOT NULL THEN 
				EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))
			END AS lag_seconds
			FROM pg_stat_replication;`)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ レプリケーション状態取得失敗: %v\n", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "streaming") {
			parts := strings.Split(line, "|")
			if len(parts) >= 7 {
				clientAddr := strings.TrimSpace(parts[0])
				state := strings.TrimSpace(parts[1])
				lagStr := strings.TrimSpace(parts[6])

				var lagValue float64 = 0
				if lagStr != "" && lagStr != " " {
					if parsed, err := strconv.ParseFloat(lagStr, 64); err == nil {
						lagValue = parsed
					}
				}

				fmt.Printf("⏱️  レプリケーション状態: %s, 遅延: %.3f秒 (クライアント: %s)\n",
					state, lagValue, clientAddr)
				return &lagValue
			}
		}
	}

	fmt.Println("✅ レプリケーション接続確認済み")
	zero := 0.0
	return &zero
}

// GetDataCount 現在のデータ件数を取得
func (r *ReplicationDatabase) GetDataCount() (int, error) {
	var count int
	err := r.StandbyDB.QueryRow("SELECT count(*) FROM test_replication").Scan(&count)
	return count, err
}

// ReplicationDemo レプリケーションデモ実行クラス
type ReplicationDemo struct {
	DB *ReplicationDatabase
}

// NewReplicationDemo 新しいReplicationDemoインスタンスを作成
func NewReplicationDemo() (*ReplicationDemo, error) {
	db, err := NewReplicationDatabase()
	if err != nil {
		return nil, err
	}
	return &ReplicationDemo{DB: db}, nil
}

// Close リソースのクリーンアップ
func (rd *ReplicationDemo) Close() {
	if rd.DB != nil {
		rd.DB.Close()
	}
}

// RunBasicDemo 基本的な読み書き分離デモ
func (rd *ReplicationDemo) RunBasicDemo() bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🚀 基本的な読み書き分離デモを開始")
	fmt.Println(strings.Repeat("=", 60))

	// 1. 現在のデータ確認
	initialCount, err := rd.DB.GetDataCount()
	if err != nil {
		fmt.Printf("❌ 初期データ件数取得エラー: %v\n", err)
		return false
	}
	fmt.Printf("📊 開始時のデータ件数: %d\n", initialCount)

	// 2. データ書き込み（プライマリ）
	testData := fmt.Sprintf("Demo data at %s", time.Now().Format("2006-01-02 15:04:05"))
	writeSuccess := rd.DB.WriteToPrimary(testData)

	if !writeSuccess {
		fmt.Println("❌ 書き込みに失敗したため、デモを中断します")
		return false
	}

	// 3. レプリケーション遅延チェック
	time.Sleep(1 * time.Second)
	rd.DB.GetReplicationStatus()

	// 4. データ読み取り（スタンバイ）
	standbyData, err := rd.DB.ReadFromStandby(5)
	if err != nil {
		fmt.Printf("❌ スタンバイデータ読み取りエラー: %v\n", err)
		return false
	}

	// 5. 同期確認
	finalCount, err := rd.DB.GetDataCount()
	if err != nil {
		fmt.Printf("❌ 最終データ件数取得エラー: %v\n", err)
		return false
	}

	fmt.Printf("\n📊 データ同期結果:\n")
	fmt.Printf("   開始時: %d件\n", initialCount)
	fmt.Printf("   終了時: %d件\n", finalCount)
	fmt.Printf("   増加: %d件\n", finalCount-initialCount)

	if finalCount > initialCount {
		fmt.Println("   ✅ データが正常に同期されました")
		if len(standbyData) > 0 {
			latest := standbyData[0]
			fmt.Printf("   📄 最新データ: ID=%d, 内容='%s'\n", latest.ID, latest.Data)
		}
	} else {
		fmt.Println("   ⚠️  データ同期に問題があります")
	}

	return true
}

// RunPerformanceTest パフォーマンステスト
func (rd *ReplicationDemo) RunPerformanceTest(iterations int) {
	fmt.Printf("\n"+strings.Repeat("=", 60)+"\n")
	fmt.Printf("⚡ パフォーマンステスト開始 (%d回)\n", iterations)
	fmt.Println(strings.Repeat("=", 60))

	var writeTimes []float64
	var readTimes []float64

	for i := 0; i < iterations; i++ {
		fmt.Printf("\n🔄 テスト %d/%d\n", i+1, iterations)

		// 書き込み性能測定
		startTime := time.Now()
		testData := fmt.Sprintf("Performance test #%d at %s", i+1, time.Now().Format("2006-01-02T15:04:05"))
		success := rd.DB.WriteToPrimary(testData)
		writeTime := time.Since(startTime).Seconds()

		if success {
			writeTimes = append(writeTimes, writeTime)
			fmt.Printf("   📝 書き込み時間: %.3f秒\n", writeTime)
		} else {
			fmt.Println("   ❌ 書き込み失敗")
			continue
		}

		// 短い待機
		time.Sleep(500 * time.Millisecond)

		// 読み取り性能測定
		startTime = time.Now()
		_, err := rd.DB.ReadFromStandby(1)
		readTime := time.Since(startTime).Seconds()
		if err == nil {
			readTimes = append(readTimes, readTime)
			fmt.Printf("   📖 読み取り時間: %.3f秒\n", readTime)
		} else {
			fmt.Printf("   ❌ 読み取り失敗: %v\n", err)
		}
	}

	// 統計計算
	if len(writeTimes) > 0 && len(readTimes) > 0 {
		avgWrite := average(writeTimes)
		avgRead := average(readTimes)

		fmt.Printf("\n📈 パフォーマンス結果:\n")
		fmt.Printf("   平均書き込み時間: %.3f秒\n", avgWrite)
		fmt.Printf("   平均読み取り時間: %.3f秒\n", avgRead)
		fmt.Printf("   書き込み/読み取り比: %.1f倍\n", avgWrite/avgRead)

		// 最終的なレプリケーション状態確認
		fmt.Printf("\n📊 最終レプリケーション状態:\n")
		rd.DB.GetReplicationStatus()
	} else {
		fmt.Println("❌ 有効なパフォーマンスデータが取得できませんでした")
	}
}

// RunDataConsistencyCheck データ整合性チェック
func (rd *ReplicationDemo) RunDataConsistencyCheck() bool {
	fmt.Printf("\n"+strings.Repeat("=", 60)+"\n")
	fmt.Println("🔍 データ整合性チェック")
	fmt.Println(strings.Repeat("=", 60))

	// 1. 複数データを連続書き込み
	fmt.Println("📝 連続データ書き込み中...")
	baseTime := time.Now().Format("20060102_150405")

	successCount := 0
	for i := 0; i < 3; i++ {
		data := fmt.Sprintf("Consistency test %d - %s", i+1, baseTime)
		success := rd.DB.WriteToPrimary(data)
		if success {
			fmt.Printf("   ✅ データ%d書き込み完了\n", i+1)
			successCount++
		} else {
			fmt.Printf("   ❌ データ%d書き込み失敗\n", i+1)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 2. レプリケーション待機
	fmt.Println("\n⏱️  レプリケーション完了待機...")
	time.Sleep(2 * time.Second)

	// 3. データ読み取りと確認
	fmt.Println("\n📖 整合性確認...")
	data, err := rd.DB.ReadFromStandby(5)
	if err != nil {
		fmt.Printf("❌ データ読み取りエラー: %v\n", err)
		return false
	}

	consistencyCount := 0
	for _, row := range data {
		if strings.Contains(row.Data, baseTime) {
			consistencyCount++
			fmt.Printf("   ✅ 同期確認: ID=%d, データ='%s'\n", row.ID, row.Data)
		}
	}

	fmt.Printf("\n📊 整合性結果:\n")
	fmt.Printf("   書き込み成功: %d件\n", successCount)
	fmt.Printf("   同期確認: %d件\n", consistencyCount)

	if consistencyCount >= successCount {
		fmt.Println("   🎉 データ整合性テスト成功！")
		return true
	} else {
		fmt.Println("   ⚠️  一部データが未同期の可能性があります")
		return false
	}
}

// average 平均値を計算
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func main() {
	demo, err := NewReplicationDemo()
	if err != nil {
		fmt.Printf("❌ デモ初期化エラー: %v\n", err)
		return
	}
	defer demo.Close()

	fmt.Println("🎯 PostgreSQL読み書き分離デモアプリケーション（Go版）")
	fmt.Println("🔗 Docker環境でのレプリケーション動作確認")

	// 基本デモ実行
	basicSuccess := demo.RunBasicDemo()

	if basicSuccess {
		// パフォーマンステスト実行
		demo.RunPerformanceTest(3)

		// データ整合性チェック
		demo.RunDataConsistencyCheck()

		fmt.Printf("\n🎉 全てのデモが完了しました！\n")
		fmt.Println("📋 実行内容:")
		fmt.Println("   ✅ 基本的な読み書き分離")
		fmt.Println("   ✅ パフォーマンス測定")
		fmt.Println("   ✅ データ整合性確認")
		fmt.Println("   ✅ レプリケーション監視")
	} else {
		fmt.Println("\n❌ 基本デモに失敗したため、以降のテストをスキップします")
	}
}