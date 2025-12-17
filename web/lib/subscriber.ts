/**
 * MoQ Subscriber - Frame Reception & Display
 */

import { background } from "@okdaichi/golikejs/context";
import type { AppState, SubscribeOptions } from "./types.ts";
import { BroadcastPath, Frame } from "@okdaichi/moq";

export class MoQSubscriber {
  private state: AppState;
  private videoElement: HTMLVideoElement;

  constructor(state: AppState, videoElement: HTMLVideoElement) {
    this.state = state;
    this.videoElement = videoElement;
  }

  async start(options: SubscribeOptions): Promise<void> {
    try {
      const broadcastPath = `/live/${options.trackName}` as BroadcastPath;
      console.log(`📥 Subscribing to: ${broadcastPath}`);

      if (!this.state.session) throw new Error("Session not established");

      // Subscribe to track
      const [trackReader, subscribeErr] = await this.state.session.subscribe(
        broadcastPath,
        "video", // Track name within broadcast path
      );
      if (subscribeErr) {
        throw new Error(`Subscribe failed: ${subscribeErr}`);
      }

      console.log("✅ Subscribed, waiting for frames...");

      this.state.subscribeAbort = false;
      this.state.subscribing = true;

      // Start receiving loop
      (async () => {
        try {
          while (!this.state.subscribeAbort && this.state.subscribing) {
            // Accept next group
            const ctx = background().done();
            const [groupReader, groupErr] = await trackReader.acceptGroup(ctx);
            if (groupErr) {
              console.error("Failed to accept group:", groupErr);
              break;
            }

            // Read frames from group
            let frameCount = 0;
            const frame = new Frame(new Uint8Array());
            while (true) {
              const frameErr = await groupReader.readFrame(frame);
              if (frameErr) {
                if (frameErr.message.includes("EOF")) {
                  // End of group, normal
                  break;
                }
                console.error("Failed to read frame:", frameErr);
                break;
              }

              if (!frame) break;

              // Decode and display frame
              this.displayFrame(frame);
              frameCount++;
            }

            if (frameCount > 0 && frameCount % 30 === 0) {
              console.log(`📊 Received ${frameCount} frames`);
            }
          }

          console.log("✅ Subscription loop ended");
        } catch (error) {
          console.error("❌ Subscription error:", error);
        }
      })();

      console.log("✅ Subscription started");
    } catch (error) {
      console.error("❌ Failed to subscribe:", error);
      throw error;
    }
  }

  stop(): void {
    console.log("🛑 Stopping subscription...");

    // Signal abort
    this.state.subscribeAbort = true;

    // Clear video
    this.videoElement.srcObject = null;

    this.state.subscribing = false;
    console.log("✅ Subscription stopped");
  }

  isSubscribing(): boolean {
    return this.state.subscribing;
  }

  private displayFrame(frameData: Frame): void {
    const blob = new Blob([new Uint8Array(frameData.data)], { type: "image/jpeg" });
    const imageUrl = URL.createObjectURL(blob);

    // Update video element with image
    const img = new Image();
    img.onload = () => {
      if (!this.videoElement.paused) {
        const canvas = document.createElement("canvas");
        canvas.width = img.width;
        canvas.height = img.height;
        const ctx = canvas.getContext("2d");
        ctx?.drawImage(img, 0, 0);

        // Use canvas stream for video element
        if (!this.state.subscribing) return;
        const canvasWithStream = canvas as HTMLCanvasElement & {
          captureStream(frameRate?: number): MediaStream;
        };
        const stream = canvasWithStream.captureStream(30);
        this.videoElement.srcObject = stream;
        this.videoElement.play().catch(() => {});
      }
      URL.revokeObjectURL(imageUrl);
    };
    img.src = imageUrl;
  }
}
