/**
 * API service for backend communication
 */

import type {
  StatusResponse,
  SourceInfo,
  OfferResponse,
  SwitchSourceResponse,
  SnapshotResponse,
  ErrorResponse,
} from '../types/api'

const API_BASE = '/api'

/**
 * Send WebRTC offer to server
 */
export const sendOffer = async (offer: RTCSessionDescriptionInit): Promise<OfferResponse> => {
  const response = await fetch(`${API_BASE}/offer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sdp: offer }),
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  return response.json()
}

/**
 * Get system status
 */
export const getStatus = async (): Promise<StatusResponse> => {
  const response = await fetch(`${API_BASE}/status`)
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return response.json()
}

/**
 * Get source information
 */
export const getSourceInfo = async (): Promise<SourceInfo> => {
  const response = await fetch(`${API_BASE}/source`)
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return response.json()
}

/**
 * Switch video source
 */
export const switchSource = async (sourceType: string): Promise<SwitchSourceResponse> => {
  const response = await fetch(`${API_BASE}/source`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ type: sourceType }),
  })

  if (!response.ok) {
    const errorData: ErrorResponse = await response.json()
    throw new Error(errorData.error || `HTTP error! status: ${response.status}`)
  }

  return response.json()
}

/**
 * Capture snapshot
 */
export const captureSnapshot = async (): Promise<SnapshotResponse> => {
  const response = await fetch(`${API_BASE}/snapshot`)
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  return response.json()
}
