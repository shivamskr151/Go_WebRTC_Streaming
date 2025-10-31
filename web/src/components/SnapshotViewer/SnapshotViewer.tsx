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
    <div className="text-center mt-5">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-gray-800 text-xl font-semibold m-0">📸 Captured Snapshot</h3>
        {onClear && (
          <button
            className="bg-red-600 text-white border-none rounded-full w-8 h-8 text-xl leading-none cursor-pointer flex items-center justify-center transition-colors duration-300 hover:bg-red-700"
            onClick={onClear}
          >
            ×
          </button>
        )}
      </div>
      <img 
        src={snapshot} 
        alt="Snapshot" 
        className="max-w-full rounded-xl shadow-lg mt-4"
      />
    </div>
  )
}
