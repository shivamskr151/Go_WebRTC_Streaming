import { useState, useEffect, useCallback } from 'react'
import { getStatus, getSourceInfo } from '../services/api'
import type { StatusResponse, SourceInfo } from '../types/api'

interface UseStatusReturn {
  status: StatusResponse
  sourceInfo: SourceInfo
  loading: boolean
  updateStatus: () => Promise<void>
  updateSourceInfo: () => Promise<void>
  refreshAll: () => Promise<void>
}

/**
 * Custom hook for managing system status
 */
export const useStatus = (refreshInterval: number = 5000): UseStatusReturn => {
  const [status, setStatus] = useState<StatusResponse>({
    webrtc: { connected_peers: 0, total_peers: 0 },
    source: { type: '', running: false, available: [] },
    streams: { rtmp: false, rtsp: false },
  })
  const [sourceInfo, setSourceInfo] = useState<SourceInfo>({
    type: '',
    running: false,
    available: [],
  })
  const [loading, setLoading] = useState<boolean>(false)

  const updateStatus = useCallback(async (): Promise<void> => {
    try {
      setLoading(true)
      const data = await getStatus()
      setStatus(data)
    } catch (error) {
      console.error('Error updating status:', error)
    } finally {
      setLoading(false)
    }
  }, [])

  const updateSourceInfo = useCallback(async (): Promise<void> => {
    try {
      const data = await getSourceInfo()
      setSourceInfo(data)
    } catch (error) {
      console.error('Error updating source info:', error)
    }
  }, [])

  const refreshAll = useCallback(async (): Promise<void> => {
    await Promise.all([updateStatus(), updateSourceInfo()])
  }, [updateStatus, updateSourceInfo])

  useEffect(() => {
    refreshAll()
    const interval = setInterval(updateStatus, refreshInterval)
    return () => clearInterval(interval)
  }, [refreshAll, updateStatus, refreshInterval])

  return {
    status,
    sourceInfo,
    loading,
    updateStatus,
    updateSourceInfo,
    refreshAll,
  }
}
