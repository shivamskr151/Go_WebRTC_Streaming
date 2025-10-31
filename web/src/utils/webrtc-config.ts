import { RTCConfiguration } from 'webrtc'

/**
 * WebRTC configuration
 */
export const getWebRTCConfig = (): RTCConfiguration => ({
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
    { urls: 'stun:stun2.l.google.com:19302' },
    { urls: 'stun:stun3.l.google.com:19302' },
    { urls: 'stun:stun4.l.google.com:19302' },
    {
      urls: 'turn:127.0.0.1:3478',
      username: 'webrtc',
      credential: 'webrtc123',
    },
    {
      urls: 'turn:127.0.0.1:3478',
      username: 'test',
      credential: 'test123',
    },
  ],
  iceTransportPolicy: 'all',
  bundlePolicy: 'balanced',
  rtcpMuxPolicy: 'require',
  iceCandidatePoolSize: 10,
})
