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
    <div className="bg-gray-50 rounded-2xl p-5 mb-5">
      <h3 className="text-gray-800 mb-4 text-xl font-semibold">📊 System Status</h3>
      <div className="flex justify-between items-center py-2 border-b border-gray-200 last:border-b-0">
        <span className="font-semibold text-gray-700">WebRTC Connection:</span>
        <span className={`px-3 py-1 rounded-full text-sm font-semibold ${
          isConnected 
            ? 'bg-green-100 text-green-800' 
            : 'bg-red-100 text-red-800'
        }`}>
          {getStatusText(isConnected)}
        </span>
      </div>
      <div className="flex justify-between items-center py-2 border-b border-gray-200 last:border-b-0">
        <span className="font-semibold text-gray-700">Video Source:</span>
        <span className={`px-3 py-1 rounded-full text-sm font-semibold ${
          status.source.running 
            ? 'bg-green-100 text-green-800' 
            : 'bg-red-100 text-red-800'
        }`}>
          {getSourceStatusText()}
        </span>
      </div>
      <div className="flex justify-between items-center py-2 border-b border-gray-200 last:border-b-0">
        <span className="font-semibold text-gray-700">Connected Peers:</span>
        <span className="px-3 py-1 rounded-full text-sm font-semibold text-gray-700">
          {status.webrtc.connected_peers}/{status.webrtc.total_peers}
        </span>
      </div>
    </div>
  )
}
