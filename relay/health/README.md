# Health Check

QumoリレーサーバーのヘルスチェックAPI実装です。

## エンドポイント

### 1. `/health/live` - Liveness Probe

**用途**: Kubernetesのliveness probeに使用

プロセスが生きているかの基本チェックです。常に200 OKを返します（プロセスが完全に停止していない限り）。

**レスポンス例**:
```json
{
  "status": "alive"
}
```

**HTTPステータスコード**:
- `200 OK` - プロセスは生存中

### 2. `/health/ready` - Readiness Probe

**用途**: Kubernetesのreadiness probeに使用

サーバーがトラフィックを受け入れ可能な状態かをチェックします。

**判定基準**:
- アクティブ接続数が正常範囲内か
- upstreamが必要な場合、upstream接続が確立されているか

**レスポンス例（準備完了）**:
```json
{
  "ready": true
}
```

**レスポンス例（準備未完了）**:
```json
{
  "ready": false,
  "reason": "upstream_not_connected"
}
```

**HTTPステータスコード**:
- `200 OK` - サービス準備完了
- `503 Service Unavailable` - サービス準備未完了

**失敗理由**:
- `invalid_connection_state` - 接続数が異常状態
- `upstream_not_connected` - 必要なupstream接続が未接続

### 3. `/health` - 詳細ステータス

**用途**: モニタリング、デバッグ、詳細情報の取得

サーバーの詳細なステータス情報を返します。

**レスポンス例**:
```json
{
  "status": "healthy",
  "timestamp": "2025-12-14T10:30:00Z",
  "uptime": "2h15m30s",
  "active_connections": 42,
  "upstream_connected": true,
  "version": "v0.1.0"
}
```

**ステータス値**:
- `healthy` - 正常
- `degraded` - 機能低下（upstreamが必要だが未接続など）
- `unhealthy` - 異常

**HTTPステータスコード**:
- `200 OK` - healthy または degraded
- `503 Service Unavailable` - unhealthy

## Kubernetesの設定例

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: qumo-relay
spec:
  containers:
  - name: relay
    image: qumo-relay:latest
    ports:
    - containerPort: 8080
    
    # Liveness Probe - プロセスの生存確認
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 2
      failureThreshold: 3
    
    # Readiness Probe - トラフィック受け入れ準備確認
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
      timeoutSeconds: 2
      failureThreshold: 2
```

## 使用方法

### 初期化

```go
import "github.com/okdaichi/qumo/relay/health"

// ヘルスチェックハンドラーを作成
healthHandler := health.NewStatusHandler()

// upstreamが必須の場合は設定
healthHandler.SetUpstreamRequired(true)
```

### エンドポイントの登録

```go
// 標準的なhttp.ServeMuxへの登録
mux := http.NewServeMux()
mux.HandleFunc("/health/live", healthHandler.ServeLive)
mux.HandleFunc("/health/ready", healthHandler.ServeReady)
mux.HandleFunc("/health", healthHandler.ServeHTTP)
```

### 状態の更新

```go
// 接続の追加・削除
healthHandler.IncrementConnections()
defer healthHandler.DecrementConnections()

// upstream接続状態の更新
healthHandler.SetUpstreamConnected(true)
```

## ベストプラクティス

1. **Liveness Probeは失敗しにくく設定する**
   - `failureThreshold: 3` 以上を推奨
   - 失敗するとPodが再起動されるため慎重に

2. **Readiness Probeは敏感に設定する**
   - `failureThreshold: 2` 程度を推奨
   - 失敗してもPodは再起動されない（Serviceから除外されるだけ）

3. **upstream接続の設定**
   - リレーサーバーがupstream必須の構成なら `SetUpstreamRequired(true)` を呼び出す
   - 独立動作可能なら不要（デフォルト: false）

4. **初期遅延の調整**
   - アプリケーション起動時間に応じて `initialDelaySeconds` を調整
   - 起動が遅い場合は長めに設定

5. **モニタリング**
   - `/health` エンドポイントを定期的に監視
   - `degraded` 状態を検知したらアラート
