import React from 'react'

export const Header: React.FC = () => {
  return (
    <div className="text-center mb-3 space-y-1 flex-shrink-0">
      <div className="flex items-center justify-center gap-2">
        <div className="text-2xl md:text-3xl animate-pulse">🎥</div>
        <h1 className="text-gray-800 text-xl md:text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent">
          Go WebRTC Streaming
        </h1>
      </div>
    </div>
  )
}
