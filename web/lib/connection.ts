/**
 * MoQ Connection Management
 */

import { Client, type Session, TrackMux } from "@okdaichi/moq";
import type { AppState, ConnectionStatus } from "./types.ts";
import { CONFIG } from "./config.ts";

export class MoQConnection {
  private state: AppState;
  private onStatusChange?: (status: ConnectionStatus) => void;

  constructor(state: AppState, onStatusChange?: (status: ConnectionStatus) => void) {
    this.state = state;
    this.onStatusChange = onStatusChange;
  }

  async connect(url: string): Promise<void> {
    this.updateStatus("connecting");

    try {
      console.log(`🔌 Connecting to MoQ relay: ${url}`);

      // Create MoQ client and track mux
      // Using system CA (mkcert) for local development
      this.state.client = new Client({
        transportOptions: {
          allowPooling: false,
          congestionControl: "low-latency",
          requireUnreliable: true,
        },
      });

      this.state.mux = new TrackMux();

      console.log("📡 Attempting to dial relay...");
      
      // Connect to relay
      this.state.session = await this.state.client.dial(url, this.state.mux);
      console.log("📡 Session created, waiting for ready...");
      
      await this.state.session.ready;

      this.state.connected = true;
      this.updateStatus("connected");
      console.log("✅ MoQ connection established");
    } catch (error) {
      console.error("❌ Connection failed:", error);
      this.updateStatus("disconnected");
      throw error;
    }
  }

  async disconnect(): Promise<void> {
    console.log("🔌 Disconnecting from relay...");

    try {
      if (this.state.session) {
        await this.state.session.close();
        this.state.session = null;
      }
      if (this.state.client) {
        await this.state.client.close();
        this.state.client = null;
      }
      this.state.mux = null;
    } catch (error) {
      console.error("Error closing connection:", error);
    }

    this.state.connected = false;
    this.updateStatus("disconnected");
    console.log("✅ Disconnected");
  }

  isConnected(): boolean {
    return this.state.connected;
  }

  getSession(): Session | null {
    return this.state.session;
  }

  getMux(): TrackMux | null {
    return this.state.mux;
  }

  private updateStatus(status: ConnectionStatus): void {
    if (this.onStatusChange) {
      this.onStatusChange(status);
    }
  }
}
