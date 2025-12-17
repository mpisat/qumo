# qumo Web Demo

Modern web interface for testing the qumo MoQ relay server, built with **Deno 2.x** and **Vite**.

## 🚀 Quick Start

```bash
# From project root (starts relay + web demo)
mage web

# Or directly with Deno
cd web
deno task dev
```

Open browser at http://localhost:5173

## ✨ Features

- 📹 **Camera/Screen Publishing**: Capture video at 30 FPS with JPEG encoding
- 📡 **MoQ Subscriber**: Real-time stream playback via TrackReader
- 🔄 **Full MoQ Protocol**: Built with @okdaichi/moq Client/Session API
- 🚀 **WebTransport**: QUIC-based connection to relay server
- 🦕 **Deno 2.x**: Modern runtime with JSR support
- ⚡ **Vite HMR**: Instant hot reload
- 🎯 **Type-safe**: Full TypeScript with strict mode

## 📦 Tech Stack

- **Runtime**: [Deno 2.x](https://deno.com/)
- **Frontend**: [Vite](https://vitejs.dev/) + TypeScript
- **Protocol**: [@okdaichi/moq](https://jsr.io/@okdaichi/moq) from JSR
- **Relay**: qumo MoQ relay server

## 📁 Project Structure

```
web/
├── main.ts              # Application entry point
├── lib/
│   ├── types.ts         # TypeScript type definitions
│   ├── dom.ts           # DOM utility functions
│   ├── connection.ts    # MoQ connection management
│   ├── publisher.ts     # Video publishing logic
│   └── subscriber.ts    # Video subscribing logic
├── index.html           # UI layout
├── deno.json            # Deno configuration
└── vite.config.ts       # Vite bundler config
```

## 🛠️ Development

```bash
# Start dev server
deno task dev

# Build for production
deno task build

# Preview production build
deno task preview

# From project root with Mage
mage web         # Start relay + web demo
mage webBuild    # Build production bundle
mage webClean    # Clean build artifacts
```

## 📝 Implementation Status

### ✅ Completed

- **Full MoQ Client/Session integration** with @okdaichi/moq
- **Publisher**: TrackWriter with Group/Frame API
  - Camera capture (1280x720@30fps)
  - Screen sharing (1920x1080@30fps)
  - Real-time JPEG encoding (85% quality)
- **Subscriber**: TrackReader with frame decoding
  - Group/frame reading
  - JPEG decode and display
  - Canvas stream playback
- Modern TypeScript architecture
- UI controls with source selector (camera/screen)
- Error handling and logging
- Vite + Deno integration
- Cross-platform Mage automation

### 🚧 Future Enhancements

- AudStart relay**: `mage up` (or run `mage web` to auto-start)

2. **Open browser**: http://localhost:5173
3. **Connect**: Click "Connect to Relay" (default: https://localhost:5000)
4. **Publish**:
   - Select source: 📷 Camera or 🖥️ Screen Share
   - Click "Start Publishing" to stream media
   - Video frames sent at 30 FPS via MoQ TrackWriter
5. **Subscribe**:
   - Enter track name (e.g., "my-stream")
   - Click "Subscribe" to receive stream
   - Watch published stream in real-time via TrackReader

## 🎯 How It Works

### Publishing Flow

```
Camera/Screen → Canvas (30fps) → JPEG encode → MoQ Frame → TrackWriter → Relay
```

1. Capture video from `getUserMedia()` or `getDisplayMedia()`
2. Draw frames to canvas at 30 FPS
3. Encode canvas as JPEG blob (85% quality)
4. Wrap in MoQ Frame with group structure
5. Send via `TrackWriter.openGroup()` → `writeFrame()` to relay

### Subscribing Flow

```
Relay → TrackReader → acceptGroup() → readFrame() → JPEG decode → Canvas → Video
```

1. Subscribe to broadcast path (`/live/{trackName}`)
2. Accept groups from `TrackReader.acceptGroup()`
3. Read frames from `GroupReader.readFrame()`
4. Decode JPEG blob to Image element
5. Display in video element via `canvas.captureStream()`

- Multi-track publishing/subscribing
- Real-time metrics dashboard

## 🧪 Usage

1. **Connect**: Enter relay URL (default: `https://localhost:5000`)
2. **Publish**: Click "Start Publishing" to share camera/mic
3. **Subscribe**: Enter track name and click "Subscribe"

## 🐛 Troubleshooting

### WebTransport Connection Failed

If you see `ERR_CONNECTION_RESET` or `Opening handshake failed`:

**Option 1: Accept Certificate (Recommended for Chrome)**
1. Open a new tab and navigate to https://localhost:4433
2. Chrome will show "Your connection is not private"
3. Click "Advanced" → "Proceed to localhost (unsafe)"
4. Return to the web demo and try connecting again

**Option 2: Enable Chrome Flag**
1. Navigate to `chrome://flags/#unsafely-treat-insecure-origin-as-secure`
2. Add `https://localhost:4433` to the list
3. Restart Chrome
4. Try connecting again

**Option 3: Use Firefox (More Lenient)**
Firefox may accept self-signed certificates more easily than Chrome for WebTransport.

### Dependencies not installing

Run `deno task dev` - dependencies auto-install on first run.

### Relay connection refused

- Ensure relay is running: `mage e2e` or `docker compose ps`
- Verify relay URL: default is `https://localhost:5000`

### Camera/mic access denied

- Grant browser permissions when prompted
- HTTPS required (accept self-signed cert in dev)
- Check system privacy settings

## 📚 References

- [Deno Web Development](https://docs.deno.com/runtime/fundamentals/web_dev/)
- [Deno 2025 Frontend Trends (JP)](https://zenn.dev/uki00a/articles/frontend-development-in-deno-2025-summer)
- [JSR Package Registry](https://jsr.io/)
- [MoQ Transport Spec](https://datatracker.ietf.org/doc/draft-ietf-moq-transport/)

## 🔐 Security

- HTTPS required for MediaDevices API
- Self-signed certs OK in development
- Production needs valid TLS certificates
