import React, { forwardRef } from 'react'

interface VideoPlayerProps extends React.VideoHTMLAttributes<HTMLVideoElement> {}

export const VideoPlayer = forwardRef<HTMLVideoElement, VideoPlayerProps>((props, ref) => {
  return (
    <div className="relative bg-black rounded-2xl overflow-hidden mb-8 aspect-video">
      <video
        ref={ref}
        id="videoElement"
        className="w-full h-full object-cover"
        autoPlay
        muted
        playsInline
        {...props}
      />
    </div>
  )
})

VideoPlayer.displayName = 'VideoPlayer'
