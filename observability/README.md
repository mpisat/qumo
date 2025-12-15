# Observability

MoQ リレーサーバーのための統合オブザーバビリティパッケージ。

## 3シグナル対応

| シグナル | バックエンド | エクスポート |
|---------|------------|------------|
| **Traces** | OpenTelemetry | OTLP gRPC |
| **Logs** | slog → OTel Bridge | OTLP gRPC |
| **Metrics** | Prometheus | `/metrics` HTTP |

## クイックスタート

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/okdaichi/qumo/observability"
)

func main() {
    ctx := context.Background()
    
    // Setup
    err := observability.Setup(ctx, observability.Config{
        Service:      "qumo-relay",
        Version:      "v0.1.0",
        TraceAddr:    "localhost:4317",  // OTel Collector
        LogAddr:      "localhost:4317",  // 同じエンドポイントでOK
        SamplingRate: 0.1,               // 10%サンプリング
        Metrics:      true,
    })
    if err != nil {
        panic(err)
    }
    defer observability.Shutdown(ctx)
    
    // slog は自動的に OTel に送信される
    slog.Info("server started")
    
    // Span の作成
    ctx, span := observability.Start(ctx, "operation",
        observability.Track("video"),
    )
    defer span.End()
    
    // エラー時
    if err := doSomething(); err != nil {
        span.Error(err, "operation failed")
        return
    }
}
```

## 環境変数

| 変数 | 説明 | デフォルト |
|-----|------|---------|
| `OTEL_ENDPOINT` | トレース・ログ共通エンドポイント | (無効) |
| `OTEL_LOG_ENDPOINT` | ログ専用エンドポイント | `OTEL_ENDPOINT` |
| `OTEL_SAMPLING_RATE` | サンプリング率 (0.0-1.0) | 1.0 |

## API

### Setup / Shutdown

```go
// 初期化（main で1回）
observability.Setup(ctx, observability.Config{...})

// 終了時
defer observability.Shutdown(ctx)
```

### Span

```go
// シンプル
ctx, span := observability.Start(ctx, "operation")
defer span.End()

// 属性付き
ctx, span := observability.Start(ctx, "relay.write",
    observability.Track("video"),
    observability.GroupSequence(42),
)

// オプション付き（レイテンシメトリクス等）
ctx, span := observability.StartWith(ctx, "relay.receive",
    observability.Attrs(observability.Track("video")),
    observability.Latency(recorder.LatencyObs("receive")),
    observability.OnEnd(cleanup),
)

// エラー記録
span.Error(err, "failed to write")

// イベント追加
span.Event("cache.hit", observability.Frames(10))
```

### Metrics (Recorder)

```go
rec := observability.NewRecorder("video")

rec.GroupReceived()          // グループ受信
rec.CacheHit()               // キャッシュヒット
rec.CacheMiss()              // キャッシュミス
rec.Catchup(lag)             // キャッチアップ（ラグ付き）
rec.Broadcast(duration, total, notified)  // ブロードキャスト

rec.IncSubscribers()         // 購読者増加
rec.DecSubscribers()         // 購読者減少
```

### 属性ヘルパー

```go
observability.Track("video")        // moq.track
observability.Group(42)             // moq.group
observability.GroupSequence(42)     // moq.group (エイリアス)
observability.Frames(10)            // moq.frames
observability.Broadcast("/live")    // moq.broadcast
observability.Subscribers(5)        // moq.subscribers
```

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Code                          │
├─────────────────────────────────────────────────────────────┤
│  slog.Info()    │  Start()      │  NewRecorder()            │
│       ↓         │      ↓        │       ↓                   │
│  otelslog       │  OTel Span    │  Prometheus               │
│  Bridge         │               │  Metrics                  │
├─────────────────┼───────────────┼───────────────────────────┤
│        OTLP gRPC Export         │  /metrics HTTP Scrape     │
└─────────────────┴───────────────┴───────────────────────────┘
         ↓                                    ↓
┌─────────────────┐              ┌────────────────────────────┐
│  OTel Collector │              │  Prometheus Server         │
│  → Jaeger/Tempo │              │  → Grafana                 │
└─────────────────┘              └────────────────────────────┘
```

## MoQ 標準準拠

- **サーバーサイドのみ**: MoQ フレームにトレースデータを注入しない
- **相関**: track_name, group_id, timestamp で Trace ↔ Metrics を紐付け
- **標準準拠**: MoQ Transport プロトコルを改変しない
