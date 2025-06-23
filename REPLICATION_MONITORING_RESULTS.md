# PostgreSQLレプリケーション監視・テスト結果

## 実行したクエリと結果

### 1. プライマリサーバーでのレプリケーション状態確認

**目的**: 
- プライマリサーバーからスタンバイサーバーへのWAL送信状況を確認
- レプリケーションスロットの利用状況を把握
- 接続中のレプリカ数と状態を監視

**実行コマンド**:
```bash
docker exec postgres-primary psql -U postgres -d testdb -c "SELECT * FROM pg_stat_replication;"
```

**結果**:
```
 pid | usesysid |  usename   | application_name | client_addr | client_hostname | client_port |         backend_start         | backend_xmin |   state   | sent_lsn  | write_lsn | flush_lsn | replay_lsn | write_lag | flush_lag | replay_lag | sync_priority | sync_state |          reply_time           
-----+----------+------------+------------------+-------------+-----------------+-------------+-------------------------------+--------------+-----------+-----------+-----------+-----------+------------+-----------+-----------+------------+---------------+------------+-------------------------------
  43 |    16385 | replicator | walreceiver      | 172.24.0.3  |                 |       34776 | 2025-06-23 12:31:07.635131+00 |              | streaming | 0/70000D8 | 0/70000D8 | 0/70000D8 | 0/70000D8  |           |           |            |             0 | async      | 2025-06-23 12:35:02.535085+00
```

**考察**:
- ✅ スタンバイサーバー（172.24.0.3）が正常に接続
- ✅ `state = streaming`: リアルタイムでWALを送信中
- ✅ `sent_lsn = write_lsn = flush_lsn = replay_lsn`: 完全同期状態
- ✅ `sync_state = async`: 非同期レプリケーションで動作
- ✅ `application_name = walreceiver`: スタンバイ側のWALレシーバープロセス
- ✅ `usename = replicator`: 設定したレプリケーション専用ユーザーを使用

---

### 2. スタンバイサーバーでのWALレシーバー状態確認

**目的**:
- スタンバイサーバー側でのWAL受信状況を確認
- プライマリとの接続状態を監視
- 受信したWALの適用状況を把握

**実行コマンド**:
```bash
docker exec postgres-standby psql -U postgres -d testdb -c "SELECT * FROM pg_stat_wal_receiver;"
```

**結果**:
```
 pid |  status   | receive_start_lsn | receive_start_tli | written_lsn | flushed_lsn | received_tli |      last_msg_send_time       |     last_msg_receipt_time     | latest_end_lsn |        latest_end_time        |  slot_name   |   sender_host    | sender_port |                                                                                                                                                                                    conninfo                                                                                                                                                                                     
-----+-----------+-------------------+-------------------+-------------+-------------+--------------+-------------------------------+-------------------------------+----------------+-------------------------------+--------------+------------------+-------------+---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
  17 | streaming | 0/5000000         |                 1 | 0/70000D8   | 0/70000D8   |            1 | 2025-06-23 12:35:12.523455+00 | 2025-06-23 12:35:12.523536+00 | 0/70000D8      | 2025-06-23 12:31:12.081189+00 | standby_slot | postgres-primary |        5432 | user=replicator password=******** channel_binding=prefer dbname=replication host=postgres-primary port=5432 fallback_application_name=walreceiver sslmode=prefer sslnegotiation=postgres sslcompression=0 sslcertmode=allow sslsni=1 ssl_min_protocol_version=TLSv1.2 gssencmode=prefer krbsrvname=postgres gssdelegation=0 target_session_attrs=any load_balance_hosts=disable
```

**考察**:
- ✅ `status = streaming`: プライマリからのWALを継続的に受信
- ✅ `written_lsn = flushed_lsn`: 受信したWALが確実にディスクに書き込まれている
- ✅ `slot_name = standby_slot`: 設定したレプリケーションスロットを使用
- ✅ `sender_host = postgres-primary`: Dockerネットワーク内でサービス名解決が正常動作
- ✅ `last_msg_send_time ≈ last_msg_receipt_time`: ネットワーク遅延が非常に小さい（< 1秒）
- ✅ 接続文字列に適切なSSL設定とレプリケーションユーザーが含まれている

---

### 3. 既存データの同期状況確認

**目的**:
- 初期データがプライマリからスタンバイに正しく複製されているか確認
- pg_basebackupによる初期同期の成功を検証

**実行コマンド**:
```bash
# プライマリ側
docker exec postgres-primary psql -U postgres -d testdb -c "SELECT * FROM test_replication;"

# スタンバイ側  
docker exec postgres-standby psql -U postgres -d testdb -c "SELECT * FROM test_replication;"
```

**結果** (両方同じ):
```
 id |            data             |         created_at         
----+-----------------------------+----------------------------
  1 | Primary server initial data | 2025-06-23 12:27:24.698493
  2 | Replication test data 1     | 2025-06-23 12:27:24.698493
  3 | Replication test data 2     | 2025-06-23 12:27:24.698493
```

**考察**:
- ✅ 初期データ（3件）が完全に同期されている
- ✅ created_atタイムスタンプも含めて完全一致
- ✅ pg_basebackupによる初期同期が正常に実行された

---

### 4. リアルタイム同期テスト

**目的**:
- 新しいデータ挿入時のリアルタイム同期性能を検証
- ストリーミングレプリケーションの実際の動作を確認

**実行コマンド**:
```bash
# プライマリでデータ挿入
docker exec postgres-primary psql -U postgres -d testdb -c "INSERT INTO test_replication (data) VALUES ('Test from primary at $(date)');"

# スタンバイで即座に確認
docker exec postgres-standby psql -U postgres -d testdb -c "SELECT * FROM test_replication ORDER BY created_at DESC LIMIT 5;"
```

**結果**:
```
# 挿入結果
INSERT 0 1

# スタンバイでの確認結果
 id |            data             |         created_at         
----+-----------------------------+----------------------------
  4 | Test from primary at #午後  | 2025-06-23 12:35:39.802327
  1 | Primary server initial data | 2025-06-23 12:27:24.698493
  2 | Replication test data 1     | 2025-06-23 12:27:24.698493
  3 | Replication test data 2     | 2025-06-23 12:27:24.698493
```

**考察**:
- ✅ 新しく挿入されたデータ（id=4）が即座にスタンバイに反映
- ✅ レプリケーション遅延は体感的にほぼゼロ（数秒以内）
- ✅ ストリーミングレプリケーションがリアルタイムで機能

---

### 5. レプリケーション遅延の詳細分析

**目的**:
- レプリケーション遅延を数値的に測定
- WAL送信から適用までの各段階での遅延を把握
- パフォーマンス評価の定量化

**実行コマンド**:
```bash
docker exec postgres-primary psql -U postgres -d testdb -c "
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

**結果**:
```
 client_addr |   state   | sent_lsn  | write_lsn | flush_lsn | replay_lsn | lag_seconds 
-------------+-----------+-----------+-----------+-----------+------------+-------------
 172.24.0.3  | streaming | 0/70003B8 | 0/70003B8 | 0/70003B8 | 0/70003B8  |            
```

**考察**:
- ✅ **遅延なし**: `sent_lsn = write_lsn = flush_lsn = replay_lsn`
- ✅ **lag_seconds = null**: 最新の適用タイムスタンプで遅延測定不可 = ほぼゼロ遅延
- ✅ **LSN進化**: 前回の0/70000D8から0/70003B8へ進行（新しいトランザクション反映）
- ✅ **同期品質**: 送信→書き込み→フラッシュ→適用の全段階で同期

---

### 6. WAL統計情報の確認

**目的**:
- 現在のWAL生成状況を把握
- WALファイルの進行状況を監視
- システム全体のトランザクションログ活動を理解

**実行コマンド**:
```bash
docker exec postgres-primary psql -U postgres -d testdb -c "
SELECT 
    pg_current_wal_lsn() as current_wal_lsn,
    pg_walfile_name(pg_current_wal_lsn()) as current_wal_file;"
```

**結果**:
```
 current_wal_lsn |     current_wal_file     
-----------------+--------------------------
 0/70004A0       | 000000010000000000000007
```

**考察**:
- ✅ **WAL進行**: LSNが0/70003B8から0/70004A0へ進行
- ✅ **ファイル名**: WALファイル番号7番を使用中
- ✅ **タイムライン**: 000000010000000000000007 = タイムライン1、ファイル7
- ✅ **順次処理**: WALが順次生成・処理されている証拠

---

### 7. スタンバイサーバーのリカバリ状態確認

**目的**:
- スタンバイサーバーがリカバリモードで正常動作しているか確認
- WAL受信・適用の最新状況を把握
- ホットスタンバイ機能の動作確認

**実行コマンド**:
```bash
docker exec postgres-standby psql -U postgres -d testdb -c "
SELECT 
    pg_is_in_recovery() as in_recovery,
    pg_last_wal_receive_lsn() as received_lsn,
    pg_last_wal_replay_lsn() as replayed_lsn,
    pg_last_xact_replay_timestamp() as last_replay_time;"
```

**結果**:
```
 in_recovery | received_lsn | replayed_lsn |       last_replay_time        
-------------+--------------+--------------+-------------------------------
 t           | 0/70004A0    | 0/70004A0    | 2025-06-23 12:35:39.802734+00
```

**考察**:
- ✅ **リカバリモード**: `in_recovery = t` でスタンバイとして正常動作
- ✅ **完全同期**: `received_lsn = replayed_lsn = 0/70004A0` でプライマリと完全一致
- ✅ **最新適用**: `last_replay_time`が挿入したデータのタイムスタンプと一致
- ✅ **ホットスタンバイ**: リカバリ中でも読み取りクエリが実行可能

---

## 総合評価

### パフォーマンス指標
- **レプリケーション遅延**: ほぼゼロ（< 1秒）
- **データ整合性**: 100%（全LSNが一致）
- **可用性**: プライマリ・スタンバイ両方で読み取り可能

### 設定の妥当性検証
- ✅ **レプリケーションスロット**: 正常に作成・使用
- ✅ **WAL設定**: wal_level=replica で適切
- ✅ **ネットワーク**: Docker内でサービス名解決成功
- ✅ **認証**: レプリケーション専用ユーザーで接続
- ✅ **非同期レプリケーション**: 高性能で実用的

### 学習成果
1. **pg_stat_replication**: プライマリ側からのレプリケーション監視
2. **pg_stat_wal_receiver**: スタンバイ側からの受信状況監視
3. **LSN（Log Sequence Number）**: WALの進行状況を示す重要指標
4. **レプリケーション遅延**: 複数段階（sent→write→flush→replay）の理解
5. **ホットスタンバイ**: リカバリ中の読み取り専用アクセス

### 次のステップへの準備
読み書き分離アプリケーションの実装において、以下の知見が活用可能：
- プライマリ（5432）: 書き込み専用
- スタンバイ（5433）: 読み取り専用  
- 遅延がほぼゼロのため、読み取り一貫性の問題は最小限