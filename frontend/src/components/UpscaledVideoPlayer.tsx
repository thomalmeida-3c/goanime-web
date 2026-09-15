"use client";

import { useEffect, useRef, useState } from "react";
import {
  createUpscalePipeline,
  scaleFactorFor,
  supportsWebGL2,
  type UpscalePipeline,
} from "@/lib/upscaleGL";

function formatTime(seconds: number): string {
  if (!isFinite(seconds)) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

// requestVideoFrameCallback isn't in the default DOM lib types yet — feature
// detected at runtime, typed just enough to call safely.
type VideoWithVFC = HTMLVideoElement & {
  requestVideoFrameCallback?: (cb: () => void) => number;
  cancelVideoFrameCallback?: (handle: number) => void;
};

// Renders the video through a WebGL2 upscale+sharpen shader (see
// lib/upscaleGL.ts) onto a canvas. The real <video> stays mounted (it's the
// actual decode/audio source) but invisible — which means native controls
// disappear with it, so this ships its own play/seek/mute/fullscreen bar.
// Falls back to a plain <video controls> when WebGL2 isn't available.
export default function UpscaledVideoPlayer({ src }: { src: string }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const pipelineRef = useRef<UpscalePipeline | null>(null);
  const rafRef = useRef<number | null>(null);
  const vfcRef = useRef<number | null>(null);

  const [supported, setSupported] = useState<boolean | null>(null);
  const [upscaleEnabled, setUpscaleEnabled] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);
  const [buffering, setBuffering] = useState(true);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [muted, setMuted] = useState(false);

  useEffect(() => {
    setSupported(supportsWebGL2());
  }, []);

  useEffect(() => {
    if (!supported || !canvasRef.current) return;
    const pipeline = createUpscalePipeline(canvasRef.current);
    pipelineRef.current = pipeline;
    return () => {
      pipeline?.dispose();
      pipelineRef.current = null;
    };
  }, [supported]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !supported) return;

    function onLoadedMetadata() {
      if (!video || !pipelineRef.current) return;
      const factor = scaleFactorFor(video.videoWidth);
      pipelineRef.current.setSize(
        Math.round(video.videoWidth * factor),
        Math.round(video.videoHeight * factor),
      );
      setDuration(video.duration);
    }
    video.addEventListener("loadedmetadata", onLoadedMetadata);
    return () => video.removeEventListener("loadedmetadata", onLoadedMetadata);
  }, [supported]);

  useEffect(() => {
    if (!supported) return;
    const video = videoRef.current as VideoWithVFC | null;
    if (!video) return;

    function draw() {
      pipelineRef.current?.render(video!, upscaleEnabled ? 0.6 : 0.0);
    }

    if (video.requestVideoFrameCallback) {
      const loop = () => {
        draw();
        vfcRef.current = video.requestVideoFrameCallback!(loop);
      };
      vfcRef.current = video.requestVideoFrameCallback(loop);
      return () => {
        if (vfcRef.current !== null) video.cancelVideoFrameCallback?.(vfcRef.current);
      };
    }

    const loop = () => {
      draw();
      rafRef.current = requestAnimationFrame(loop);
    };
    rafRef.current = requestAnimationFrame(loop);
    return () => {
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current);
    };
  }, [supported, upscaleEnabled]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    const onPlay = () => setIsPlaying(true);
    const onPause = () => setIsPlaying(false);
    const onWaiting = () => setBuffering(true);
    const onPlaying = () => setBuffering(false);
    const onTimeUpdate = () => setCurrentTime(video.currentTime);
    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("waiting", onWaiting);
    video.addEventListener("playing", onPlaying);
    video.addEventListener("timeupdate", onTimeUpdate);
    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("waiting", onWaiting);
      video.removeEventListener("playing", onPlaying);
      video.removeEventListener("timeupdate", onTimeUpdate);
    };
    // supported starts out null/false, and the fallback branch below renders
    // a <video> with no ref — videoRef.current is null on that first commit.
    // Depending on `supported` re-runs this once the real, ref'd <video>
    // (the main return) actually exists, instead of attaching to nothing and
    // never trying again.
  }, [supported]);

  function togglePlay() {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) video.play().catch(() => {});
    else video.pause();
  }

  function seek(e: React.ChangeEvent<HTMLInputElement>) {
    const video = videoRef.current;
    if (!video) return;
    video.currentTime = Number(e.target.value);
    setCurrentTime(video.currentTime);
  }

  function toggleMute() {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !video.muted;
    setMuted(video.muted);
  }

  function toggleFullscreen() {
    const wrapper = wrapperRef.current;
    if (!wrapper) return;
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      wrapper.requestFullscreen().catch(() => {});
    }
  }

  // Still detecting support, or no WebGL2: plain native player.
  if (supported !== true) {
    return (
      <video controls autoPlay className="w-full rounded-lg bg-black" src={src}>
        Seu navegador não suporta vídeo HTML5.
      </video>
    );
  }

  return (
    <div ref={wrapperRef} className="relative w-full overflow-hidden rounded-lg bg-black">
      <video
        ref={videoRef}
        src={src}
        autoPlay
        playsInline
        crossOrigin="anonymous"
        aria-hidden
        className="pointer-events-none absolute inset-0 h-full w-full opacity-0"
      />
      <canvas ref={canvasRef} className="block w-full" />

      {buffering && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/30">
          <div className="h-10 w-10 animate-spin rounded-full border-2 border-white/30 border-t-white" />
        </div>
      )}

      <div className="absolute inset-x-0 bottom-0 flex flex-col gap-1.5 bg-gradient-to-t from-black/90 to-transparent px-4 pb-3 pt-8">
        <input
          type="range"
          min={0}
          max={duration || 0}
          step={0.1}
          value={currentTime}
          onChange={seek}
          className="h-1 w-full cursor-pointer accent-orange-500"
        />
        <div className="flex items-center gap-3 text-xs text-neutral-200">
          <button onClick={togglePlay} className="text-lg leading-none">
            {isPlaying ? "⏸" : "▶"}
          </button>
          <button onClick={toggleMute} className="text-base leading-none">
            {muted ? "🔇" : "🔊"}
          </button>
          <span className="tabular-nums">
            {formatTime(currentTime)} / {formatTime(duration)}
          </span>

          <span className="flex-1" />

          <label className="flex select-none items-center gap-1.5">
            <input
              type="checkbox"
              checked={upscaleEnabled}
              onChange={(e) => setUpscaleEnabled(e.target.checked)}
            />
            Upscaling
          </label>

          <button onClick={toggleFullscreen} className="text-base leading-none">
            ⛶
          </button>
        </div>
      </div>
    </div>
  );
}
