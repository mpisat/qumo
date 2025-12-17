/**
 * DOM utility functions
 */

export const $ = <T extends HTMLElement>(id: string): T => {
  const el = document.getElementById(id);
  if (!el) throw new Error(`Element #${id} not found`);
  return el as T;
};

export interface DOMElements {
  connectBtn: HTMLButtonElement;
  connectionStatus: HTMLDivElement;
  startPublishBtn: HTMLButtonElement;
  stopPublishBtn: HTMLButtonElement;
  subscribeBtn: HTMLButtonElement;
  unsubscribeBtn: HTMLButtonElement;
  localVideo: HTMLVideoElement;
  remoteVideo: HTMLVideoElement;
  relayUrl: HTMLInputElement;
  trackName: HTMLInputElement;
  subscribeTrackName: HTMLInputElement;
  sourceType: HTMLSelectElement;
}

export const getElements = (): DOMElements => ({
  connectBtn: $<HTMLButtonElement>("connectBtn"),
  connectionStatus: $<HTMLDivElement>("connectionStatus"),
  startPublishBtn: $<HTMLButtonElement>("startPublishBtn"),
  stopPublishBtn: $<HTMLButtonElement>("stopPublishBtn"),
  subscribeBtn: $<HTMLButtonElement>("subscribeBtn"),
  unsubscribeBtn: $<HTMLButtonElement>("unsubscribeBtn"),
  localVideo: $<HTMLVideoElement>("localVideo"),
  remoteVideo: $<HTMLVideoElement>("remoteVideo"),
  relayUrl: $<HTMLInputElement>("relayUrl"),
  trackName: $<HTMLInputElement>("trackName"),
  subscribeTrackName: $<HTMLInputElement>("subscribeTrackName"),
  sourceType: $<HTMLSelectElement>("sourceType"),
});
