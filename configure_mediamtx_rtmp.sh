#!/bin/bash

# Configure MediaMTX to pull from RTMP source via API
# This script configures MediaMTX after it starts

RTMP_URL="rtmp://safetycaptain.arresto.in/camera_0051/0051?username=wrakash&password=akash@1997"

echo "Configuring MediaMTX to pull RTMP stream..."
echo "RTMP URL: $RTMP_URL"

# URL encode the RTMP URL
ENCODED_URL=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$RTMP_URL', safe=':/?'))")

# Configure via MediaMTX API v3
# Note: MediaMTX doesn't support RTMP pull directly, but we can use runOnReady hook
# or configure it to use FFmpeg via external source

echo ""
echo "Since MediaMTX doesn't support RTMP pull directly, you have two options:"
echo ""
echo "Option 1: Camera pushes to MediaMTX (Recommended)"
echo "  Configure your camera to stream to: rtmp://localhost:1935/live"
echo ""
echo "Option 2: Use MediaMTX API to configure FFmpeg-based pull"
echo "  This requires configuring via runOnReady hook in the path"
echo ""
echo "For now, MediaMTX is configured to accept RTMP publishers."
echo "Update your camera to push to: rtmp://localhost:1935/live"

