import React, { forwardRef, useState, useEffect } from 'react'

interface VideoPlayerProps extends React.VideoHTMLAttributes<HTMLVideoElement> {
  isConnected?: boolean
  isConnecting?: boolean
}

export const VideoPlayer = forwardRef<HTMLVideoElement, VideoPlayerProps>(({ isConnected, isConnecting, ...props }, ref) => {
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)
  const videoRef = ref as React.MutableRefObject<HTMLVideoElement | null>

  useEffect(() => {
    const video = videoRef.current
    if (video) {
      const handleCanPlay = () => {
        setIsLoading(false)
        setHasError(false)
        console.log('✅ Video can play - stream ready')
      }
      const handleLoadedMetadata = () => {
        console.log('✅ Video metadata loaded')
        setIsLoading(false)
      }
      const handleLoadedData = () => {
        console.log('✅ Video data loaded')
      }
      const handleLoadStart = () => {
        setIsLoading(true)
        setHasError(false)
        console.log('🔄 Video load started')
      }
      const handlePlay = () => {
        setIsLoading(false)
        setHasError(false)
        console.log('▶️ Video playing - stream active!')
      }
      const handlePlaying = () => {
        setIsLoading(false)
        setHasError(false)
        console.log('▶️ Video is playing')
      }
      const handlePause = () => {
        console.log('⏸️ Video paused')
      }
      const handleError = (e: Event) => {
        console.error('❌ Video error:', e, video.error)
        setIsLoading(false)
        setHasError(true)
      }
      const handleWaiting = () => {
        setIsLoading(true)
        console.log('⏳ Video waiting for data')
      }
      const handleStalled = () => {
        console.warn('⚠️ Video stalled')
      }
      
      // Monitor srcObject changes
      const checkSrcObject = () => {
        if (video.srcObject) {
          console.log('✅ Video srcObject set:', video.srcObject)
          const stream = video.srcObject as MediaStream
          if (stream.getVideoTracks().length > 0) {
            const track = stream.getVideoTracks()[0]
            console.log('✅ Video track found:', {
              id: track.id,
              label: track.label,
              enabled: track.enabled,
              readyState: track.readyState,
              kind: track.kind,
            })
            track.onended = () => {
              console.warn('⚠️ Video track ended')
              setIsLoading(true)
            }
            track.onmute = () => {
              console.warn('⚠️ Video track muted')
            }
            track.onunmute = () => {
              console.log('✅ Video track unmuted')
            }
          } else {
            console.warn('⚠️ No video tracks in stream')
          }
        } else {
          console.log('ℹ️ Video srcObject cleared')
        }
      }
      
      // Initial check
      checkSrcObject()
      
      // Set up observer for srcObject changes
      const observer = new MutationObserver(() => {
        checkSrcObject()
      })

      video.addEventListener('canplay', handleCanPlay)
      video.addEventListener('loadedmetadata', handleLoadedMetadata)
      video.addEventListener('loadeddata', handleLoadedData)
      video.addEventListener('loadstart', handleLoadStart)
      video.addEventListener('play', handlePlay)
      video.addEventListener('playing', handlePlaying)
      video.addEventListener('pause', handlePause)
      video.addEventListener('error', handleError)
      video.addEventListener('waiting', handleWaiting)
      video.addEventListener('stalled', handleStalled)

      // Observe video element attributes
      observer.observe(video, {
        attributes: true,
        attributeFilter: ['src', 'srcObject'],
      })

      return () => {
        video.removeEventListener('canplay', handleCanPlay)
        video.removeEventListener('loadedmetadata', handleLoadedMetadata)
        video.removeEventListener('loadeddata', handleLoadedData)
        video.removeEventListener('loadstart', handleLoadStart)
        video.removeEventListener('play', handlePlay)
        video.removeEventListener('playing', handlePlaying)
        video.removeEventListener('pause', handlePause)
        video.removeEventListener('error', handleError)
        video.removeEventListener('waiting', handleWaiting)
        video.removeEventListener('stalled', handleStalled)
        observer.disconnect()
      }
    }
  }, [videoRef])

  // Reset loading state when connection state changes
  useEffect(() => {
    if (!isConnected && !isConnecting) {
      // Don't clear loading if we have a stream
      if (!videoRef.current?.srcObject) {
        setIsLoading(false)
        setHasError(false)
      }
    } else if (isConnecting) {
      setIsLoading(true)
      setHasError(false)
    } else if (isConnected && videoRef.current?.srcObject) {
      // Connection is established and we have a stream
      // Video events will handle loading state
      console.log('✅ Connection established with video stream')
    }
  }, [isConnected, isConnecting])
  
  // Monitor stream state
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const checkStream = () => {
      if (video.srcObject) {
        const stream = video.srcObject as MediaStream
        const tracks = stream.getVideoTracks()
        if (tracks.length > 0) {
          const track = tracks[0]
          console.log('📹 Video track state:', {
            enabled: track.enabled,
            readyState: track.readyState,
            muted: track.muted,
          })
          
          if (track.readyState === 'ended') {
            console.warn('⚠️ Video track has ended')
            setIsLoading(true)
          }
        }
      }
    }

    // Check periodically
    const interval = setInterval(checkStream, 2000)
    checkStream() // Initial check

    return () => clearInterval(interval)
  }, [isConnected])

  const showLoading = (isLoading || isConnecting) && !hasError

  return (
    <div className="relative bg-gradient-to-br from-gray-900 via-gray-800 to-black rounded-xl overflow-hidden h-full shadow-xl border-2 border-white/10">
      <video
        ref={ref}
        id="videoElement"
        className="w-full h-full object-cover"
        autoPlay
        muted
        playsInline
        {...props}
      />
      {showLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-900/80 backdrop-blur-sm z-10">
          <div className="text-center space-y-2">
            <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
            <p className="text-white text-sm font-medium">
              {isConnecting ? 'Connecting to stream...' : 'Loading video stream...'}
            </p>
          </div>
        </div>
      )}
      {hasError && !isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-900/80 backdrop-blur-sm z-10">
          <div className="text-center space-y-2">
            <p className="text-red-400 text-sm font-medium">Stream error</p>
            <p className="text-gray-400 text-xs">Try starting the stream again</p>
          </div>
        </div>
      )}
      {!isConnected && !isConnecting && !hasError && !videoRef.current?.srcObject && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-900/60 backdrop-blur-sm z-10">
          <div className="text-center space-y-2">
            <div className="w-16 h-16 mx-auto mb-4 flex items-center justify-center">
              <svg className="w-full h-full text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
            </div>
            <p className="text-white text-sm font-medium">Ready to stream</p>
            <p className="text-gray-400 text-xs">Click "Start" to begin WebRTC streaming</p>
          </div>
        </div>
      )}
    </div>
  )
})

VideoPlayer.displayName = 'VideoPlayer'
