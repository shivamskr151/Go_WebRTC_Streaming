import React from 'react'

interface SnapshotViewerProps {
  snapshot: string | null
  onClear?: () => void
}

export const SnapshotViewer: React.FC<SnapshotViewerProps> = ({ snapshot, onClear }) => {
  if (!snapshot) {
    return null
  }

  return (
    <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-xl p-2 border border-gray-200 shadow-lg flex-shrink-0">
      <div className="flex justify-between items-center mb-2">
        <div className="flex items-center gap-1.5">
          <span className="text-sm">📸</span>
          <h3 className="text-gray-800 text-xs font-bold">Snapshot</h3>
        </div>
        {onClear && (
          <button
            className="bg-gradient-to-r from-red-500 to-rose-500 text-white border-none rounded-full w-6 h-6 text-xs leading-none cursor-pointer flex items-center justify-center transition-all duration-300 hover:from-red-600 hover:to-rose-600 hover:scale-110 hover:shadow-md active:scale-95"
            onClick={onClear}
            aria-label="Close snapshot"
          >
            ×
          </button>
        )}
      </div>
      <div className="relative rounded-lg overflow-hidden border-2 border-white shadow-lg max-h-32">
        <img 
          src={snapshot} 
          alt="Snapshot" 
          className="w-full h-full object-cover block"
        />
      </div>
    </div>
  )
}
