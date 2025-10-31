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
    <div className="mb-8">
      <h3 className="text-center mb-4 text-gray-800 text-xl font-semibold">🎯 Source Selection</h3>
      <div className="flex gap-4 justify-center flex-wrap mb-4 md:flex-row flex-col">
        <button
          className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-purple-500 to-purple-700 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-purple-500/30 w-full md:w-auto"
          onClick={() => onSwitchSource('rtsp')}
        >
          Switch to RTSP
        </button>
        <button
          className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-purple-500 to-purple-700 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-purple-500/30 w-full md:w-auto"
          onClick={() => onSwitchSource('rtmp')}
        >
          Switch to RTMP
        </button>
        <button 
          className="px-6 py-3 rounded-full text-base font-semibold cursor-pointer transition-all duration-300 uppercase tracking-wide disabled:opacity-60 disabled:cursor-not-allowed disabled:transform-none bg-gradient-to-r from-blue-400 to-cyan-400 text-white hover:translate-y-[-2px] hover:shadow-lg hover:shadow-blue-500/30 w-full md:w-auto" 
          onClick={onRefresh}
        >
          Refresh Source Info
        </button>
      </div>
      <div className="text-center mt-4 p-3 bg-gray-50 rounded-lg">
        <span className="block">{getCurrentSourceText()}</span>
        <small className="text-gray-600">{getAvailableSourcesText()}</small>
      </div>
    </div>
  )
}
