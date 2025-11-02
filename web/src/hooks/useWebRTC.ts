import { useState, useRef, useCallback } from 'react'
import { sendOffer } from '../services/api'
import { getWebRTCConfig } from '../utils/webrtc-config'

type MessageCallback = (type: 'success' | 'error' | '', text: string, duration?: number) => void
type TrackReceivedCallback = (stream: MediaStream) => void

/**
 * Optimize SDP for ultra-low latency streaming
 * Simplified version to avoid breaking SDP negotiation
 */
function optimizeSDPForLowLatency(sdp: string): string {
  // Return SDP as-is for now - server already handles packetization-mode=1
  // Adding modifications here can break SDP negotiation
  // The backend WebRTC manager already has packetization-mode=1 configured
  return sdp
}

interface UseWebRTCReturn {
  isConnected: boolean
  isConnecting: boolean
  startConnection: (videoElement: HTMLVideoElement | null) => Promise<void>
  stopConnection: (videoElement: HTMLVideoElement | null, onMessage?: MessageCallback) => void
}

/**
 * Custom hook for WebRTC connection management
 */
export const useWebRTC = (
  onTrackReceived?: TrackReceivedCallback,
  onMessage?: MessageCallback
): UseWebRTCReturn => {
  const [isConnected, setIsConnected] = useState<boolean>(false)
  const [isConnecting, setIsConnecting] = useState<boolean>(false)
  const pcRef = useRef<RTCPeerConnection | null>(null)

  const startConnection = useCallback(
    async (videoElement: HTMLVideoElement | null) => {
      try {
        setIsConnecting(true)
        if (onMessage) onMessage('', '', 0)

        const config = getWebRTCConfig()
        const pc = new RTCPeerConnection(config)

        // Handle incoming tracks
        pc.ontrack = (event: RTCTrackEvent) => {
          console.log('Received track:', event.track.kind, event.track)
          if (event.track.kind === 'video') {
            let stream = event.streams && event.streams[0]
            if (!stream) {
              stream = new MediaStream()
              stream.addTrack(event.track)
            }
            if (videoElement) {
              videoElement.srcObject = stream
            }
            if (onTrackReceived) onTrackReceived(stream)
            if (onMessage) onMessage('success', 'Video track received!')
          }
        }

        // Handle connection state changes
        pc.onconnectionstatechange = () => {
          console.log('Connection state:', pc.connectionState)
          setIsConnected(pc.connectionState === 'connected')
        }

        // Handle ICE connection state changes
        pc.oniceconnectionstatechange = () => {
          console.log('ICE connection state:', pc.iceConnectionState)
          setIsConnected(pc.iceConnectionState === 'connected')
        }

        // Handle ICE candidates
        pc.onicecandidate = (event: RTCPeerConnectionIceEvent) => {
          if (event.candidate) {
            console.log('ICE candidate:', event.candidate)
          } else {
            console.log('ICE gathering complete')
          }
        }

        // Create data channel
        const dataChannel = pc.createDataChannel('streaming', { ordered: true })
        dataChannel.onopen = () => console.log('Data channel opened')
        dataChannel.onmessage = (event: MessageEvent) => console.log('Received message:', event.data)

        // Request to receive media with low-latency optimizations
        pc.addTransceiver('video', { 
          direction: 'recvonly',
        })
        pc.addTransceiver('audio', { direction: 'recvonly' })

        // Create and send offer (no additional options needed - addTransceiver handles it)
        const offer = await pc.createOffer()
        
        // SDP is already optimized by backend, no need to modify here
        await pc.setLocalDescription(offer)

        const answer = await sendOffer(offer)
        const answerDesc: RTCSessionDescriptionInit = {
          type: 'answer',
          sdp: answer.sdp,
        }
        await pc.setRemoteDescription(answerDesc)

        pcRef.current = pc
        setIsConnected(true)
        if (onMessage) onMessage('success', 'Stream started successfully!')
      } catch (error) {
        console.error('Error starting stream:', error)
        const errorMessage = error instanceof Error ? error.message : 'Unknown error'
        if (onMessage) onMessage('error', `Failed to start stream: ${errorMessage}`)
        throw error
      } finally {
        setIsConnecting(false)
      }
    },
    [onTrackReceived, onMessage]
  )

  const stopConnection = useCallback(
    (videoElement: HTMLVideoElement | null, onMessage?: MessageCallback) => {
      if (pcRef.current) {
        pcRef.current.close()
        pcRef.current = null
      }
      if (videoElement) {
        videoElement.srcObject = null
      }
      setIsConnected(false)
      if (onMessage) onMessage('success', 'Stream stopped')
    },
    []
  )

  return {
    isConnected,
    isConnecting,
    startConnection,
    stopConnection,
  }
}
