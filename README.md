# Go WebRTC Streaming Server

A comprehensive Go-based WebRTC streaming server that supports RTMP streaming and snapshot capture functionality.

## 🚀 Features

- **RTMP Streaming**: Connect to RTMP streams and forward to WebRTC clients
- **RTMP Server**: Accept RTMP streams and forward to WebRTC
- **WebRTC Streaming**: Real-time video streaming using pion/webrtc
- **Snapshot Capture**: Capture JPEG snapshots via API
- **Modern Web Interface**: Beautiful, responsive web client
- **RESTful API**: Complete API for stream management
- **Real-time Status**: Live monitoring of connections and streams

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
- FFmpeg (for RTMP processing)

### Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd golang-webrtc-streaming
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Configure environment variables** (optional):
   ```bash
   export HTTP_PORT=8080
   export RTMP_PORT=1935
   ```

4. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

## 🌐 Usage

### Web Interface

1. Open your browser and navigate to `http://localhost:8080`
2. Click "Start Stream" to begin WebRTC streaming
3. Use "Capture Snapshot" to take JPEG snapshots
4. Monitor system status in real-time

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
```

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
├── web/
│   ├── templates/
│   │   └── index.html          # Web interface
│   └── static/
│       └── css/
│           └── style.css       # Additional styles
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

```bash
go build -o webrtc-server cmd/server/main.go
```

### Testing

```bash
go test ./...
```

### Running with Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o webrtc-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/webrtc-server .
COPY --from=builder /app/web ./web
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

### Logs

The application uses structured logging with different levels:
- `INFO`: General information
- `WARN`: Warning messages
- `ERROR`: Error conditions

## 📚 Dependencies

- [pion/webrtc](https://github.com/pion/webrtc) - WebRTC implementation
- [gin-gonic/gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [deepch/vdk](https://github.com/deepch/vdk) - Video development kit
- [sirupsen/logrus](https://github.com/sirupsen/logrus) - Structured logging

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
- [Gin](https://github.com/gin-gonic/gin) for the web framework

## 📞 Support

For issues and questions:
- Create an issue on GitHub
- Check the troubleshooting section
- Review the API documentation

---

**Happy Streaming! 🎥**
