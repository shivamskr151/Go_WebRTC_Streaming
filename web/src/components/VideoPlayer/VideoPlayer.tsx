import React, { forwardRef, useState, useEffect } from 'react'

interface VideoPlayerProps extends React.VideoHTMLAttributes<HTMLVideoElement> {}

export const VideoPlayer = forwardRef<HTMLVideoElement, VideoPlayerProps>((props, ref) => {
  const [isLoading, setIsLoading] = useState(true)
  const videoRef = ref as React.MutableRefObject<HTMLVideoElement | null>

  useEffect(() => {
    const video = videoRef.current
    if (video) {
      const handleCanPlay = () => setIsLoading(false)
      const handleLoadStart = () => setIsLoading(true)
      video.addEventListener('canplay', handleCanPlay)
      video.addEventListener('loadstart', handleLoadStart)
      return () => {
        video.removeEventListener('canplay', handleCanPlay)
        video.removeEventListener('loadstart', handleLoadStart)
      }
    }
  }, [videoRef])

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
      {isLoading && !videoRef.current?.srcObject && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-900/80 backdrop-blur-sm">
          <div className="text-center space-y-2">
            <div className="w-12 h-12 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
            <p className="text-white text-sm font-medium">Loading video stream...</p>
          </div>
        </div>
      )}
    </div>
  )
})

VideoPlayer.displayName = 'VideoPlayer'
