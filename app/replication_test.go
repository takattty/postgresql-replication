package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
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

// TestReplicationData レプリケーションデータ構造体
type TestReplicationData struct {
	ID        int
	Data      string
	CreatedAt time.Time
}

// TestDatabaseManager テスト用データベースマネージャー
type TestDatabaseManager struct {
	StandbyDB *sql.DB
}

// NewTestDatabaseManager テスト用データベースマネージャーを作成
func NewTestDatabaseManager() (*TestDatabaseManager, error) {
	// 環境変数から接続情報を取得（デモ用デフォルト値付き）
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "password")
	dbName := getEnv("POSTGRES_DB", "testdb")
	
	standbyConnStr := fmt.Sprintf("host=localhost port=5433 user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
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

	return &TestDatabaseManager{StandbyDB: standbyDB}, nil
}

// Close データベース接続を閉じる
func (tm *TestDatabaseManager) Close() {
	if tm.StandbyDB != nil {
		_ = tm.StandbyDB.Close()
	}
}

// WriteToPrimary プライマリサーバーにデータを書き込み
func (tm *TestDatabaseManager) WriteToPrimary(dataText string) (bool, int) {
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb",
		"-c", fmt.Sprintf("INSERT INTO test_replication (data) VALUES ('%s') RETURNING id;", dataText))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if id, err := strconv.Atoi(line); err == nil && id > 0 {
			return true, id
		}
	}
	return true, 0
}

// ReadFromStandby スタンバイサーバーからデータを読み取り
func (tm *TestDatabaseManager) ReadFromStandby(limit int) ([]TestReplicationData, error) {
	query := "SELECT id, data, created_at FROM test_replication ORDER BY created_at DESC LIMIT $1"
	rows, err := tm.StandbyDB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("データ読み取りエラー: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var results []TestReplicationData
	for rows.Next() {
		var data TestReplicationData
		err := rows.Scan(&data.ID, &data.Data, &data.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("データスキャンエラー: %v", err)
		}
		results = append(results, data)
	}

	return results, nil
}

// GetDataCount データ件数を取得
func (tm *TestDatabaseManager) GetDataCount() (int, error) {
	var count int
	err := tm.StandbyDB.QueryRow("SELECT count(*) FROM test_replication").Scan(&count)
	return count, err
}

// GetReplicationStatus レプリケーション遅延を取得
func (tm *TestDatabaseManager) GetReplicationStatus() (float64, error) {
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb", "-t",
		"-c", `SELECT COALESCE(EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())), 0) FROM pg_stat_replication WHERE state = 'streaming' LIMIT 1;`)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return -1, fmt.Errorf("レプリケーション状態取得エラー: %v", err)
	}

	lagStr := strings.TrimSpace(string(output))
	if lagStr == "" {
		return 0, nil
	}

	lag, err := strconv.ParseFloat(lagStr, 64)
	if err != nil {
		return 0, nil
	}

	return lag, nil
}

// TestConnection データベース接続をテスト
func (tm *TestDatabaseManager) TestConnection() bool {
	// スタンバイ接続テスト
	var standbyVersion string
	err := tm.StandbyDB.QueryRow("SELECT version()").Scan(&standbyVersion)
	if err != nil {
		return false
	}

	// プライマリ接続テスト
	cmd := exec.Command("docker", "exec", "postgres-primary",
		"psql", "-U", "postgres", "-d", "testdb", "-t", "-c", "SELECT version();")
	_, err = cmd.CombinedOutput()
	return err == nil
}

// TestDatabaseConnection データベース接続テスト
func TestDatabaseConnection(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	if !tm.TestConnection() {
		t.Fatal("データベース接続テストに失敗")
	}

	t.Log("✅ データベース接続テスト成功")
}

// TestBasicReplication 基本的なレプリケーション機能テスト
func TestBasicReplication(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	// 初期データ件数確認
	initialCount, err := tm.GetDataCount()
	if err != nil {
		t.Fatalf("初期データ件数取得エラー: %v", err)
	}

	// データ書き込み
	testData := fmt.Sprintf("Test data at %s", time.Now().Format("2006-01-02 15:04:05"))
	writeSuccess, _ := tm.WriteToPrimary(testData)
	if !writeSuccess {
		t.Fatal("データ書き込みに失敗")
	}

	// レプリケーション待機
	time.Sleep(2 * time.Second)

	// 最終データ件数確認
	finalCount, err := tm.GetDataCount()
	if err != nil {
		t.Fatalf("最終データ件数取得エラー: %v", err)
	}

	if finalCount <= initialCount {
		t.Fatalf("データ同期に失敗: 初期=%d, 最終=%d", initialCount, finalCount)
	}

	t.Logf("✅ 基本レプリケーションテスト成功: 初期=%d, 最終=%d", initialCount, finalCount)
}

// TestReplicationLag レプリケーション遅延テスト
func TestReplicationLag(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	lag, err := tm.GetReplicationStatus()
	if err != nil {
		t.Fatalf("レプリケーション遅延取得エラー: %v", err)
	}

	// 遅延が10秒以上は異常
	if lag > 10.0 {
		t.Fatalf("レプリケーション遅延が異常: %.3f秒", lag)
	}

	t.Logf("✅ レプリケーション遅延テスト成功: %.3f秒", lag)
}

// TestReadWriteSeparation 読み書き分離テスト
func TestReadWriteSeparation(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	// 書き込み性能測定
	writeStart := time.Now()
	testData := fmt.Sprintf("Performance test at %s", time.Now().Format("2006-01-02T15:04:05"))
	writeSuccess, _ := tm.WriteToPrimary(testData)
	writeTime := time.Since(writeStart)

	if !writeSuccess {
		t.Fatal("書き込み処理に失敗")
	}

	// 読み取り性能測定
	time.Sleep(500 * time.Millisecond)
	readStart := time.Now()
	_, err = tm.ReadFromStandby(1)
	readTime := time.Since(readStart)

	if err != nil {
		t.Fatalf("読み取り処理に失敗: %v", err)
	}

	t.Logf("✅ 読み書き分離テスト成功: 書き込み=%.3f秒, 読み取り=%.3f秒", 
		writeTime.Seconds(), readTime.Seconds())
}

// TestDataConsistency データ整合性テスト
func TestDataConsistency(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	baseTime := time.Now().Format("20060102_150405")
	successCount := 0

	// 3件の連続書き込み
	for i := 0; i < 3; i++ {
		data := fmt.Sprintf("Consistency test %d - %s", i+1, baseTime)
		success, _ := tm.WriteToPrimary(data)
		if success {
			successCount++
		}
		time.Sleep(300 * time.Millisecond)
	}

	// レプリケーション待機
	time.Sleep(2 * time.Second)

	// 整合性確認
	data, err := tm.ReadFromStandby(10)
	if err != nil {
		t.Fatalf("データ読み取りエラー: %v", err)
	}

	consistencyCount := 0
	for _, row := range data {
		if strings.Contains(row.Data, baseTime) {
			consistencyCount++
		}
	}

	if consistencyCount < successCount {
		t.Fatalf("データ整合性に問題: 書き込み成功=%d, 同期確認=%d", successCount, consistencyCount)
	}

	t.Logf("✅ データ整合性テスト成功: 書き込み成功=%d, 同期確認=%d", successCount, consistencyCount)
}

// TestPerformanceBenchmark パフォーマンスベンチマークテスト
func TestPerformanceBenchmark(t *testing.T) {
	tm, err := NewTestDatabaseManager()
	if err != nil {
		t.Fatalf("データベースマネージャー作成エラー: %v", err)
	}
	defer tm.Close()

	iterations := 5
	var writeTimes, readTimes []float64

	for i := 0; i < iterations; i++ {
		// 書き込み時間測定
		writeStart := time.Now()
		testData := fmt.Sprintf("Benchmark test #%d", i+1)
		success, _ := tm.WriteToPrimary(testData)
		writeTime := time.Since(writeStart).Seconds()

		if success {
			writeTimes = append(writeTimes, writeTime)
		}

		time.Sleep(300 * time.Millisecond)

		// 読み取り時間測定
		readStart := time.Now()
		_, err := tm.ReadFromStandby(1)
		readTime := time.Since(readStart).Seconds()

		if err == nil {
			readTimes = append(readTimes, readTime)
		}
	}

	// 平均計算
	avgWrite := calculateAverage(writeTimes)
	avgRead := calculateAverage(readTimes)

	t.Logf("✅ パフォーマンステスト成功: 平均書き込み=%.3f秒, 平均読み取り=%.3f秒", avgWrite, avgRead)
}

// calculateAverage 平均値を計算
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}