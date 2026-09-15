"use client";

import { Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { type Anime, type MangaItem, searchAnime, searchManga } from "@/lib/api";

type Suggestion = { kind: "anime"; data: Anime } | { kind: "manga"; data: MangaItem };

// Debounced (~300ms) live search across both catalogs, reusing the same
// /api/search and /api/manga/search the full search view already calls —
// no new backend endpoint needed, just a lighter/faster-feeling entry point.
export default function SearchAutocomplete({
  onSelectAnime,
  onSelectManga,
  onSubmit,
}: {
  onSelectAnime: (anime: Anime) => void;
  onSelectManga: (manga: MangaItem) => void;
  onSubmit: (query: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const q = query.trim();
    if (q.length < 2) {
      setSuggestions([]);
      setOpen(false);
      return;
    }
    setLoading(true);
    const timer = setTimeout(() => {
      Promise.all([searchAnime(q).catch(() => []), searchManga(q).catch(() => [])])
        .then(([animes, mangas]) => {
          const merged: Suggestion[] = [
            ...(animes ?? []).slice(0, 4).map((a): Suggestion => ({ kind: "anime", data: a })),
            ...(mangas ?? []).slice(0, 3).map((m): Suggestion => ({ kind: "manga", data: m })),
          ];
          setSuggestions(merged);
          setOpen(true);
        })
        .finally(() => setLoading(false));
    }, 300);
    return () => clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setOpen(false);
    onSubmit(query);
  }

  function pick(s: Suggestion) {
    setOpen(false);
    setQuery("");
    if (s.kind === "anime") onSelectAnime(s.data);
    else onSelectManga(s.data);
  }

  return (
    <div ref={containerRef} className="relative w-full max-w-md">
      <form onSubmit={handleSubmit} className="flex gap-2">
        <div className="relative flex-1">
          <Search
            size={16}
            className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-500"
          />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onFocus={() => suggestions.length > 0 && setOpen(true)}
            placeholder="Buscar anime ou mangá..."
            className="w-full rounded-md border border-neutral-700 bg-neutral-900 py-1.5 pl-9 pr-3 text-sm text-white outline-none focus:border-purple-500"
          />
        </div>
        <button
          type="submit"
          className="rounded-md bg-purple-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-purple-500"
        >
          Buscar
        </button>
      </form>

      {open && (
        <div className="absolute left-0 right-0 top-full z-20 mt-1 max-h-96 overflow-y-auto rounded-md border border-neutral-800 bg-neutral-900 shadow-xl">
          {loading && <p className="px-3 py-2 text-xs text-neutral-500">Buscando...</p>}
          {!loading && suggestions.length === 0 && (
            <p className="px-3 py-2 text-xs text-neutral-500">Nenhum resultado.</p>
          )}
          {suggestions.map((s, i) => {
            const title = s.kind === "anime" ? s.data.Name : s.data.title;
            const image = s.kind === "anime" ? s.data.ImageURL : s.data.coverUrl;
            const source = s.kind === "anime" ? s.data.Source : s.data.source;
            return (
              <button
                key={`${s.kind}-${i}`}
                onClick={() => pick(s)}
                className="flex w-full items-center gap-3 px-3 py-2 text-left hover:bg-neutral-800"
              >
                <div className="h-12 w-9 shrink-0 overflow-hidden rounded bg-neutral-800">
                  {image && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={image} alt={title} className="h-full w-full object-cover" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm text-neutral-200">{title}</p>
                  <p className="text-xs text-neutral-500">
                    {s.kind === "anime" ? "Anime" : "Mangá"} · {source}
                  </p>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
