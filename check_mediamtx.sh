#!/bin/bash

echo "========================================="
echo "MediaMTX URL Information"
echo "========================================="
echo ""

echo "📡 MediaMTX Status:"
docker ps --filter "name=mediamtx" --format "table {{.Names}}\t{{.Status}}"
echo ""

echo "🔗 MediaMTX URLs:"
echo ""
echo "RTMP Ingest (publish stream here):"
echo "  rtmp://localhost:1935/live"
echo "  rtmp://mediamtx:1935/live (from other containers)"
echo ""
echo "RTSP Playback (pull stream from here):"
echo "  rtsp://localhost:8554/live"
echo "  rtsp://mediamtx:8554/live (from other containers)"
echo ""
echo "HTTP API:"
echo "  http://localhost:8888"
echo ""
echo "Metrics:"
echo "  http://localhost:9998/metrics"
echo ""

echo "📋 Check MediaMTX Paths:"
echo "  curl http://localhost:8888/v3/paths/list"
echo ""
curl -s http://localhost:8888/v3/paths/list 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (API not accessible or curl failed)"
echo ""

echo "📊 Check Active Sessions:"
echo "  curl http://localhost:8888/v3/sessions/list"
echo ""
curl -s http://localhost:8888/v3/sessions/list 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "  (API not accessible or curl failed)"
echo ""

echo "🧪 Test RTSP Endpoint:"
echo "  ffprobe rtsp://localhost:8554/live"
echo ""

echo "========================================="

