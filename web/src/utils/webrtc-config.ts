/**
 * WebRTC configuration
 * RTCConfiguration is a built-in Web API type, no import needed
 */
export const getWebRTCConfig = (): RTCConfiguration => {
  // For localhost, we don't need STUN - direct connection should work
  const isLocalhost = window.location.hostname === 'localhost' || 
                     window.location.hostname === '127.0.0.1' ||
                     window.location.hostname === ''
  
  const config: RTCConfiguration = {
    iceServers: isLocalhost 
      ? [
          // Still include STUN for localhost as fallback
          { urls: 'stun:stun.l.google.com:19302' },
        ]
      : [
          // Use multiple STUN servers for remote connections for better reliability
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
          { urls: 'stun:stun2.l.google.com:19302' },
        ],
    iceTransportPolicy: 'all', // Allow both host and relay candidates
    bundlePolicy: 'max-compat', // More compatible bundle policy
    rtcpMuxPolicy: 'require', // Require RTCP multiplexing
    iceCandidatePoolSize: 0, // Disable pre-gathering for faster connection
    // Add connection constraints for better reliability
  }
  
  return config
}
