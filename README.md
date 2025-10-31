# Go WebRTC Streaming Server

A comprehensive Go-based WebRTC streaming server with a modern React frontend that supports RTMP streaming and snapshot capture functionality.

## 🚀 Features

- **RTMP Streaming**: Connect to RTMP streams and forward to WebRTC clients
- **RTMP Server**: Accept RTMP streams and forward to WebRTC
- **WebRTC Streaming**: Real-time video streaming using pion/webrtc
- **Snapshot Capture**: Capture JPEG snapshots via API
- **Modern React Frontend**: Beautiful, responsive TypeScript/React web client with Tailwind CSS
- **Component-Based Architecture**: Clean, modular React component structure
- **RESTful API**: Complete API for stream management
- **Real-time Status**: Live monitoring of connections and streams
- **Single-Screen Layout**: Optimized UI that fits everything on one screen

## 📋 Architecture

```
[RTMP Camera] → [Go RTMP Client] ↘
                                  → WebRTC Server (pion/webrtc) → Browser
[RTMP Stream] → [Go RTMP Server] ↗

API Endpoints:
- /api/offer → WebRTC offer/answer
- /api/snapshot → JPEG snapshot capture
- /api/status → System status
- /api/peers → Connected peers info
```

## 🛠️ Installation

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ and npm (for frontend)
- FFmpeg (for RTMP processing)

### Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd golang-webrtc-streaming
   ```

2. **Install all dependencies** (Go + Frontend):
   ```bash
   make deps
   ```
   
   Or install separately:
   ```bash
   # Install Go dependencies
   go mod tidy
   
   # Install frontend dependencies
   cd web && npm install
   ```

3. **Build the application**:
   ```bash
   make build
   ```
   
   This builds both the React frontend and Go backend.

4. **Configure environment variables** (optional):
   ```bash
   export HTTP_PORT=8080
   export RTMP_PORT=1935
   export RTMP_URL="rtmp://your-camera-url"
   ```

5. **Run the server**:
   ```bash
   make run
   # or
   go run cmd/server/main.go
   ```

## 🌐 Usage

### Web Interface

1. **Start the server**:
   ```bash
   make run
   ```

2. **Open your browser** and navigate to `http://localhost:8080`

3. **Use the interface**:
   - Click "Start Stream" to begin WebRTC streaming
   - Use "Capture Snapshot" to take JPEG snapshots
   - Click on captured snapshots to view them in full-screen
   - Switch between RTSP and RTMP sources
   - Monitor system status in real-time

### Frontend Development

For frontend development with hot-reload:

```bash
# Terminal 1: Run backend
make run

# Terminal 2: Run frontend dev server
cd web && npm run dev
```

The frontend dev server runs on `http://localhost:5173` (Vite default) with proxy to the backend API.

### API Endpoints

#### WebRTC Offer
```bash
POST /api/offer
Content-Type: application/json

{
  "sdp": "{\"type\":\"offer\",\"sdp\":\"v=0\\r\\n...\"}"
}
```

#### Snapshot Capture
```bash
GET /api/snapshot
```

#### System Status
```bash
GET /api/status
```

#### Peers Information
```bash
GET /api/peers
```

### RTMP Stream Integration

The server automatically connects to the configured RTMP URL. Supported formats:
- H.264 video
- H.265 video (requires transcoding)
- AAC audio

### RTMP Streaming

Stream to the server using:
```bash
ffmpeg -i input.mp4 -c copy -f flv rtmp://localhost:1935/live
```

### Configure your RTMP camera URL (no hardcoding)

Set `RTMP_URL` via environment variable so the server picks it up at runtime:

```bash
export RTMP_URL="rtmp://safetycaptain.arresto.in/camera_0051/0051?username=wrakash&password=akash@1997"
go build -o webrtc-server cmd/server/main.go && ./webrtc-server
```

With Docker Compose, the same variable is passed in `docker-compose.yml` under `environment:`; edit it or override at run:

```bash
RTMP_URL="rtmp://..." docker compose up --build
```

At startup the server logs will show the active RTMP stream value.

## 📁 Project Structure

```
golang-webrtc-streaming/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── webrtc/
│   │   └── manager.go           # WebRTC peer management
│   ├── rtmp/
│   │   └── server.go           # RTMP server implementation
│   └── server/
│       └── http.go              # HTTP server and API routes
├── web/                         # React Frontend
│   ├── src/
│   │   ├── components/         # React components
│   │   │   ├── Header/
│   │   │   ├── VideoPlayer/
│   │   │   ├── StreamControls/
│   │   │   ├── SourceSelector/
│   │   │   ├── StatusPanel/
│   │   │   ├── SnapshotViewer/
│   │   │   └── MessageToast/
│   │   ├── hooks/              # Custom React hooks
│   │   │   ├── useWebRTC.ts
│   │   │   └── useStatus.ts
│   │   ├── services/           # API service layer
│   │   │   └── api.ts
│   │   ├── types/              # TypeScript type definitions
│   │   │   └── api.ts
│   │   ├── utils/              # Utility functions
│   │   ├── App.tsx             # Main App component
│   │   ├── main.tsx            # React entry point
│   │   └── index.css           # Global styles with Tailwind
│   ├── dist/                   # Built frontend (generated)
│   ├── package.json            # Frontend dependencies
│   ├── vite.config.ts          # Vite configuration
│   ├── tailwind.config.js      # Tailwind CSS configuration
│   ├── tsconfig.json           # TypeScript configuration
│   └── index.html              # HTML template
├── Makefile                    # Build automation
├── go.mod                      # Go module definition
└── README.md                   # This file
```

## ⚙️ Configuration

The application can be configured using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | 8080 | HTTP server port |
| `RTMP_PORT` | 1935 | RTMP server port |

## 🔧 Development

### Building

**Build everything (frontend + backend):**
```bash
make build
```

**Build only backend** (requires frontend to be built separately):
```bash
make build-backend
```

**Build only frontend:**
```bash
make build-frontend
```

**Manual build:**
```bash
# Build frontend
cd web && npm run build

# Build backend
go build -o webrtc-server cmd/server/main.go
```

### Development Workflow

**Backend development:**
```bash
make run
# or with hot reload (requires air)
make dev
```

**Frontend development:**
```bash
cd web && npm run dev
```

**Clean build artifacts:**
```bash
make clean          # Clean everything
make clean-frontend # Clean only frontend
```

### Testing

```bash
make test              # Run all tests
make test-coverage     # Run tests with coverage
```

### Other Make Commands

```bash
make help              # Show all available commands
make fmt               # Format Go code
make lint              # Lint code
make docker-build      # Build Docker image
make cross-compile     # Cross-compile for different platforms
```

### Running with Docker

The Dockerfile includes a multi-stage build that:
1. Builds the React frontend using Node.js
2. Builds the Go backend
3. Copies both to the final image

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Or use Docker Compose
make docker-up
```

**Dockerfile structure:**
```dockerfile
# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o webrtc-server cmd/server/main.go

# Stage 3: Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/webrtc-server .
COPY --from=frontend-builder /app/web/dist ./web/dist
CMD ["./webrtc-server"]
```

## 🐛 Troubleshooting

### Common Issues

1. **RTMP Connection Failed**
   - Verify camera URL is accessible
   - Check network connectivity
   - Ensure camera supports H.264 codec

2. **WebRTC Connection Issues**
   - Check browser WebRTC support
   - Verify STUN server accessibility
   - Check firewall settings

3. **RTMP Stream Not Working**
   - Verify RTMP server is running
   - Check stream format compatibility
   - Ensure proper authentication if required

4. **Frontend Build Issues**
   - Ensure Node.js 18+ is installed
   - Delete `node_modules` and `package-lock.json`, then run `npm install`
   - Check that `web/dist` directory exists after build
   - Verify Tailwind CSS is properly configured

5. **UI Not Displaying Properly**
   - Hard refresh browser (Cmd+Shift+R / Ctrl+Shift+R)
   - Clear browser cache
   - Ensure frontend was built before running backend
   - Check browser console for errors

### Logs

The application uses structured logging with different levels:
- `INFO`: General information
- `WARN`: Warning messages
- `ERROR`: Error conditions

## 📚 Dependencies

### Backend (Go)
- [pion/webrtc](https://github.com/pion/webrtc) - WebRTC implementation
- [gin-gonic/gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [deepch/vdk](https://github.com/deepch/vdk) - Video development kit
- [sirupsen/logrus](https://github.com/sirupsen/logrus) - Structured logging

### Frontend (React/TypeScript)
- [React](https://react.dev/) - UI framework
- [TypeScript](https://www.typescriptlang.org/) - Type safety
- [Vite](https://vitejs.dev/) - Build tool and dev server
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Pion WebRTC](https://github.com/pion/webrtc) for the excellent WebRTC implementation
- [VDK](https://github.com/deepch/vdk) for video processing capabilities
- [Gin](https://github.com/gin-gonic/gin) for the Go web framework
- [React](https://react.dev/) for the frontend framework
- [Vite](https://vitejs.dev/) for the fast build tool and dev server
- [Tailwind CSS](https://tailwindcss.com/) for the utility-first CSS framework

## 📞 Support

For issues and questions:
- Create an issue on GitHub
- Check the troubleshooting section
- Review the API documentation

---

**Happy Streaming! 🎥**
