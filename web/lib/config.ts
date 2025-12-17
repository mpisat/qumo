/**
 * Centralized configuration for qumo web demo
 */

// Port configuration (must match docker-compose.yml and config files)
export const PORTS = {
  RELAY: 4433,        // QUIC/MoQT relay server
  HEALTH: 8080,       // Health check HTTP endpoint
  WEB_DEV: 5173,      // Vite dev server
} as const;

// Auto-detect relay URL based on environment
export function getDefaultRelayUrl(): string {
  // If accessing via specific hostname/IP, use that
  if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
    return `https://${window.location.hostname}:${PORTS.RELAY}`;
  }
  
  // For localhost development, try to detect WSL IP from web server
  // If web server is running on WSL IP, use that for relay too
  const currentHost = window.location.hostname;
  if (currentHost.match(/^172\./) || currentHost.match(/^192\.168\./) || currentHost.match(/^10\./)) {
    return `https://${currentHost}:${PORTS.RELAY}`;
  }
  
  // For Windows with WSL, use common WSL IP range
  // User should access via WSL IP (e.g., http://10.237.238.211:5173)
  return `https://localhost:${PORTS.RELAY}`;
}

// Certificate hash for self-signed certificate (development only)
// Required when using certificate hash pinning for Docker/self-signed certs
export const DEV_CERT_HASH = "d4a9fb715a0a1eefebd5f4f264f6b42814771cb1e0f0003c9583dc32ca2b098d";

export const CONFIG = {
  defaultRelayUrl: getDefaultRelayUrl(),
  defaultTrackName: "my-stream",
  certHashHex: DEV_CERT_HASH,
} as const;
