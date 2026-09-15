"use client";

import { useEffect, useState } from "react";
import HomeRow from "@/components/HomeRow";
import {
  type Anime,
  type Episode,
  type HomeItem,
  type HomeResponse,
  getEpisodes,
  getHome,
  getStream,
  playbackUrl,
  searchAnime,
} from "@/lib/api";

type View = "home" | "search" | "episodes" | "player";

function homeItemToAnime(item: HomeItem): Anime {
  return {
    Name: item.title,
    URL: item.animeUrl ?? "",
    ImageURL: item.imageUrl,
    Episodes: null,
    AnilistID: item.anilistId,
    MalID: 0,
    Source: item.source ?? "",
    Details: null,
  };
}

export default function Home() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Anime[]>([]);
  const [selectedAnime, setSelectedAnime] = useState<Anime | null>(null);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [selectedEpisode, setSelectedEpisode] = useState<Episode | null>(null);
  const [videoSrc, setVideoSrc] = useState<string | null>(null);
  const [view, setView] = useState<View>("home");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [home, setHome] = useState<HomeResponse | null>(null);
  const [homeLoading, setHomeLoading] = useState(true);
  const [homeError, setHomeError] = useState<string | null>(null);

  useEffect(() => {
    getHome()
      .then(setHome)
      .catch((err) => setHomeError(err instanceof Error ? err.message : "Falha ao carregar a home"))
      .finally(() => setHomeLoading(false));
  }, []);

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const animes = await searchAnime(query.trim());
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

  function handleSelectHomeItem(item: HomeItem) {
    if (!item.animeUrl || !item.source) return;
    handleSelectAnime(homeItemToAnime(item));
  }

  async function handleSelectEpisode(episode: Episode) {
    if (!selectedAnime) return;
    setSelectedEpisode(episode);
    setLoading(true);
    setError(null);
    setVideoSrc(null);
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

  const heroItem = home?.trending?.[0];

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100">
      <header className="sticky top-0 z-10 border-b border-neutral-800 bg-neutral-950/90 px-6 py-3 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-4">
          <button onClick={goHome} className="shrink-0 text-lg font-semibold tracking-tight">
            GoAnime Web
          </button>
          <form onSubmit={handleSearch} className="flex flex-1 gap-2">
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
              className="rounded-md bg-indigo-600 px-4 py-1.5 text-sm font-medium hover:bg-indigo-500 disabled:opacity-50"
            >
              {loading ? "Buscando..." : "Buscar"}
            </button>
          </form>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-8">
        {view === "home" && (
          <>
            {heroItem?.bannerUrl && (
              <div className="relative mb-8 h-56 overflow-hidden rounded-xl sm:h-72">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={heroItem.bannerUrl}
                  alt={heroItem.title}
                  className="h-full w-full object-cover"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-neutral-950 via-neutral-950/30 to-transparent" />
                <div className="absolute bottom-0 left-0 p-5">
                  <p className="mb-1 text-xs uppercase tracking-wide text-indigo-400">
                    Em alta agora
                  </p>
                  <h2 className="text-2xl font-bold">{heroItem.title}</h2>
                  {heroItem.animeUrl && heroItem.source && (
                    <button
                      onClick={() => handleSelectHomeItem(heroItem)}
                      className="mt-3 rounded-md bg-indigo-600 px-4 py-1.5 text-sm font-medium hover:bg-indigo-500"
                    >
                      Ver episódios
                    </button>
                  )}
                </div>
              </div>
            )}

            {homeLoading && (
              <p className="text-sm text-neutral-400">
                Carregando destaques (primeira carga pode levar ~20s)...
              </p>
            )}
            {homeError && <p className="text-sm text-red-400">{homeError}</p>}

            {home && (
              <>
                <HomeRow title="Em alta" items={home.trending} onSelect={handleSelectHomeItem} />
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
              </>
            )}
          </>
        )}

        {view === "search" && (
          <>
            <button onClick={goHome} className="mb-4 text-sm text-neutral-400 hover:text-neutral-200">
              ← Voltar para a home
            </button>
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
          <>
            <button
              onClick={backToEpisodes}
              className="mb-4 text-sm text-neutral-400 hover:text-neutral-200"
            >
              ← Voltar para episódios
            </button>
            <h2 className="mb-1 text-lg font-semibold">
              {selectedAnime.Name} — {selectedEpisode.Number}
            </h2>
            <p className="mb-4 text-sm text-neutral-400">{selectedAnime.Source}</p>

            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
            {loading && <p className="text-sm text-neutral-400">Resolvendo stream...</p>}

            {videoSrc && (
              <video
                key={videoSrc}
                controls
                autoPlay
                className="w-full rounded-lg bg-black"
                src={videoSrc}
              >
                Seu navegador não suporta vídeo HTML5.
              </video>
            )}
          </>
        )}
      </main>
    </div>
  );
}
