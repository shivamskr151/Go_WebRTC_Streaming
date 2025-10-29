#!/bin/bash

# Local TURN Server Setup for WebRTC Development
# This script sets up a local TURN server to help with ICE connection issues

echo "🔧 Setting up Local TURN Server for WebRTC Development"
echo "====================================================="

# Check if coturn is installed
if ! command -v turnserver &> /dev/null; then
    echo "❌ coturn is not installed. Installing..."
    
    # Install coturn based on OS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install coturn
        else
            echo "Please install Homebrew first: https://brew.sh/"
            exit 1
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        if command -v apt-get &> /dev/null; then
            sudo apt-get update
            sudo apt-get install -y coturn
        elif command -v yum &> /dev/null; then
            sudo yum install -y coturn
        else
            echo "Please install coturn manually for your Linux distribution"
            exit 1
        fi
    else
        echo "Unsupported OS. Please install coturn manually."
        exit 1
    fi
fi

echo "✅ coturn is installed"

# Create TURN server configuration
TURN_CONFIG="/tmp/turnserver.conf"
cat > "$TURN_CONFIG" << EOF
# TURN Server Configuration for Local Development
listening-port=3478
tls-listening-port=5349
listening-ip=127.0.0.1
external-ip=127.0.0.1
realm=localhost
server-name=localhost
user=webrtc:webrtc123
user=test:test123
# No authentication for development
no-auth
# Allow all IPs for development
allowed-peer-ip=0.0.0.0-255.255.255.255
denied-peer-ip=0.0.0.0-0.255.255.255
denied-peer-ip=10.0.0.0-10.255.255.255
denied-peer-ip=172.16.0.0-172.31.255.255
denied-peer-ip=192.168.0.0-192.168.255.255
denied-peer-ip=169.254.0.0-169.254.255.255
denied-peer-ip=127.0.0.0-127.255.255.255
# Logging
log-file=/tmp/turnserver.log
verbose
EOF

echo "📝 TURN server configuration created: $TURN_CONFIG"

# Start TURN server
echo "🚀 Starting TURN server..."
echo "TURN Server will run on:"
echo "  - STUN: stun://127.0.0.1:3478"
echo "  - TURN: turn://127.0.0.1:3478 (username: webrtc, password: webrtc123)"
echo "  - TURN: turn://127.0.0.1:3478 (username: test, password: test123)"
echo ""
echo "Press Ctrl+C to stop the TURN server"
echo ""

# Start the TURN server
turnserver -c "$TURN_CONFIG"
