#!/bin/bash

# Script to configure MediaMTX to pull from RTMP source
# This uses MediaMTX API to configure an external FFmpeg source

RTMP_URL="rtmp://safetycaptain.arresto.in/camera_0051/0051?username=wrakash&password=akash@1997"
PATH_NAME="live"

echo "Configuring MediaMTX to pull RTMP stream via FFmpeg..."
echo "RTMP URL: $RTMP_URL"
echo ""

# Wait for MediaMTX to be ready
echo "Waiting for MediaMTX API to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8888/v3/config/global/get > /dev/null 2>&1; then
        echo "MediaMTX API is ready!"
        break
    fi
    sleep 1
done

# Configure the path with FFmpeg exec source via API
# Note: This requires MediaMTX to have FFmpeg available

echo ""
echo "Configuring path '$PATH_NAME' to pull from RTMP..."
echo ""
echo "Since MediaMTX doesn't natively support RTMP pull, you need to:"
echo ""
echo "1. Run FFmpeg externally to pull RTMP and push to MediaMTX:"
echo "   ffmpeg -i '$RTMP_URL' -c copy -f flv rtmp://localhost:1935/live"
echo ""
echo "OR"
echo ""
echo "2. Configure your camera/streaming source to push directly to MediaMTX:"
echo "   rtmp://localhost:1935/live"
echo ""
echo "MediaMTX is ready to receive RTMP streams on port 1935"
echo "Your Go server will pull from: rtsp://mediamtx:8554/live"

