import type { GroupSequence } from "@okdaichi/moq";

export interface VideoMetadata extends VideoDecoderConfig {
	startGroup: GroupSequence;
}
export interface AudioMetadata extends AudioDecoderConfig {
	startGroup: GroupSequence;
}
