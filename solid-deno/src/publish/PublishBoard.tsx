import { createEffect, createSignal, onMount, Show } from "solid-js";
import { type BroadcastPath, GroupWriter, TrackMux } from "@okdaichi/moq";
import {
	AudioEncodeNode,
	MediaStreamVideoSourceNode,
	VideoContext,
	VideoEncodeNode,
	videoEncoderConfig,
} from "@okdaichi/av-nodes";
import { getMediaStream, type MediaSourceType } from "./media.ts";
import { background, type CancelFunc, type Context, withCancel } from "@okdaichi/golikejs/context";
import { MediaFrame } from "./media_frame.ts";
import type { AudioMetadata, VideoMetadata } from "../metadata/mod.ts";
import { useBroadcastPath } from "../useBroadcastPath.ts";

const GOP_DURATION = 1000; // 1 second

export function PublishBoard(props: { mux: TrackMux }) {
	const broadcastPath: BroadcastPath = useBroadcastPath();
	const mux = props.mux;

	const [sourceType, setSourceType] = createSignal<MediaSourceType>("camera");
	const [isStreaming, setIsStreaming] = createSignal(false);
	const [error, setError] = createSignal<string | null>(null);
	const [canvasWidth, setCanvasWidth] = createSignal(1280);
	const [canvasHeight, setCanvasHeight] = createSignal(720);

	let canvasEle: HTMLCanvasElement | undefined;
	let lastKeyframeTime = 0;
	let videoContext: VideoContext | undefined;
	let sourceNode: MediaStreamVideoSourceNode | null = null;
	let videoEncodeNode: VideoEncodeNode | undefined;
	let audioEncodeNode: AudioEncodeNode | undefined;

	onMount(() => {
		if (canvasEle) {
			videoContext = new VideoContext({ canvas: canvasEle });
			videoEncodeNode = new VideoEncodeNode(videoContext, {
				isKey: (timestamp, _) => {
					// timestamp is in microseconds, so convert GOP_DURATION to microseconds
					if (timestamp - lastKeyframeTime >= GOP_DURATION * 1000) {
						lastKeyframeTime = timestamp;
						return true;
					}
					return false;
				},
			});

			// VideoContextのcanvasサイズから初期値を取得
			setCanvasWidth(videoContext.destination.canvas.width);
			setCanvasHeight(videoContext.destination.canvas.height);
		}
	});

	// Canvasサイズが変更されたら自動で再configure
	createEffect(async () => {
		const width = canvasWidth();
		const height = canvasHeight();

		if (videoEncodeNode && width > 0 && height > 0) {
			const videoConfig = await videoEncoderConfig({
				width,
				height,
				bitrate: 2_500_000,
				frameRate: 30,
				tryHardware: true,
			});
			videoEncodeNode.configure(videoConfig);
			console.log(`Video encoder reconfigured to ${width}x${height}`);
		}
	});

	let publishCtx: Context;
	let cancelPublish: CancelFunc;

	const startStreaming = async () => {
		[publishCtx, cancelPublish] = withCancel(background());

		try {
			setError(null);

			if (!videoContext || !videoEncodeNode) {
				throw new Error("Video context not initialized");
			}

			const stream = await getMediaStream(sourceType());

			// Create and configure source node
			sourceNode = new MediaStreamVideoSourceNode(videoContext, { mediaStream: stream });
			sourceNode.connect(videoContext.destination);
			sourceNode.connect(videoEncodeNode);
			sourceNode.start();

			setIsStreaming(true);
			console.log(`Started streaming from ${sourceType()}`);
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : String(err);
			setError(errorMessage);
			console.error("Failed to start streaming:", err);
		}

		// Video metadata
		const videoMetaStream = new TransformStream<VideoMetadata>(); // TODO: specify type
		const videoMetaWriter = videoMetaStream.writable.getWriter();
		// const videoMetaReader = videoMetaStream.readable.getReader();
		let videoMeta: VideoMetadata | undefined;

		// Audio metadata
		const audioMetaStream = new TransformStream<AudioMetadata>(); // TODO: specify type
		const audioMetaWriter = audioMetaStream.writable.getWriter();
		// const audioMetaReader = audioMetaStream.readable.getReader();
		let audioMeta: AudioMetadata | undefined;

		// Publish
		mux.publishFunc(
			publishCtx.done(),
			broadcastPath,
			async (track) => {
				console.log("[publishFunc] Track handler called for:", track.trackName);
				switch (track.trackName) {
					case "video": {
						console.log("[publishFunc] Starting video track processing");
						if (!videoEncodeNode) {
							throw new Error("Encode node not initialized");
						}

						let currentGroup: GroupWriter | undefined = undefined;

						// Pass the track as the VideoEncodeDestination
						const { done } = videoEncodeNode.encodeTo({
							output: async (
								chunk: EncodedVideoChunk,
								decoderConfig?: VideoDecoderConfig,
							) => {
								switch (chunk.type) {
									case "key": {
										if (currentGroup) {
											void currentGroup.close();
										}
										const [group, err] = await track.openGroup();
										if (err) {
											return err;
										}
										currentGroup = group;

										if (decoderConfig) {
											videoMeta = {
												...decoderConfig,
												startGroup: currentGroup.sequence,
											};
											await videoMetaWriter.write(videoMeta);
										}
										break;
									}
									case "delta": {
										if (!currentGroup) {
											// Drop delta frames until we get a keyframe
											return;
										}

										break;
									}
								}

								const frame = new MediaFrame(chunk);

								const err = await currentGroup.writeFrame(frame);
								if (err) {
									throw err;
								}
							},
						});

						await done;
						break;
					}
					case "audio": {
						if (!audioEncodeNode) {
							throw new Error("Audio encode node not initialized");
						}

						const { done } = audioEncodeNode.encodeTo({
							output: async (
								chunk: EncodedAudioChunk,
								decoderConfig?: AudioDecoderConfig,
							) => {
								const [group, err] = await track.openGroup();
								if (err) {
									return err;
								}

								if (decoderConfig) {
									audioMeta = { ...decoderConfig, startGroup: group.sequence };
									await audioMetaWriter.write(audioMeta);
								}

								const writeErr = await group.writeFrame(new MediaFrame(chunk));
								if (writeErr) {
									// TODO: handle error
								}

								void group.close();
							},
						});

						await done;
						break;
					}
					default: {
						console.log("[publishFunc] Unknown track:", track.trackName, "- ignoring");
						return;
					}
				}
			},
		);
	};

	const stopStreaming = () => {
		cancelPublish();
		if (sourceNode) {
			sourceNode.stop();
			sourceNode.dispose();
			sourceNode = null;
		}
		setIsStreaming(false);
		console.log("Stopped streaming");
	};

	return (
		<div class="publish-board">
			<h2>Publish Board</h2>

			<div class="controls">
				<div class="source-selector">
					<label for="source-type">Media Source:</label>
					<select
						id="source-type"
						value={sourceType()}
						onChange={(e) => setSourceType(e.currentTarget.value as MediaSourceType)}
						disabled={isStreaming()}
					>
						<option value="camera">Camera</option>
						<option value="screen">Screen Share</option>
					</select>
				</div>

				<div class="stream-controls">
					<Show
						when={!isStreaming()}
						fallback={
							<button type="button" onClick={stopStreaming} class="btn-stop">
								Stop Streaming
							</button>
						}
					>
						<button type="button" onClick={startStreaming} class="btn-start">
							Start Streaming
						</button>
					</Show>
				</div>
			</div>

			<Show when={error()}>
				<div class="error-message">
					Error: {error()}
				</div>
			</Show>

			<Show when={isStreaming()}>
				<div class="status-message">
					Streaming from: {sourceType()}
				</div>
			</Show>

			<div class="video-preview">
				<canvas
					ref={canvasEle}
					width={canvasWidth()}
					height={canvasHeight()}
					style={{
						width: "100%",
						"max-width": "800px",
						height: "auto",
						border: "1px solid #ccc",
						"border-radius": "8px",
						background: "#000",
					}}
				/>
			</div>
		</div>
	);
}
