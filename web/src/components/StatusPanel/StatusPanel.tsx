import React from 'react'
import type { StatusResponse } from '../../types/api'

interface StatusPanelProps {
  isConnected: boolean
  status: StatusResponse
}

export const StatusPanel: React.FC<StatusPanelProps> = ({ isConnected, status }) => {
  const getStatusText = (isActive: boolean): string => {
    return isActive ? 'Connected' : 'Disconnected'
  }

  const getSourceStatusText = (): string => {
    if (status.source.running) {
      return `${status.source.type.toUpperCase()} - Running`
    }
    return 'Disconnected'
  }

  return (
    <div className="bg-gradient-to-br from-slate-50 via-gray-50 to-slate-50 rounded-xl p-3 border border-gray-200 shadow-lg flex-shrink-0">
      <div className="flex items-center justify-center gap-1.5 mb-3">
        <span className="text-lg">📊</span>
        <h3 className="text-gray-800 text-sm font-bold">System Status</h3>
      </div>
      <div className="space-y-2">
        <div className="flex justify-between items-center p-2 bg-white rounded-lg border border-gray-100 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex items-center gap-2">
            <span className="text-sm">🔌</span>
            <span className="font-semibold text-gray-700 text-xs">WebRTC</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></span>
            <span className={`px-2 py-1 rounded text-xs font-bold ${
              isConnected 
                ? 'bg-green-100 text-green-700 border border-green-200' 
                : 'bg-red-100 text-red-700 border border-red-200'
            }`}>
              {getStatusText(isConnected)}
            </span>
          </div>
        </div>
        <div className="flex justify-between items-center p-2 bg-white rounded-lg border border-gray-100 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex items-center gap-2">
            <span className="text-sm">📹</span>
            <span className="font-semibold text-gray-700 text-xs">Video</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className={`w-1.5 h-1.5 rounded-full ${status.source.running ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></span>
            <span className={`px-2 py-1 rounded text-xs font-bold ${
              status.source.running 
                ? 'bg-green-100 text-green-700 border border-green-200' 
                : 'bg-red-100 text-red-700 border border-red-200'
            }`}>
              {getSourceStatusText()}
            </span>
          </div>
        </div>
        <div className="flex justify-between items-center p-2 bg-white rounded-lg border border-gray-100 shadow-sm hover:shadow-md transition-shadow">
          <div className="flex items-center gap-2">
            <span className="text-sm">👥</span>
            <span className="font-semibold text-gray-700 text-xs">Peers</span>
          </div>
          <span className="px-2 py-1 rounded text-xs font-bold bg-blue-100 text-blue-700 border border-blue-200">
            {status.webrtc.connected_peers}/{status.webrtc.total_peers}
          </span>
        </div>
      </div>
    </div>
  )
}
