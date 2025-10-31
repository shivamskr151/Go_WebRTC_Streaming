import React, { useState, useEffect } from 'react'

interface SnapshotViewerProps {
  snapshot: string | null
  onClear?: () => void
}

export const SnapshotViewer: React.FC<SnapshotViewerProps> = ({ snapshot, onClear }) => {
  const [isExpanded, setIsExpanded] = useState(false)

  // Close modal on ESC key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isExpanded) {
        setIsExpanded(false)
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isExpanded])

  // Prevent body scroll when modal is open
  useEffect(() => {
    if (isExpanded) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }
    return () => {
      document.body.style.overflow = 'unset'
    }
  }, [isExpanded])

  if (!snapshot) {
    return (
      <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-xl p-3 border border-gray-200 shadow-lg h-full flex items-center justify-center">
        <p className="text-gray-400 text-xs font-medium">No snapshot captured</p>
      </div>
    )
  }

  return (
    <>
      <div className="bg-gradient-to-br from-gray-50 to-gray-100 rounded-xl p-3 border border-gray-200 shadow-lg h-full flex flex-col overflow-hidden">
        <div className="flex justify-between items-center mb-2">
          <div className="flex items-center gap-1.5">
            <span className="text-lg">📸</span>
            <h3 className="text-gray-800 text-sm font-bold">Captured Snapshot</h3>
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
        <div 
          className="relative rounded-lg overflow-hidden border-2 border-white shadow-lg w-full flex-1 min-h-0 cursor-pointer transition-transform duration-300 hover:scale-105"
          onClick={() => setIsExpanded(true)}
        >
          <img 
            src={snapshot} 
            alt="Snapshot" 
            className="w-full h-full object-contain pointer-events-none"
          />
          <div className="absolute inset-0 bg-black/0 hover:bg-black/10 transition-colors duration-300 flex items-center justify-center">
            <span className="opacity-0 hover:opacity-100 text-white text-xs font-semibold bg-black/50 px-2 py-1 rounded transition-opacity">
              Click to expand
            </span>
          </div>
        </div>
      </div>

      {/* Expanded Modal */}
      {isExpanded && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm animate-fade-in"
          onClick={() => setIsExpanded(false)}
        >
          <div className="relative max-w-[95vw] max-h-[95vh] p-4">
            <button
              className="absolute top-2 right-2 bg-gradient-to-r from-red-500 to-rose-500 text-white border-none rounded-full w-10 h-10 text-xl leading-none cursor-pointer flex items-center justify-center transition-all duration-300 hover:from-red-600 hover:to-rose-600 hover:scale-110 hover:shadow-lg shadow-red-200/50 active:scale-95 z-10"
              onClick={(e) => {
                e.stopPropagation()
                setIsExpanded(false)
              }}
              aria-label="Close expanded view"
            >
              ×
            </button>
            <img
              src={snapshot}
              alt="Expanded Snapshot"
              className="max-w-full max-h-[95vh] object-contain rounded-lg shadow-2xl"
              onClick={(e) => e.stopPropagation()}
            />
            <div className="absolute bottom-4 left-1/2 transform -translate-x-1/2 bg-black/70 text-white px-4 py-2 rounded-lg text-sm">
              Press ESC or click outside to close
            </div>
          </div>
        </div>
      )}
    </>
  )
}
