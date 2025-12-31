/**
 * Connection status
 */
export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected'

/**
 * Video source type
 */
export type VideoSource = 'camera' | 'screen'

/**
 * Publish configuration
 */
export interface PublishConfig {
  source: VideoSource
  trackName: string
  width?: number
  height?: number
  frameRate?: number
}

/**
 * Subscribe configuration
 */
export interface SubscribeConfig {
  trackName: string
}

/**
 * Connection info for MoQ
 */
export interface ConnectionInfo {
  status: ConnectionStatus
  error?: Error
}
