/**
 * API response types
 */

export interface StatusResponse {
  webrtc: {
    connected_peers: number
    total_peers: number
  }
  source: {
    type: string
    running: boolean
    available: string[]
  }
  streams: {
    rtmp: boolean
    rtsp: boolean
  }
}

export interface SourceInfo {
  type: string
  running: boolean
  available: string[]
}

export interface OfferRequest {
  sdp: RTCSessionDescriptionInit
}

export interface OfferResponse {
  sdp: string
}

export interface SwitchSourceRequest {
  type: string
}

export interface SwitchSourceResponse {
  success: boolean
  message: string
  type: string
}

export interface SnapshotResponse {
  success: boolean
  data?: string
  error?: string
}

export interface ErrorResponse {
  error: string
  available?: string[]
}
