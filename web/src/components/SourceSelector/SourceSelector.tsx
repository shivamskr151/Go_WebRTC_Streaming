import React from 'react'
import type { SourceInfo } from '../../types/api'

interface SourceSelectorProps {
  sourceInfo: SourceInfo
  onSwitchSource: (sourceType: string) => void
  onRefresh: () => void
}

export const SourceSelector: React.FC<SourceSelectorProps> = ({ sourceInfo, onSwitchSource, onRefresh }) => {
  const getCurrentSourceText = (): string => {
    if (sourceInfo.running && sourceInfo.type) {
      return `Current: ${sourceInfo.type.toUpperCase()} (Running)`
    }
    if (sourceInfo.type) {
      return `Current: ${sourceInfo.type.toUpperCase()} (Stopped)`
    }
    return 'No active source'
  }

  const getAvailableSourcesText = (): string => {
    if (sourceInfo.available.length > 0) {
      return `Available: ${sourceInfo.available.join(', ').toUpperCase()}`
    }
    return 'No sources available'
  }

  return (
    <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-xl p-3 border border-gray-200 shadow-lg flex-shrink-0">
      <div className="flex items-center justify-center gap-1.5 mb-2">
        <span className="text-lg">🎯</span>
        <h3 className="text-gray-800 text-sm font-bold">Source Selection</h3>
      </div>
      <div className="flex gap-2 justify-center flex-wrap mb-2">
        <button
          className="px-3 py-1.5 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide bg-gradient-to-r from-emerald-500 to-teal-500 text-white hover:from-emerald-600 hover:to-teal-600 hover:scale-105 hover:shadow-lg hover:shadow-emerald-500/40 active:scale-95 flex items-center justify-center gap-1"
          onClick={() => onSwitchSource('rtsp')}
        >
          <span className="text-xs">📹</span>
          <span>RTSP</span>
        </button>
        <button
          className="px-3 py-1.5 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide bg-gradient-to-r from-orange-500 to-amber-500 text-white hover:from-orange-600 hover:to-amber-600 hover:scale-105 hover:shadow-lg hover:shadow-orange-500/40 active:scale-95 flex items-center justify-center gap-1"
          onClick={() => onSwitchSource('rtmp')}
        >
          <span className="text-xs">📺</span>
          <span>RTMP</span>
        </button>
        <button 
          className="px-3 py-1.5 rounded-lg text-xs font-bold cursor-pointer transition-all duration-300 uppercase tracking-wide bg-gradient-to-r from-blue-500 to-indigo-500 text-white hover:from-blue-600 hover:to-indigo-600 hover:scale-105 hover:shadow-lg hover:shadow-blue-500/40 active:scale-95 flex items-center justify-center gap-1" 
          onClick={onRefresh}
        >
          <span className="text-xs">🔄</span>
          <span>Refresh</span>
        </button>
      </div>
      <div className="text-center p-2 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="flex items-center justify-center gap-1.5 mb-1">
          <span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></span>
          <span className="font-semibold text-gray-800 text-xs">{getCurrentSourceText()}</span>
        </div>
        <small className="text-gray-600 font-medium text-xs">{getAvailableSourcesText()}</small>
      </div>
    </div>
  )
}
