"use client";

import { useEffect, useState } from "react";
import { TransformComponent, TransformWrapper } from "react-zoom-pan-pinch";

// Paged (lateral) reader: one page at a time, navigate left/right — as
// opposed to MangaDex's own web reader, which defaults to an infinite
// vertical strip. Simpler to build too: fixed page list, no scroll-position
// tracking or lazy-loading needed.
export default function MangaReader({ pages }: { pages: string[] }) {
  const [index, setIndex] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [scale, setScale] = useState(1);
  const zoomed = scale > 1.02;

  useEffect(() => {
    setLoaded(false);
    setScale(1);
  }, [index]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "ArrowLeft") goTo(index - 1);
      if (e.key === "ArrowRight") goTo(index + 1);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [index, pages.length]);

  function goTo(i: number) {
    if (i < 0 || i >= pages.length) return;
    setIndex(i);
  }

  if (pages.length === 0) {
    return <p className="text-sm text-neutral-400">Nenhuma página encontrada para este capítulo.</p>;
  }

  return (
    <div className="flex h-full flex-col items-center sm:h-auto">
      <div className="h-1 w-full shrink-0 overflow-hidden bg-neutral-800 sm:mb-3 sm:max-w-3xl sm:rounded-full">
        <div
          className="h-full bg-purple-500 transition-all"
          style={{ width: `${((index + 1) / pages.length) * 100}%` }}
        />
      </div>

      <div className="relative flex min-h-0 w-full flex-1 items-center justify-center overflow-hidden bg-neutral-900 sm:max-w-3xl sm:flex-none sm:rounded-lg">
        {!loaded && (
          <div className="absolute inset-0 z-10 flex items-center justify-center">
            <div className="h-10 w-10 animate-spin rounded-full border-2 border-white/30 border-t-white" />
          </div>
        )}

        {/* key resets pan/zoom on every page turn */}
        <TransformWrapper
          key={pages[index]}
          initialScale={1}
          minScale={1}
          maxScale={4}
          centerOnInit
          doubleClick={{ mode: "toggle", step: 1.4 }}
          panning={{ disabled: !zoomed }}
          onTransform={(_, state) => setScale(state.scale)}
        >
          <TransformComponent
            wrapperClass="!h-full !w-full"
            contentClass="!h-full !w-full !flex !items-center !justify-center"
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={pages[index]}
              alt={`Página ${index + 1}`}
              onLoad={() => setLoaded(true)}
              className="h-full w-full select-none object-contain sm:h-auto sm:w-auto sm:max-h-[85vh]"
              draggable={false}
            />
          </TransformComponent>
        </TransformWrapper>

        {/* Narrow edge strips, not full halves — a wide tap zone would sit on
            top of the zoom layer and eat every pinch/double-tap before it
            ever reaches react-zoom-pan-pinch. The middle ~70% is left clear
            for zoom gestures; once zoomed, edges hand off to panning too. */}
        <button
          aria-label="Página anterior"
          onClick={() => goTo(index - 1)}
          className={`absolute inset-y-0 left-0 w-[15%] cursor-pointer ${zoomed ? "pointer-events-none" : ""}`}
        />
        <button
          aria-label="Próxima página"
          onClick={() => goTo(index + 1)}
          className={`absolute inset-y-0 right-0 w-[15%] cursor-pointer ${zoomed ? "pointer-events-none" : ""}`}
        />

        {pages[index + 1] && (
          // Warm the browser cache so the next page appears instantly.
          // eslint-disable-next-line @next/next/no-img-element
          <img src={pages[index + 1]} alt="" className="hidden" aria-hidden />
        )}
      </div>

      <div className="flex shrink-0 items-center gap-4 py-3 sm:mt-4 sm:py-0">
        <button
          onClick={() => goTo(index - 1)}
          disabled={index === 0}
          className="rounded-md border border-neutral-800 bg-neutral-900 px-4 py-2 text-sm hover:border-purple-500 hover:text-purple-300 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-neutral-800 disabled:hover:text-inherit"
        >
          ← Anterior
        </button>
        <span className="tabular-nums text-sm text-neutral-400">
          {index + 1} / {pages.length}
        </span>
        <button
          onClick={() => goTo(index + 1)}
          disabled={index === pages.length - 1}
          className="rounded-md border border-neutral-800 bg-neutral-900 px-4 py-2 text-sm hover:border-purple-500 hover:text-purple-300 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-neutral-800 disabled:hover:text-inherit"
        >
          Próxima →
        </button>
      </div>
    </div>
  );
}
