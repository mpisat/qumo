/**
 * Type definitions for MoQ web demo
 */

import type { Client, Session, TrackMux } from "@okdaichi/moq";

export interface AppState {
  connected: boolean;
  publishing: boolean;
  subscribing: boolean;
  localStream: MediaStream | null;
  client: Client | null;
  session: Session | null;
  mux: TrackMux | null;
  publishAbort: boolean;
  subscribeAbort: boolean;
}

export interface PublishOptions {
  sourceType: "camera" | "screen";
  trackName: string;
  width: number;
  height: number;
  frameRate: number;
}

export interface SubscribeOptions {
  trackName: string;
}

export type ConnectionStatus = "connecting" | "connected" | "disconnected";
