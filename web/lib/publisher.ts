/**
 * MoQ Publisher - Video/Screen Capture & Publishing
 */

import { background } from "@okdaichi/golikejs/context";
import { BroadcastPath, Frame, type TrackWriter } from "@okdaichi/moq";
import type { AppState, PublishOptions } from "./types.ts";

export class MoQPublisher {
  private state: AppState;
  private videoElement: HTMLVideoElement;
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;

  constructor(state: AppState, videoElement: HTMLVideoElement) {
    this.state = state;
    this.videoElement = videoElement;
    this.canvas = document.createElement("canvas");
    const ctx = this.canvas.getContext("2d");
    if (!ctx) throw new Error("Failed to get canvas context");
    this.ctx = ctx;
  }

  async start(options: PublishOptions): Promise<void> {
    try {
      console.log(`🎥 Requesting ${options.sourceType} access...`);

      // Get media stream based on source type
      const stream = await this.getMediaStream(options);
      this.state.localStream = stream;
      this.videoElement.srcObject = stream;
      await this.videoElement.play();

      const broadcastPath = `/live/${options.trackName}` as BroadcastPath;
      console.log(`📤 Publishing to: ${broadcastPath}`);

      // Register publish handler with MoQ
      if (!this.state.mux) throw new Error("TrackMux not initialized");

      this.state.publishAbort = false;
      const publishCtx = background().done();

      // Start publishing loop
      this.state.mux.publishFunc(
        publishCtx,
        broadcastPath,
        async (tw: TrackWriter) => {
          await this.publishLoop(tw, stream, options);
        },
      );

      this.state.publishing = true;
      console.log("✅ Publishing started");
    } catch (error) {
      console.error("❌ Failed to start publishing:", error);
      throw error;
    }
  }

  stop(): void {
    console.log("🛑 Stopping publish...");

    // Signal abort
    this.state.publishAbort = true;

    // Stop media tracks
    if (this.state.localStream) {
      this.state.localStream.getTracks().forEach((track) => {
        track.stop();
        console.log(`  Stopped ${track.kind} track`);
      });
      this.videoElement.srcObject = null;
      this.state.localStream = null;
    }

    this.state.publishing = false;
    console.log("✅ Publishing stopped");
  }

  isPublishing(): boolean {
    return this.state.publishing;
  }

  private async getMediaStream(options: PublishOptions): Promise<MediaStream> {
    if (options.sourceType === "screen") {
      const mediaDevices = navigator.mediaDevices as typeof navigator.mediaDevices & {
        getDisplayMedia(constraints?: MediaStreamConstraints): Promise<MediaStream>;
      };
      const stream = await mediaDevices.getDisplayMedia({
        video: {
          width: { ideal: options.width },
          height: { ideal: options.height },
          frameRate: { ideal: options.frameRate },
        },
        audio: false,
      });
      console.log("📺 Screen capture started");
      return stream;
    } else {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: { ideal: options.width },
          height: { ideal: options.height },
          frameRate: { ideal: options.frameRate },
        },
        audio: false,
      });
      console.log("📹 Camera started");
      return stream;
    }
  }

  private async publishLoop(
    tw: TrackWriter,
    stream: MediaStream,
    options: PublishOptions,
  ): Promise<void> {
    console.log("🎬 Publisher handler called, streaming video frames...");

    try {
      // Setup canvas dimensions
      const videoTrack = stream.getVideoTracks()[0];
      const settings = videoTrack.getSettings();
      this.canvas.width = settings.width || options.width;
      this.canvas.height = settings.height || options.height;

      let groupSeq = 0n;
      const frameInterval = 1000 / options.frameRate;
      let lastFrameTime = performance.now();

      // Publishing loop
      while (!this.state.publishAbort && this.state.publishing) {
        const now = performance.now();
        if (now - lastFrameTime < frameInterval) {
          await new Promise((resolve) =>
            setTimeout(resolve, frameInterval - (now - lastFrameTime))
          );
          continue;
        }
        lastFrameTime = now;

        // Capture video frame to canvas
        this.ctx.drawImage(
          this.videoElement,
          0,
          0,
          this.canvas.width,
          this.canvas.height,
        );

        // Get frame data as JPEG (better compression than PNG)
        // Quality set to 0.6 to keep frames under 64KB limit
        const blob = await new Promise<Blob>((resolve) => {
          this.canvas.toBlob((b) => resolve(b!), "image/jpeg", 0.6);
        });
        const frameData = new Uint8Array(await blob.arrayBuffer());

        // Open new group for this frame
        const [group, groupErr] = await tw.openGroup();
        if (groupErr) {
          console.error("Failed to open group:", groupErr);
          break;
        }

        // Write frame to group
        const frame = new Frame(frameData);
        const writeErr = await group.writeFrame(frame);
        if (writeErr) {
          console.error("Failed to write frame:", writeErr);
          await group.close();
          break;
        }

        await group.close();
        groupSeq++;

        if (groupSeq % 30n === 0n) {
          console.log(`📊 Published ${groupSeq} frames`);
        }
      }

      console.log("✅ Publishing loop ended");
    } catch (error) {
      console.error("❌ Publishing error:", error);
      await tw.closeWithError(1); // Generic error code
    }
  }
}
