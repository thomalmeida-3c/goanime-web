import { useEffect, useState } from "react";
import type { HomeItem } from "@/lib/api";

export default function HeroCarousel({
  items,
  onSelect,
}: {
  items: HomeItem[];
  onSelect: (item: HomeItem) => void;
}) {
  const [index, setIndex] = useState(0);

  // Re-armed on every index change (auto or manual), so a manual click
  // resets the auto-advance clock instead of jumping again a moment later.
  useEffect(() => {
    if (items.length <= 1) return;
    const id = setTimeout(() => setIndex((i) => (i + 1) % items.length), 7000);
    return () => clearTimeout(id);
  }, [index, items.length]);

  if (items.length === 0) return null;
  const item = items[index];

  function go(delta: number) {
    setIndex((i) => (i + delta + items.length) % items.length);
  }

  return (
    <div className="relative mb-10 h-[420px] overflow-hidden rounded-xl bg-neutral-900 sm:h-[480px] md:h-[560px]">
      {(item.bannerUrl ?? item.imageUrl) && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          key={item.anilistId}
          src={item.bannerUrl ?? item.imageUrl}
          alt={item.title}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-r from-neutral-950 via-neutral-950/60 to-transparent" />
      <div className="absolute inset-0 bg-gradient-to-t from-neutral-950 via-transparent to-transparent" />

      {items.length > 1 && (
        <>
          <button
            onClick={() => go(-1)}
            aria-label="Anterior"
            className="absolute left-3 top-1/2 -translate-y-1/2 text-3xl text-white/50 transition hover:text-white"
          >
            ‹
          </button>
          <button
            onClick={() => go(1)}
            aria-label="Próximo"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-3xl text-white/50 transition hover:text-white"
          >
            ›
          </button>
        </>
      )}

      <div className="absolute bottom-0 left-0 max-w-xl p-6 sm:p-10">
        <h1 className="text-3xl font-extrabold leading-tight text-white drop-shadow-lg sm:text-5xl">
          {item.title}
        </h1>

        {(item.score || (item.genres && item.genres.length > 0)) && (
          <p className="mt-3 text-xs font-medium text-neutral-300 sm:text-sm">
            {item.score ? <span className="text-orange-400">★ {(item.score / 10).toFixed(1)}</span> : null}
            {item.score && item.genres?.length ? " • " : ""}
            {item.genres?.join(", ")}
          </p>
        )}

        {item.description && (
          <p className="mt-3 line-clamp-3 text-sm text-neutral-300">{item.description}</p>
        )}

        <div className="mt-5 flex items-center gap-3">
          <button
            onClick={() => onSelect(item)}
            className="flex items-center gap-2 rounded-md bg-orange-600 px-5 py-2.5 text-sm font-bold uppercase tracking-wide text-white transition hover:bg-orange-500"
          >
            ▶ Começar a assistir
          </button>
          <button
            aria-label="Salvar"
            className="rounded-md border border-neutral-500 p-2.5 text-white transition hover:border-white"
          >
            🔖
          </button>
        </div>

        {items.length > 1 && (
          <div className="mt-5 flex gap-1.5">
            {items.map((it, i) => (
              <button
                key={it.anilistId}
                onClick={() => setIndex(i)}
                aria-label={`Ir para slide ${i + 1}`}
                className={`h-1.5 rounded-full transition-all ${
                  i === index ? "w-6 bg-orange-500" : "w-4 bg-neutral-600 hover:bg-neutral-400"
                }`}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
