import { useState, useRef, useCallback, useEffect } from 'react'
import { sendOffer } from '../services/api'
import { getWebRTCConfig } from '../utils/webrtc-config'

type MessageCallback = (type: 'success' | 'error' | '', text: string, duration?: number) => void
type TrackReceivedCallback = (stream: MediaStream) => void

interface UseWebRTCReturn {
  isConnected: boolean
  isConnecting: boolean
  startConnection: (videoElement: HTMLVideoElement | null) => Promise<void>
  stopConnection: (videoElement: HTMLVideoElement | null, onMessage?: MessageCallback) => void
  reconnect: (videoElement: HTMLVideoElement | null) => Promise<void>
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
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const iceRestartTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const healthCheckIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const videoElementRef = useRef<HTMLVideoElement | null>(null)
  const reconnectAttemptsRef = useRef<number>(0)
  const maxReconnectAttempts = 3
  const reconnectFnRef = useRef<((videoElement: HTMLVideoElement | null) => Promise<void>) | null>(null)

  const startConnection = useCallback(
    async (videoElement: HTMLVideoElement | null) => {
      try {
        setIsConnecting(true)
        reconnectAttemptsRef.current = 0 // Reset reconnect attempts on manual start
        if (onMessage) onMessage('', '', 0)

        // Store video element reference
        videoElementRef.current = videoElement

        const config = getWebRTCConfig()
        const pc = new RTCPeerConnection(config)

        // Handle incoming tracks with connection validation
        pc.ontrack = async (event: RTCTrackEvent) => {
          const connectionState = pc.connectionState
          const iceState = pc.iceConnectionState
          
          console.log('📥 Received track:', {
            kind: event.track.kind,
            id: event.track.id,
            enabled: event.track.enabled,
            readyState: event.track.readyState,
            muted: event.track.muted,
            streamId: event.streams?.[0]?.id,
            connectionState: connectionState,
            iceConnectionState: iceState,
          })
          
          // Set up track event handlers immediately
          event.track.onended = () => {
            console.warn('⚠️ Track ended:', event.track.id)
          }
          
          event.track.onmute = () => {
            console.warn('⚠️ Track muted:', event.track.id)
          }
          
          event.track.onunmute = () => {
            console.log('✅ Track unmuted:', event.track.id)
          }
          
          if (event.track.kind === 'video') {
            let stream = event.streams && event.streams[0]
            if (!stream) {
              console.log('🆕 Creating new MediaStream for video track')
              stream = new MediaStream()
              stream.addTrack(event.track)
            } else {
              console.log('✅ Using existing stream:', stream.id)
            }
            
            // Log stream details
            console.log('📹 Video stream details:', {
              id: stream.id,
              active: stream.active,
              videoTracks: stream.getVideoTracks().length,
              audioTracks: stream.getAudioTracks().length,
            })
            
            if (videoElement) {
              // Set the stream as source
              console.log('🎥 Setting video element srcObject')
              
              // Clear any existing stream first
              if (videoElement.srcObject) {
                const oldStream = videoElement.srcObject as MediaStream
                oldStream.getTracks().forEach(track => track.stop())
              }
              
              videoElement.srcObject = stream
              
              // Ensure video element is ready
              if (videoElement.readyState < 2) {
                videoElement.load()
              }
              
              // Verify stream has active tracks
              const videoTracks = stream.getVideoTracks()
              console.log('📹 Video stream tracks:', {
                count: videoTracks.length,
                tracks: videoTracks.map(t => ({
                  id: t.id,
                  enabled: t.enabled,
                  readyState: t.readyState,
                  muted: t.muted,
                })),
              })
              
              // Set up track monitoring
              videoTracks.forEach(track => {
                track.addEventListener('ended', () => {
                  console.error('❌ Video track ended unexpectedly')
                  if (onMessage) {
                    onMessage('error', 'Video track ended. Connection may be lost.', 4000)
                  }
                })
                
                track.addEventListener('mute', () => {
                  console.warn('⚠️ Video track muted')
                })
                
                track.addEventListener('unmute', () => {
                  console.log('✅ Video track unmuted')
                })
              })
              
              // Wait a moment for stream to be ready
              await new Promise(resolve => setTimeout(resolve, 100))
              
              // Explicitly play the video to handle autoplay restrictions
              try {
                const playPromise = videoElement.play()
                if (playPromise !== undefined) {
                  await playPromise
                }
                console.log('✅ Video playback started successfully')
                console.log('Video element state:', {
                  readyState: videoElement.readyState,
                  paused: videoElement.paused,
                  muted: videoElement.muted,
                  currentTime: videoElement.currentTime,
                })
                if (onMessage) onMessage('success', 'Live stream started and playing!')
              } catch (error) {
                console.error('❌ Error playing video:', error)
                // Try again after a short delay in case it's a timing issue
                setTimeout(async () => {
                  if (videoElement && videoElement.srcObject === stream) {
                    try {
                      await videoElement.play()
                      console.log('✅ Video playback started on retry')
                      if (onMessage) onMessage('success', 'Live stream started!')
                    } catch (retryError) {
                      console.error('❌ Retry failed:', retryError)
                      if (onMessage) {
                        onMessage('error', 'Video playback failed. Please click on the page and try again.')
                      }
                    }
                  }
                }, 500)
              }
            } else {
              console.warn('⚠️ Video element not available')
            }
            
            if (onTrackReceived) onTrackReceived(stream)
          } else if (event.track.kind === 'audio') {
            // Handle audio tracks if needed
            console.log('🔊 Received audio track:', event.track.id)
          }
        }

        // Handle connection state changes
        pc.onconnectionstatechange = () => {
          const state = pc.connectionState
          const iceState = pc.iceConnectionState
          
          console.log('🔗 PeerConnection state changed:', {
            connectionState: state,
            iceConnectionState: iceState,
          })
          
          // Update connected state based on both connection and ICE states
          const isFullyConnected = (state === 'connected' || state === 'connecting') &&
            (iceState === 'connected' || iceState === 'completed' || iceState === 'checking')
          
          setIsConnected(isFullyConnected)
          
          if (state === 'failed') {
            console.error('❌ PeerConnection failed')
            setIsConnected(false)
            if (onMessage) {
              onMessage('error', 'Connection failed. Please try again.', 4000)
            }
          } else if (state === 'disconnected') {
            console.warn('⚠️ PeerConnection disconnected')
            setIsConnected(false)
            if (onMessage) {
              onMessage('', 'Connection interrupted', 2000)
            }
          } else if (state === 'closed') {
            console.log('🔒 PeerConnection closed')
            setIsConnected(false)
          } else if (state === 'connecting') {
            console.log('🔄 PeerConnection connecting...')
            setIsConnected(false)
            if (onMessage) {
              onMessage('', 'Establishing connection...', 0)
            }
          } else if (state === 'connected' && (iceState === 'connected' || iceState === 'completed')) {
            console.log('✅ PeerConnection fully connected!')
            setIsConnected(true)
            if (onMessage) {
              onMessage('success', 'Connection established!', 3000)
            }
          }
        }

        // Handle ICE connection state changes with reconnection logic
        pc.oniceconnectionstatechange = async () => {
          console.log('ICE connection state:', pc.iceConnectionState)
          const state = pc.iceConnectionState
          setIsConnected(state === 'connected' || state === 'completed')
          
          // Clear any pending reconnection attempts
          if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current)
            reconnectTimeoutRef.current = null
          }
          if (iceRestartTimeoutRef.current) {
            clearTimeout(iceRestartTimeoutRef.current)
            iceRestartTimeoutRef.current = null
          }
          
          // Handle connection states
          if (state === 'failed') {
            console.error('ICE connection failed - attempting automatic reconnection')
            setIsConnected(false)
            
            // Attempt automatic reconnection instead of ICE restart
            reconnectTimeoutRef.current = setTimeout(async () => {
              if (pc.iceConnectionState === 'failed' && pc.connectionState !== 'closed') {
                console.log('ICE failed - triggering reconnection...')
                if (onMessage) {
                  onMessage('', 'Connection failed. Attempting to reconnect...', 2000)
                }
                if (reconnectFnRef.current) {
                  await reconnectFnRef.current(videoElementRef.current)
                }
              }
            }, 2000)
            
            if (onMessage) {
              onMessage('error', 'Connection failed. Attempting to recover...', 3000)
            }
          } else if (state === 'disconnected') {
            console.warn('⚠️ ICE connection disconnected - monitoring for recovery')
            console.warn('Disconnection details:', {
              iceConnectionState: pc.iceConnectionState,
              connectionState: pc.connectionState,
              signalingState: pc.signalingState,
              iceGatheringState: pc.iceGatheringState,
              localDescription: pc.localDescription ? 'set' : 'not set',
              remoteDescription: pc.remoteDescription ? 'set' : 'not set',
            })
            
            setIsConnected(false)
            
            // Check if we have active tracks - if not, connection might be failing
            const receivers = pc.getReceivers()
            const tracks = receivers.map(r => r.track).filter(t => t && t.readyState === 'live')
            console.log('Active tracks count:', tracks.length)
            
            // Give it time to recover before treating as failure
            reconnectTimeoutRef.current = setTimeout(async () => {
              const currentICEState = pc.iceConnectionState
              const currentPCState = pc.connectionState
              
              if (currentICEState === 'disconnected' && currentPCState !== 'closed') {
                console.warn('⏰ Disconnection timeout - attempting automatic reconnection')
                console.warn('Final states:', {
                  iceConnectionState: currentICEState,
                  connectionState: currentPCState,
                })
                
                if (onMessage) {
                  onMessage('', 'Connection lost. Attempting to reconnect...', 2000)
                }
                // Attempt automatic reconnection
                if (reconnectFnRef.current) {
                  await reconnectFnRef.current(videoElementRef.current)
                }
              } else if (currentICEState === 'connected' || currentICEState === 'completed') {
                console.log('✅ Connection recovered automatically!')
                setIsConnected(true)
              }
            }, 5000) // Increased to 5 seconds for better recovery chance
            
            if (onMessage) {
              onMessage('', 'Connection interrupted, attempting to recover...', 2000)
            }
          } else if (state === 'connected' || state === 'completed') {
            console.log('✅ ICE connection established!')
            
            // Check if peer connection is also connected
            const pcState = pc.connectionState
            const fullyConnected = (pcState === 'connected' || pcState === 'connecting') &&
              (state === 'connected' || state === 'completed')
            
            setIsConnected(fullyConnected)
            reconnectAttemptsRef.current = 0 // Reset reconnect attempts on successful connection
            
            if (fullyConnected) {
              console.log('✅ Full connection established! Ready for media')
              
              // Try to play video if track is already received
              if (videoElement && videoElement.srcObject) {
                try {
                  await videoElement.play()
                  console.log('✅ Video playback initiated')
                } catch (error) {
                  console.warn('⚠️ Could not auto-play video:', error)
                }
              }
              
              if (onMessage) {
                onMessage('success', 'Connection established! Stream ready.', 3000)
              }
            } else {
              console.log('🔄 ICE connected, waiting for PeerConnection...', { pcState })
            }
          } else if (state === 'checking') {
            console.log('ICE connection checking...')
            setIsConnected(false)
            if (onMessage) {
              onMessage('', 'Establishing connection...', 0)
            }
          } else if (state === 'new') {
            console.log('ICE connection new state')
            setIsConnected(false)
          }
        }

        // Handle ICE candidates with detailed logging
        pc.onicecandidate = (event: RTCPeerConnectionIceEvent) => {
          if (event.candidate) {
            console.log('🧊 ICE candidate:', {
              candidate: event.candidate.candidate,
              type: event.candidate.type,
              protocol: event.candidate.protocol,
              priority: event.candidate.priority,
            })
            // Log candidate type for debugging
            if (event.candidate.type === 'host') {
              console.log('✅ Host candidate (good for local connections)')
            } else if (event.candidate.type === 'srflx') {
              console.log('✅ Server reflexive candidate (STUN)')
            } else if (event.candidate.type === 'relay') {
              console.log('✅ Relay candidate (TURN)')
            }
          } else {
            console.log('✅ ICE gathering complete - all candidates collected')
            console.log('Current connection states:', {
              iceGatheringState: pc.iceGatheringState,
              iceConnectionState: pc.iceConnectionState,
              connectionState: pc.connectionState,
            })
          }
        }
        
        // Monitor for connection establishment
        let connectionEstablished = false
        const checkConnectionEstablished = () => {
          const iceState = pc.iceConnectionState
          const connState = pc.connectionState
          
          if (!connectionEstablished && 
              (iceState === 'connected' || iceState === 'completed') &&
              connState === 'connected') {
            connectionEstablished = true
            console.log('🎉 Connection fully established!')
            console.log('Connection details:', {
              iceConnectionState: iceState,
              connectionState: connState,
              iceGatheringState: pc.iceGatheringState,
              localDescription: pc.localDescription?.type,
              remoteDescription: pc.remoteDescription?.type,
            })
          }
        }
        
        // Check periodically for first 10 seconds
        const connectionCheckInterval = setInterval(() => {
          checkConnectionEstablished()
          if (connectionEstablished) {
            clearInterval(connectionCheckInterval)
          }
        }, 500)
        
        setTimeout(() => {
          clearInterval(connectionCheckInterval)
          if (!connectionEstablished) {
            console.warn('⚠️ Connection not established after 10 seconds')
            console.warn('Current states:', {
              iceConnectionState: pc.iceConnectionState,
              connectionState: pc.connectionState,
              iceGatheringState: pc.iceGatheringState,
            })
          }
        }, 10000)

        // Handle ICE gathering state
        pc.onicegatheringstatechange = () => {
          console.log('ICE gathering state:', pc.iceGatheringState)
        }

        // Request to receive media - use addTransceiver with proper direction
        // This tells the peer we want to receive video
        const videoTransceiver = pc.addTransceiver('video', { direction: 'recvonly' })
        const audioTransceiver = pc.addTransceiver('audio', { direction: 'recvonly' })
        
        console.log('📺 Created transceivers:', {
          video: { mid: videoTransceiver.mid, direction: videoTransceiver.direction },
          audio: { mid: audioTransceiver.mid, direction: audioTransceiver.direction },
        })

        // Create and send offer
        const offer = await pc.createOffer({
          offerToReceiveVideo: true,
          offerToReceiveAudio: false, // We're not handling audio
        })
        await pc.setLocalDescription(offer)

        // Wait for ICE gathering to complete or timeout
        // Increased timeout for better reliability
        await new Promise<void>((resolve) => {
          if (pc.iceGatheringState === 'complete') {
            console.log('ICE gathering already complete')
            resolve()
          } else {
            const checkState = () => {
              if (pc.iceGatheringState === 'complete') {
                console.log('ICE gathering completed')
                pc.removeEventListener('icegatheringstatechange', checkState)
                resolve()
              }
            }
            pc.addEventListener('icegatheringstatechange', checkState)
            // Increased timeout to 10 seconds for more reliable connections
            setTimeout(() => {
              console.log('ICE gathering timeout - proceeding with current candidates')
              pc.removeEventListener('icegatheringstatechange', checkState)
              resolve()
            }, 10000)
          }
        })

        const answer = await sendOffer(offer)
        const answerDesc: RTCSessionDescriptionInit = {
          type: 'answer',
          sdp: answer.sdp,
        }
        
        console.log('📥 Received answer SDP, setting remote description...')
        await pc.setRemoteDescription(answerDesc)
        console.log('✅ Remote description set successfully')

        pcRef.current = pc
        
        // Wait a bit for connection to establish
        // Monitor connection progress
        console.log('⏳ Waiting for ICE connection to establish...')
        console.log('Current states:', {
          connectionState: pc.connectionState,
          iceConnectionState: pc.iceConnectionState,
          iceGatheringState: pc.iceGatheringState,
        })
        
        // Set up a promise that resolves when connection is established
        const waitForConnection = new Promise<void>((resolve) => {
          const checkConnection = () => {
            const iceState = pc.iceConnectionState
            const connState = pc.connectionState
            
            if (iceState === 'connected' || iceState === 'completed') {
              console.log('✅ ICE connection established!')
              resolve()
            } else if (iceState === 'failed' || connState === 'failed') {
              console.error('❌ Connection failed during setup')
              resolve() // Resolve anyway to continue
            } else {
              // Check again in 500ms
              setTimeout(checkConnection, 500)
            }
          }
          
          // Start checking immediately
          checkConnection()
          
          // Timeout after 15 seconds
          setTimeout(() => {
            console.warn('⚠️ Connection establishment timeout, proceeding...')
            resolve()
          }, 15000)
        })
        
        // Don't block, but wait a bit
        waitForConnection.then(() => {
          console.log('Connection establishment check complete:', {
            connectionState: pc.connectionState,
            iceConnectionState: pc.iceConnectionState,
          })
        })
        
        console.log('✅ Offer/Answer exchange complete, ICE negotiation in progress...')
        
        // Set up periodic connection health check with automatic reconnection
        healthCheckIntervalRef.current = setInterval(async () => {
          if (!pcRef.current || pcRef.current.connectionState === 'closed') {
            if (healthCheckIntervalRef.current) {
              clearInterval(healthCheckIntervalRef.current)
              healthCheckIntervalRef.current = null
            }
            return
          }
          
          const iceState = pcRef.current.iceConnectionState
          const pcState = pcRef.current.connectionState
          
          if (iceState === 'disconnected' || iceState === 'failed' || pcState === 'failed') {
            console.warn('Connection health check: connection is unhealthy', { iceState, pcState })
            
            // Only attempt reconnection if we haven't exceeded max attempts
            if (reconnectAttemptsRef.current < maxReconnectAttempts) {
              if (onMessage) {
                onMessage('', 'Connection unstable. Reconnecting...', 2000)
              }
              if (reconnectFnRef.current) {
                await reconnectFnRef.current(videoElementRef.current)
              }
            } else {
              if (onMessage) {
                onMessage('error', 'Connection lost. Please refresh the page.', 5000)
              }
            }
          } else if (iceState === 'connected' || iceState === 'completed') {
            // Connection is healthy - reset reconnect attempts
            reconnectAttemptsRef.current = 0
            console.log('Connection health check: OK')
          }
        }, 8000) // Check every 8 seconds
        
        // Clean up interval when connection closes
        pc.addEventListener('connectionstatechange', () => {
          if (pc.connectionState === 'closed' || pc.connectionState === 'failed') {
            if (healthCheckIntervalRef.current) {
              clearInterval(healthCheckIntervalRef.current)
              healthCheckIntervalRef.current = null
            }
          }
        })
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

  // Reconnect function (must be defined after startConnection)
  const reconnect = useCallback(
    async (videoElement: HTMLVideoElement | null) => {
      if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
        console.error('Max reconnect attempts reached')
        if (onMessage) {
          onMessage('error', 'Connection failed after multiple attempts. Please refresh the page.', 5000)
        }
        reconnectAttemptsRef.current = 0
        return
      }

      reconnectAttemptsRef.current++
      console.log(`Reconnection attempt ${reconnectAttemptsRef.current}/${maxReconnectAttempts}`)
      
      // Close existing connection
      if (pcRef.current) {
        try {
          pcRef.current.close()
        } catch (e) {
          console.warn('Error closing old connection:', e)
        }
        pcRef.current = null
      }

      // Clear all timeouts
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
      if (iceRestartTimeoutRef.current) {
        clearTimeout(iceRestartTimeoutRef.current)
        iceRestartTimeoutRef.current = null
      }
      if (healthCheckIntervalRef.current) {
        clearInterval(healthCheckIntervalRef.current)
        healthCheckIntervalRef.current = null
      }

      // Wait a bit before reconnecting
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      // Start new connection
      try {
        await startConnection(videoElement)
        reconnectAttemptsRef.current = 0 // Reset on success
      } catch (error) {
        console.error('Reconnection failed:', error)
        if (onMessage) {
          onMessage('error', `Reconnection failed (${reconnectAttemptsRef.current}/${maxReconnectAttempts})`, 3000)
        }
      }
    },
    [startConnection, onMessage]
  )

  // Store reconnect function in ref for use in startConnection
  useEffect(() => {
    reconnectFnRef.current = reconnect
  }, [reconnect])

  const stopConnection = useCallback(
    (videoElement: HTMLVideoElement | null, onMessage?: MessageCallback) => {
      // Clear any pending timeouts and intervals
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
      if (iceRestartTimeoutRef.current) {
        clearTimeout(iceRestartTimeoutRef.current)
        iceRestartTimeoutRef.current = null
      }
      if (healthCheckIntervalRef.current) {
        clearInterval(healthCheckIntervalRef.current)
        healthCheckIntervalRef.current = null
      }
      
      if (pcRef.current) {
        // Close connection gracefully
        pcRef.current.close()
        pcRef.current = null
      }
      if (videoElement) {
        videoElement.srcObject = null
      }
      setIsConnected(false)
      setIsConnecting(false)
      if (onMessage) onMessage('success', 'Stream stopped')
    },
    []
  )

  return {
    isConnected,
    isConnecting,
    startConnection,
    stopConnection,
    reconnect,
  }
}
