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
    <div className="flex gap-2 flex-wrap justify-center flex-shrink-0">
      <button
        className="group relative px-4 py-2 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-purple-600 via-purple-500 to-purple-600 text-white hover:from-purple-700 hover:via-purple-600 hover:to-purple-700 hover:scale-105 hover:shadow-lg hover:shadow-purple-500/50 active:scale-95 flex items-center justify-center gap-1.5"
        onClick={onStart}
        disabled={isConnecting || isConnected}
      >
        {isConnecting ? (
          <>
            <span className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
            <span>Connecting...</span>
          </>
        ) : (
          <>
            <span className="text-sm">▶</span>
            <span>Start</span>
          </>
        )}
      </button>
      <button
        className="group relative px-4 py-2 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-rose-500 via-pink-500 to-rose-500 text-white hover:from-rose-600 hover:via-pink-600 hover:to-rose-600 hover:scale-105 hover:shadow-lg hover:shadow-rose-500/50 active:scale-95 flex items-center justify-center gap-1.5"
        onClick={onStop}
        disabled={!isConnected}
      >
        <span className="text-sm">⏹</span>
        <span>Stop</span>
      </button>
      <button
        className="group relative px-4 py-2 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-cyan-500 via-blue-500 to-cyan-500 text-white hover:from-cyan-600 hover:via-blue-600 hover:to-cyan-600 hover:scale-105 hover:shadow-lg hover:shadow-cyan-500/50 active:scale-95 flex items-center justify-center gap-1.5"
        onClick={onCaptureSnapshot}
        disabled={!isConnected}
      >
        <span className="text-sm">📷</span>
        <span>Snapshot</span>
      </button>
      <button 
        className="group relative px-4 py-2 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-indigo-500 via-purple-500 to-indigo-500 text-white hover:from-indigo-600 hover:via-purple-600 hover:to-indigo-600 hover:scale-105 hover:shadow-lg hover:shadow-indigo-500/50 active:scale-95 flex items-center justify-center gap-1.5"
        onClick={onRefreshStatus}
      >
        <span className="text-sm">🔄</span>
        <span>Refresh</span>
      </button>
    </div>
  )
}
