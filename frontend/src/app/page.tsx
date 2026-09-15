"use client";

import { useEffect, useState } from "react";
import HeroCarousel from "@/components/HeroCarousel";
import HomeRow from "@/components/HomeRow";
import UpscaledVideoPlayer from "@/components/UpscaledVideoPlayer";
import {
  type Anime,
  type Episode,
  type HomeItem,
  type HomeResponse,
  type SkipTimesResponse,
  getEpisodes,
  getHome,
  getSkipTimes,
  getStream,
  playbackUrl,
  searchAnime,
} from "@/lib/api";

type View = "home" | "search" | "episodes" | "player";

export default function Home() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Anime[]>([]);
  const [selectedAnime, setSelectedAnime] = useState<Anime | null>(null);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [selectedEpisode, setSelectedEpisode] = useState<Episode | null>(null);
  const [videoSrc, setVideoSrc] = useState<string | null>(null);
  const [skipTimes, setSkipTimes] = useState<SkipTimesResponse | null>(null);
  const [view, setView] = useState<View>("home");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [home, setHome] = useState<HomeResponse | null>(null);
  const [homeError, setHomeError] = useState<string | null>(null);

  useEffect(() => {
    getHome()
      .then(setHome)
      .catch((err) => setHomeError(err instanceof Error ? err.message : "Falha ao carregar a home"));
  }, []);

  // Home only carries AniList metadata (title/poster/synopsis) — it never
  // resolves a source. Picking a title (from the hero, a row, or the search
  // bar) always lands here: a plain text search across our own sources,
  // exactly like typing it in manually.
  async function runSearch(q: string) {
    if (!q.trim()) return;
    setQuery(q);
    setLoading(true);
    setError(null);
    try {
      const animes = await searchAnime(q.trim());
      setResults(animes);
      setView("search");
      if (animes.length === 0) {
        setError("Nenhum resultado encontrado.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha na busca");
    } finally {
      setLoading(false);
    }
  }

  function handleSearchSubmit(e: React.FormEvent) {
    e.preventDefault();
    runSearch(query);
  }

  function handleSelectHomeItem(item: HomeItem) {
    runSearch(item.title);
  }

  async function handleSelectAnime(anime: Anime) {
    setSelectedAnime(anime);
    setLoading(true);
    setError(null);
    try {
      const eps = await getEpisodes(anime.URL, anime.Source);
      setEpisodes(eps);
      setView("episodes");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao carregar episódios");
    } finally {
      setLoading(false);
    }
  }

  async function handleSelectEpisode(episode: Episode) {
    if (!selectedAnime) return;
    setSelectedEpisode(episode);
    setLoading(true);
    setError(null);
    setVideoSrc(null);
    setSkipTimes(null);

    // Best-effort and off the critical path: AniSkip needs a MAL id lookup
    // first (our sources never expose one), which can take a few seconds —
    // the episode plays immediately either way, the skip button just shows
    // up a beat later once/if this resolves.
    getSkipTimes(selectedAnime.Name, selectedAnime.URL, episode.Num)
      .then(setSkipTimes)
      .catch(() => {});

    try {
      const stream = await getStream(episode.URL, selectedAnime.Source);
      setVideoSrc(playbackUrl(stream.playbackUrl));
      setView("player");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao resolver o stream");
    } finally {
      setLoading(false);
    }
  }

  function goHome() {
    setView("home");
    setQuery("");
    setResults([]);
    setSelectedAnime(null);
    setEpisodes([]);
    setError(null);
  }

  function backToSearch() {
    setView("search");
    setSelectedAnime(null);
    setEpisodes([]);
  }

  function backToEpisodes() {
    setView("episodes");
    setSelectedEpisode(null);
    setVideoSrc(null);
  }

  const currentEpisodeIndex = selectedEpisode
    ? episodes.findIndex((ep) => ep.URL === selectedEpisode.URL)
    : -1;
  const previousEpisode = currentEpisodeIndex > 0 ? episodes[currentEpisodeIndex - 1] : null;
  const nextEpisode =
    currentEpisodeIndex >= 0 && currentEpisodeIndex < episodes.length - 1
      ? episodes[currentEpisodeIndex + 1]
      : null;

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100">
      <header className="sticky top-0 z-10 border-b border-neutral-800 px-6 py-3 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-4">
          <button onClick={goHome} className="shrink-0 text-lg font-semibold tracking-tight">
            nomad
          </button>
          <form onSubmit={handleSearchSubmit} className="flex flex-1 gap-2">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar anime..."
              className="flex-1 rounded-md border border-neutral-700 bg-neutral-900 px-4 py-1.5 text-sm outline-none focus:border-neutral-500"
            />
            <button
              type="submit"
              disabled={loading}
              className="rounded-md bg-purple-600 px-4 py-1.5 text-sm font-medium hover:bg-purple-500 disabled:opacity-50"
            >
              {loading ? "Buscando..." : "Buscar"}
            </button>
          </form>
        </div>
      </header>

      <main className="mx-auto">
        {view === "home" && (
          <>
            {homeError && <p className="mb-4 text-sm text-purple-400">{homeError}</p>}
            {home && (
              <>
                <HeroCarousel items={home.trending.slice(0, 6)} onSelect={handleSelectHomeItem} />
                <div className="mx-auto max-w-6xl">
                  <HomeRow
                    title="Animes em alta no Brasil"
                    items={home.trending}
                    onSelect={handleSelectHomeItem}
                  />
                  <HomeRow
                    title="Populares da temporada"
                    items={home.seasonPopular}
                    onSelect={handleSelectHomeItem}
                  />
                  <HomeRow
                    title="Mais populares"
                    items={home.allTimePopular}
                    onSelect={handleSelectHomeItem}
                  />
                </div>
              </>
            )}
          </>
        )}

        {view === "search" && (
          <>
            <button onClick={goHome} className="mb-4 text-sm text-neutral-400 hover:text-neutral-200">
              ← Voltar para a home
            </button>
            {loading && <p className="mb-4 text-sm text-neutral-400">Buscando...</p>}
            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}

            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
              {results.map((anime, i) => (
                <button
                  key={`${anime.Source}-${anime.URL}-${i}`}
                  onClick={() => handleSelectAnime(anime)}
                  className="group text-left"
                >
                  <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-neutral-900">
                    {anime.ImageURL ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={anime.ImageURL}
                        alt={anime.Name}
                        className="h-full w-full object-cover transition group-hover:scale-105"
                      />
                    ) : (
                      <div className="flex h-full items-center justify-center text-xs text-neutral-600">
                        sem imagem
                      </div>
                    )}
                    <span className="absolute right-1 top-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-neutral-200">
                      {anime.Source}
                    </span>
                  </div>
                  <p className="mt-1.5 line-clamp-2 text-xs text-neutral-300">{anime.Name}</p>
                </button>
              ))}
            </div>
          </>
        )}

        {view === "episodes" && selectedAnime && (
          <>
            <button
              onClick={backToSearch}
              className="mb-4 text-sm text-neutral-400 hover:text-neutral-200"
            >
              ← Voltar
            </button>
            <div className="mb-6 flex gap-4">
              {selectedAnime.ImageURL && (
                <div className="relative h-40 w-28 shrink-0 overflow-hidden rounded-lg bg-neutral-900">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={selectedAnime.ImageURL}
                    alt={selectedAnime.Name}
                    className="h-full w-full object-cover"
                  />
                </div>
              )}
              <div>
                <h2 className="text-lg font-semibold">{selectedAnime.Name}</h2>
                <p className="text-sm text-neutral-400">{selectedAnime.Source}</p>
              </div>
            </div>

            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
            {loading && <p className="text-sm text-neutral-400">Carregando...</p>}

            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {episodes.map((ep, i) => (
                <button
                  key={`${ep.URL}-${i}`}
                  onClick={() => handleSelectEpisode(ep)}
                  className="rounded-md border border-neutral-800 bg-neutral-900 px-4 py-2 text-left text-sm hover:border-neutral-600"
                >
                  {ep.Number}
                </button>
              ))}
            </div>
          </>
        )}

        {view === "player" && selectedAnime && selectedEpisode && (
          <div className="px-6 py-6">
            <button
              onClick={backToEpisodes}
              className="mb-4 text-sm text-neutral-400 hover:text-neutral-200"
            >
              ← Voltar para episódios
            </button>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div className="lg:col-span-2">
                <h2 className="mb-1 text-lg font-semibold">
                  {selectedAnime.Name} — {selectedEpisode.Number}
                </h2>
                <p className="mb-4 text-sm text-neutral-400">{selectedAnime.Source}</p>

                {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
                {loading && <p className="text-sm text-neutral-400">Resolvendo stream...</p>}

                {videoSrc && (
                  <UpscaledVideoPlayer key={videoSrc} src={videoSrc} skipTimes={skipTimes} />
                )}

                <div className="mt-3 grid grid-cols-2 gap-2">
                  <button
                    onClick={() => previousEpisode && handleSelectEpisode(previousEpisode)}
                    disabled={!previousEpisode}
                    className="rounded-md border border-neutral-800 bg-neutral-900 px-4 py-2 text-sm hover:border-neutral-600 disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    ← Episódio anterior
                  </button>
                  <button
                    onClick={() => nextEpisode && handleSelectEpisode(nextEpisode)}
                    disabled={!nextEpisode}
                    className="rounded-md border border-neutral-800 bg-neutral-900 px-4 py-2 text-sm hover:border-neutral-600 disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    Próximo episódio →
                  </button>
                </div>
              </div>

              <div className="lg:col-span-1">
                <h3 className="mb-3 text-sm font-semibold text-neutral-300">Episódios</h3>
                <div className="grid max-h-[540px] grid-cols-1 gap-2 overflow-y-auto pr-1">
                  {episodes.map((ep, i) => {
                    const isCurrent = ep.URL === selectedEpisode.URL;
                    return (
                      <button
                        key={`${ep.URL}-${i}`}
                        onClick={() => handleSelectEpisode(ep)}
                        className={`rounded-md border px-4 py-2 text-left text-sm ${
                          isCurrent
                            ? "border-purple-500 bg-purple-900/20 text-white"
                            : "border-neutral-800 bg-neutral-900 hover:border-neutral-600"
                        }`}
                      >
                        {ep.Number}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
