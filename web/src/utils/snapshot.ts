/**
 * Utility functions for snapshot capture
 */

/**
 * Capture snapshot from video element
 */
export const captureVideoSnapshot = (videoElement: HTMLVideoElement): string => {
  if (!videoElement || !videoElement.srcObject) {
    throw new Error('No video stream available. Please start the stream first.')
  }

  if (videoElement.paused || videoElement.ended) {
    throw new Error('Video is not playing. Please ensure the stream is active.')
  }

  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')

  if (!ctx) {
    throw new Error('Failed to get canvas context')
  }

  const videoWidth = videoElement.videoWidth || videoElement.clientWidth
  const videoHeight = videoElement.videoHeight || videoElement.clientHeight

  if (videoWidth === 0 || videoHeight === 0) {
    throw new Error('Video dimensions not available. Please wait for the video to load.')
  }

  canvas.width = videoWidth
  canvas.height = videoHeight
  ctx.drawImage(videoElement, 0, 0, canvas.width, canvas.height)

  return canvas.toDataURL('image/jpeg', 0.9)
}
