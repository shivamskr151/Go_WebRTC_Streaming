import React, { useState, useRef, useCallback } from 'react'
import { useWebRTC } from './hooks/useWebRTC'
import { useStatus } from './hooks/useStatus'
import { switchSource } from './services/api'
import { captureVideoSnapshot } from './utils/snapshot'
import {
  Header,
  VideoPlayer,
  StreamControls,
  SourceSelector,
  StatusPanel,
  MessageToast,
  SnapshotViewer,
} from './components'

interface Message {
  type: 'success' | 'error' | ''
  text: string
}

const App: React.FC = () => {
  const [message, setMessage] = useState<Message>({ type: '', text: '' })
  const [snapshot, setSnapshot] = useState<string | null>(null)
  const videoRef = useRef<HTMLVideoElement>(null)

  const showMessage = useCallback((type: 'success' | 'error' | '', text: string, duration: number = 3000) => {
    setMessage({ type, text })
    if (duration > 0) {
      setTimeout(() => setMessage({ type: '', text: '' }), duration)
    }
  }, [])

  const { status, sourceInfo, updateStatus, updateSourceInfo, refreshAll } = useStatus()

  const handleTrackReceived = useCallback(() => {
    // Track received, status will be updated via WebRTC hook
  }, [])

  const { isConnected, isConnecting, startConnection, stopConnection } = useWebRTC(
    handleTrackReceived,
    showMessage
  )

  const handleStartStream = useCallback(async () => {
    try {
      await startConnection(videoRef.current)
      await updateStatus()
    } catch (error) {
      // Error handling is done in the hook
    }
  }, [startConnection, updateStatus])

  const handleStopStream = useCallback(() => {
    stopConnection(videoRef.current, showMessage)
    updateStatus()
  }, [stopConnection, showMessage, updateStatus])

  const handleCaptureSnapshot = useCallback(() => {
    try {
      if (!videoRef.current) {
        throw new Error('Video element not available')
      }
      const dataURL = captureVideoSnapshot(videoRef.current)
      setSnapshot(dataURL)
      showMessage('success', 'Snapshot captured from live video stream!')
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      showMessage('error', `Failed to capture snapshot: ${errorMessage}`)
    }
  }, [showMessage])

  const handleSwitchSource = useCallback(
    async (sourceType: string) => {
      try {
        showMessage('', '', 0)
        await switchSource(sourceType)
        showMessage('success', `Switched to ${sourceType.toUpperCase()} source!`)
        await refreshAll()
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error'
        showMessage('error', `Failed to switch to ${sourceType}: ${errorMessage}`)
      }
    },
    [showMessage, refreshAll]
  )

  return (
    <div className="bg-white/95 backdrop-blur-sm rounded-3xl shadow-2xl p-4 md:p-6 max-w-[95vw] w-full border border-white/20 animate-fade-in h-[95vh] max-h-[95vh] flex flex-col overflow-hidden">
      <Header />
      <div className="flex-1 grid grid-cols-1 lg:grid-cols-[1.2fr_0.8fr] gap-4 overflow-hidden min-h-0">
        {/* Left Column - Video Player & Controls */}
        <div className="flex flex-col gap-3 min-h-0 overflow-hidden">
          <div className="flex-1 min-h-0">
            <VideoPlayer ref={videoRef} />
          </div>
          <StreamControls
            isConnected={isConnected}
            isConnecting={isConnecting}
            onStart={handleStartStream}
            onStop={handleStopStream}
            onCaptureSnapshot={handleCaptureSnapshot}
            onRefreshStatus={updateStatus}
          />
          <SnapshotViewer snapshot={snapshot} onClear={() => setSnapshot(null)} />
        </div>
        
        {/* Right Column - Status & Source Selection */}
        <div className="flex flex-col gap-3 min-h-0 overflow-y-auto">
          <StatusPanel isConnected={isConnected} status={status} />
          <SourceSelector
            sourceInfo={sourceInfo}
            onSwitchSource={handleSwitchSource}
            onRefresh={updateSourceInfo}
          />
        </div>
      </div>
      <MessageToast message={message} />
    </div>
  )
}

export default App
