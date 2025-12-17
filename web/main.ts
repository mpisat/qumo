/**
 * qumo Web Demo - MoQ Relay Testing Interface
 * Built with Deno 2.x and modern Web APIs
 */

console.log("🚀 qumo Web Demo starting...");

import type { AppState, ConnectionStatus } from "./lib/types.ts";
import { MoQConnection } from "./lib/connection.ts";
import { MoQPublisher } from "./lib/publisher.ts";
import { MoQSubscriber } from "./lib/subscriber.ts";
import { type DOMElements, getElements } from "./lib/dom.ts";
import { CONFIG } from "./lib/config.ts";

// Initialize DOM elements
const elements: DOMElements = getElements();

// Set default relay URL
elements.relayUrl.value = CONFIG.defaultRelayUrl;
console.log(`📡 Default relay URL: ${CONFIG.defaultRelayUrl}`);

// Application state
const state: AppState = {
  connected: false,
  publishing: false,
  subscribing: false,
  localStream: null,
  client: null,
  session: null,
  mux: null,
  publishAbort: false,
  subscribeAbort: false,
};

// Initialize modules
const connection = new MoQConnection(state, (status: ConnectionStatus) => {
  updateConnectionStatus(status);
});
const publisher = new MoQPublisher(state, elements.localVideo);
const subscriber = new MoQSubscriber(state, elements.remoteVideo);

// UI Update Functions
const updateConnectionStatus = (status: ConnectionStatus): void => {
  const statusMessages = {
    connecting: "Connecting...",
    connected: `✅ Connected to ${elements.relayUrl.value}`,
    disconnected: "Disconnected",
  };

  elements.connectionStatus.textContent = statusMessages[status];
  elements.connectionStatus.className = `status ${status}`;
};

const updateUI = (): void => {
  elements.startPublishBtn.disabled = !state.connected || state.publishing;
  elements.stopPublishBtn.disabled = !state.publishing;
  elements.subscribeBtn.disabled = !state.connected || state.subscribing;
  elements.unsubscribeBtn.disabled = !state.subscribing;
  elements.connectBtn.textContent = state.connected ? "Disconnect" : "Connect to Relay";
};

// Event Handlers
elements.connectBtn.addEventListener("click", async () => {
  try {
    if (connection.isConnected()) {
      // Stop all active operations first
      if (publisher.isPublishing()) publisher.stop();
      if (subscriber.isSubscribing()) subscriber.stop();

      await connection.disconnect();
    } else {
      await connection.connect(elements.relayUrl.value);
    }
    updateUI();
  } catch (error) {
    console.error("Connection error:", error);
    alert(
      `Connection failed: ${error instanceof Error ? error.message : "Unknown error"}`,
    );
  }
});

elements.startPublishBtn.addEventListener("click", async () => {
  try {
    await publisher.start({
      sourceType: elements.sourceType.value as "camera" | "screen",
      trackName: elements.trackName.value,
      width: elements.sourceType.value === "screen" ? 1920 : 1280,
      height: elements.sourceType.value === "screen" ? 1080 : 720,
      frameRate: 30,
    });
    updateUI();
  } catch (error) {
    console.error("Publishing error:", error);
    alert(
      `Failed to start publishing: ${error instanceof Error ? error.message : "Unknown error"}`,
    );
  }
});

elements.stopPublishBtn.addEventListener("click", () => {
  try {
    publisher.stop();
    updateUI();
  } catch (error) {
    console.error("Stop publishing error:", error);
  }
});

elements.subscribeBtn.addEventListener("click", async () => {
  try {
    await subscriber.start({
      trackName: elements.subscribeTrackName.value,
    });
    updateUI();
  } catch (error) {
    console.error("Subscription error:", error);
    alert(
      `Failed to subscribe: ${error instanceof Error ? error.message : "Unknown error"}`,
    );
  }
});

elements.unsubscribeBtn.addEventListener("click", () => {
  try {
    subscriber.stop();
    updateUI();
  } catch (error) {
    console.error("Unsubscribe error:", error);
  }
});

// Initialize UI
updateUI();
console.log("✅ Web Demo initialized");
console.log("📝 Ready to connect to relay");
