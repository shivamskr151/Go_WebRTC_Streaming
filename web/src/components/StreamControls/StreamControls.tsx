import React from 'react'

interface StreamControlsProps {
  isConnected: boolean
  isConnecting: boolean
  onStart: () => void
  onStop: () => void
  onCaptureSnapshot: () => void
  onRefreshStatus: () => void
}

export const StreamControls: React.FC<StreamControlsProps> = ({
  isConnected,
  isConnecting,
  onStart,
  onStop,
  onCaptureSnapshot,
  onRefreshStatus,
}) => {
  return (
    <div className="flex gap-4 mb-8 flex-wrap justify-center md:flex-row flex-col">
      <button
        className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-purple-500 to-purple-700 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-purple-500/30 w-full md:w-auto"
        onClick={onStart}
        disabled={isConnecting || isConnected}
      >
        {isConnecting ? 'Connecting...' : 'Start Stream'}
      </button>
      <button
        className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-pink-400 to-red-500 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-pink-500/30 w-full md:w-auto"
        onClick={onStop}
        disabled={!isConnected}
      >
        Stop Stream
      </button>
      <button
        className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-blue-400 to-cyan-400 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-blue-500/30 w-full md:w-auto"
        onClick={onCaptureSnapshot}
        disabled={!isConnected}
      >
        Capture Snapshot
      </button>
      <button 
        className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-purple-500 to-purple-700 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-purple-500/30 w-full md:w-auto" 
        onClick={onRefreshStatus}
      >
        Refresh Status
      </button>
    </div>
  )
}
