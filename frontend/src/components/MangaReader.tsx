"use client";

import { useEffect, useState } from "react";

// Paged (lateral) reader: one page at a time, navigate left/right — as
// opposed to MangaDex's own web reader, which defaults to an infinite
// vertical strip. Simpler to build too: fixed page list, no scroll-position
// tracking or lazy-loading needed.
export default function MangaReader({ pages }: { pages: string[] }) {
  const [index, setIndex] = useState(0);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    setLoaded(false);
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
    <div className="flex flex-col items-center">
      <div className="mb-3 h-1 w-full max-w-3xl overflow-hidden rounded-full bg-neutral-800">
        <div
          className="h-full bg-purple-500 transition-all"
          style={{ width: `${((index + 1) / pages.length) * 100}%` }}
        />
      </div>

      <div className="relative flex w-full max-w-3xl items-center justify-center overflow-hidden rounded-lg bg-neutral-900">
        {!loaded && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="h-10 w-10 animate-spin rounded-full border-2 border-white/30 border-t-white" />
          </div>
        )}

        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          key={pages[index]}
          src={pages[index]}
          alt={`Página ${index + 1}`}
          onLoad={() => setLoaded(true)}
          className="max-h-[85vh] w-auto select-none"
          draggable={false}
        />

        <button
          aria-label="Página anterior"
          onClick={() => goTo(index - 1)}
          className="absolute inset-y-0 left-0 w-1/2 cursor-pointer"
        />
        <button
          aria-label="Próxima página"
          onClick={() => goTo(index + 1)}
          className="absolute inset-y-0 right-0 w-1/2 cursor-pointer"
        />

        {pages[index + 1] && (
          // Warm the browser cache so the next page appears instantly.
          // eslint-disable-next-line @next/next/no-img-element
          <img src={pages[index + 1]} alt="" className="hidden" aria-hidden />
        )}
      </div>

      <div className="mt-4 flex items-center gap-4">
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
