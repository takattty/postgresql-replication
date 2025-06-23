#!/bin/bash
set -e

# スタンバイサーバーの初期設定スクリプト

echo "Setting up PostgreSQL Standby Server for Streaming Replication"

# スタンバイサーバーは pg_basebackup で初期化されるため、
# このスクリプトでは追加の設定は行わない

echo "Standby server is ready for streaming replication"
echo "Primary connection: postgres-primary:5432"
echo "Replication user: replicator"