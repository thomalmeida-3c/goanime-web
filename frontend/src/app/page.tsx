"use client";

import { useEffect, useState } from "react";
import AnimesPage from "@/components/AnimesPage";
import DetailHeader from "@/components/DetailHeader";
import HeroCarousel from "@/components/HeroCarousel";
import HomeRow from "@/components/HomeRow";
import LoginModal from "@/components/LoginModal";
import MangaReader from "@/components/MangaReader";
import MangaRow from "@/components/MangaRow";
import MangasPage from "@/components/MangasPage";
import ProdutosPage from "@/components/ProdutosPage";
import ProfilePage from "@/components/ProfilePage";
import RelatedProducts from "@/components/RelatedProducts";
import Sidebar, { type NavView } from "@/components/Sidebar";
import TopBar from "@/components/TopBar";
import UpscaledVideoPlayer from "@/components/UpscaledVideoPlayer";
import {
  type Anime,
  type ChapterItem,
  type ChapterProgress,
  type Episode,
  type HomeItem,
  type HomeResponse,
  type LibraryItem,
  type MangaItem,
  type SkipTimesResponse,
  addToLibrary,
  getAllMangaProgress,
  getChapterPages,
  getEpisodes,
  getHome,
  getLibrary,
  getMangaChapters,
  getMangaHome,
  getMangaLatest,
  getMangaProgress,
  getSkipTimes,
  getStream,
  playbackUrl,
  removeFromLibrary,
  saveMangaProgress,
  searchAnime,
  searchManga,
} from "@/lib/api";

type View =
  | "home"
  | "animes"
  | "mangas"
  | "produtos"
  | "search"
  | "episodes"
  | "player"
  | "manga-chapters"
  | "manga-reader"
  | "profile";
type ContentFilter = "all" | "anime" | "manga";

// Chapter numbers aren't always plain integers (e.g. "108.100" for a split
// release) — parseFloat handles the common cases; anything unparseable
// (extras/specials labeled with text) sorts to the front rather than
// crashing the comparator.
function chapterNum(ch: ChapterItem): number {
  const n = parseFloat(ch.chapter);
  return Number.isNaN(n) ? 0 : n;
}

function sortChapters(chapters: ChapterItem[], ascending: boolean): ChapterItem[] {
  return [...chapters].sort((a, b) =>
    ascending ? chapterNum(a) - chapterNum(b) : chapterNum(b) - chapterNum(a),
  );
}

const EMAIL_STORAGE_KEY = "goanime:email";

export default function Home() {
  const [animeResults, setAnimeResults] = useState<Anime[]>([]);
  const [mangaResults, setMangaResults] = useState<MangaItem[]>([]);
  const [contentFilter, setContentFilter] = useState<ContentFilter>("all");
  const [selectedAnime, setSelectedAnime] = useState<Anime | null>(null);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [selectedEpisode, setSelectedEpisode] = useState<Episode | null>(null);
  const [videoSrc, setVideoSrc] = useState<string | null>(null);
  const [skipTimes, setSkipTimes] = useState<SkipTimesResponse | null>(null);
  const [selectedManga, setSelectedManga] = useState<MangaItem | null>(null);
  const [chapters, setChapters] = useState<ChapterItem[]>([]);
  const [chapterSortAsc, setChapterSortAsc] = useState(true);
  const [mangaProgress, setMangaProgress] = useState<ChapterProgress | null>(null);
  const [selectedChapter, setSelectedChapter] = useState<ChapterItem | null>(null);
  const [chapterPages, setChapterPages] = useState<string[]>([]);
  const [view, setView] = useState<View>("home");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [home, setHome] = useState<HomeResponse | null>(null);
  const [homeError, setHomeError] = useState<string | null>(null);
  const [mangaHome, setMangaHome] = useState<MangaItem[]>([]);
  const [mangaLatest, setMangaLatest] = useState<MangaItem[]>([]);

  const [email, setEmail] = useState<string | null>(null);
  const [library, setLibrary] = useState<LibraryItem[]>([]);
  const [libraryProgress, setLibraryProgress] = useState<Record<string, ChapterProgress>>({});
  const [showLogin, setShowLogin] = useState(false);

  // Shared by the initial-mount restore and a fresh login — both need the
  // same pair of fetches (saved library + per-manga "where you stopped").
  function loadLibrary(forEmail: string) {
    getLibrary(forEmail)
      .then(setLibrary)
      .catch(() => {});
    getAllMangaProgress(forEmail)
      .then(setLibraryProgress)
      .catch(() => {});
  }

  useEffect(() => {
    getHome()
      .then(setHome)
      .catch((err) => setHomeError(err instanceof Error ? err.message : "Falha ao carregar a home"));
    getMangaHome()
      .then((res) => setMangaHome(res.popular))
      .catch(() => {}); // best-effort row; the anime home still works without it
    getMangaLatest()
      .then((res) => setMangaLatest(res.latest))
      .catch(() => {});

    const savedEmail = localStorage.getItem(EMAIL_STORAGE_KEY);
    if (savedEmail) {
      setEmail(savedEmail);
      loadLibrary(savedEmail);
    }
  }, []);

  function handleLoggedIn(newEmail: string) {
    setEmail(newEmail);
    localStorage.setItem(EMAIL_STORAGE_KEY, newEmail);
    setShowLogin(false);
    loadLibrary(newEmail);
  }

  function handleLogout() {
    setEmail(null);
    setLibrary([]);
    setLibraryProgress({});
    localStorage.removeItem(EMAIL_STORAGE_KEY);
    setView("home");
  }

  function isSaved(kind: "anime" | "manga", refId: string) {
    return library.some((item) => item.kind === kind && item.refId === refId);
  }

  async function toggleSave(item: LibraryItem) {
    if (!email) {
      setShowLogin(true);
      return;
    }
    if (isSaved(item.kind, item.refId)) {
      await removeFromLibrary(email, item.kind, item.refId).catch(() => {});
      setLibrary((prev) => prev.filter((l) => !(l.kind === item.kind && l.refId === item.refId)));
    } else {
      await addToLibrary(email, item).catch(() => {});
      setLibrary((prev) => [...prev, item]);
    }
  }

  // Home only carries AniList metadata (title/poster/synopsis) — it never
  // resolves a source. Picking a title (from the hero, a row, or the search
  // bar) always lands here: a plain text search across our own sources,
  // exactly like typing it in manually. Manga is searched in parallel —
  // MangaDex is a separate catalog, not one of our scraper sources.
  async function runSearch(q: string) {
    if (!q.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const [animes, mangas] = await Promise.all([
        searchAnime(q.trim()).catch(() => []),
        searchManga(q.trim()).catch(() => []),
      ]);
      setAnimeResults(animes ?? []);
      setMangaResults(mangas ?? []);
      setView("search");
      if (animes.length === 0 && mangas.length === 0) {
        setError("Nenhum resultado encontrado.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha na busca");
    } finally {
      setLoading(false);
    }
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

  // Manga home cards carry a real MangaDex id already (unlike anime, where
  // AniList's home data never maps to a scraper URL) — so unlike
  // handleSelectHomeItem, this goes straight to the chapter list instead of
  // routing through a search.
  async function handleSelectManga(manga: MangaItem) {
    // MangaMillion is Shueisha's own official platform, not a fan
    // aggregator — its catalog is public but reading a chapter needs an
    // access token their JS mints client-side with no exposed API, so we
    // don't have a chapters/reader flow for it. Send the reader straight to
    // the source instead of pretending we can open it in-app.
    if (manga.source === "MangaMillion") {
      window.open(manga.id, "_blank", "noopener,noreferrer");
      return;
    }

    setSelectedManga(manga);
    setMangaProgress(null);
    setLoading(true);
    setError(null);
    try {
      const chs = await getMangaChapters(manga.id, manga.source);
      setChapters(chs ?? []);
      setView("manga-chapters");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao carregar capítulos");
    } finally {
      setLoading(false);
    }
    if (email) {
      getMangaProgress(email, manga.source, manga.id)
        .then((p) => setMangaProgress(p.found ? p : null))
        .catch(() => {});
    }
  }

  // Opens a saved library item from the profile page. A LibraryItem carries
  // just enough (refId/source/title/imageUrl) to re-enter either flow
  // directly — episodes/chapters don't need a search step, the same way a
  // home-row click does.
  function openLibraryItem(item: LibraryItem) {
    if (item.kind === "manga") {
      handleSelectManga({ id: item.refId, title: item.title, coverUrl: item.imageUrl, source: item.source });
    } else {
      handleSelectAnime({
        Name: item.title,
        URL: item.refId,
        ImageURL: item.imageUrl ?? "",
        Episodes: null,
        AnilistID: 0,
        MalID: 0,
        Source: item.source,
        Details: null,
      });
    }
  }

  async function handleSelectChapter(chapter: ChapterItem) {
    if (!selectedManga) return;
    setSelectedChapter(chapter);
    setLoading(true);
    setError(null);
    setChapterPages([]);
    try {
      const res = await getChapterPages(chapter.id, selectedManga.source);
      setChapterPages(res.pages ?? []);
      setView("manga-reader");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao carregar as páginas");
    } finally {
      setLoading(false);
    }
    // Best-effort, doesn't block reading: "where you stopped" is whatever
    // chapter was last opened, logged-in only.
    if (email) {
      saveMangaProgress(email, selectedManga.source, selectedManga.id, chapter.id, chapter.chapter).catch(
        () => {},
      );
      setMangaProgress({ found: true, chapterId: chapter.id, chapter: chapter.chapter });
    }
  }

  function goHome() {
    setView("home");
    setAnimeResults([]);
    setMangaResults([]);
    setContentFilter("all");
    setSelectedAnime(null);
    setEpisodes([]);
    setSelectedManga(null);
    setChapters([]);
    setMangaProgress(null);
    setError(null);
  }

  function handleNavigate(navView: NavView) {
    setAnimeResults([]);
    setMangaResults([]);
    setContentFilter("all");
    setSelectedAnime(null);
    setEpisodes([]);
    setSelectedManga(null);
    setChapters([]);
    setMangaProgress(null);
    setError(null);
    setView(navView);
  }

  function backToSearch() {
    setView("search");
    setSelectedAnime(null);
    setEpisodes([]);
    setSelectedManga(null);
    setChapters([]);
    setMangaProgress(null);
  }

  function backToEpisodes() {
    setView("episodes");
    setSelectedEpisode(null);
    setVideoSrc(null);
  }

  function backToChapters() {
    setView("manga-chapters");
    setSelectedChapter(null);
    setChapterPages([]);
  }

  const currentEpisodeIndex = selectedEpisode
    ? episodes.findIndex((ep) => ep.URL === selectedEpisode.URL)
    : -1;
  const previousEpisode = currentEpisodeIndex > 0 ? episodes[currentEpisodeIndex - 1] : null;
  const nextEpisode =
    currentEpisodeIndex >= 0 && currentEpisodeIndex < episodes.length - 1
      ? episodes[currentEpisodeIndex + 1]
      : null;

  const activeNav: NavView = (["home", "animes", "mangas", "produtos"] as const).includes(
    view as NavView,
  )
    ? (view as NavView)
    : "home";

  const isMangaReader = view === "manga-reader";

  return (
    <div className="flex min-h-screen bg-neutral-950 text-neutral-100">
      <Sidebar active={activeNav} onNavigate={handleNavigate} />

      <div className={`min-w-0 flex-1 ${isMangaReader ? "" : "pb-16 sm:pb-0"}`}>
        <div className={isMangaReader ? "hidden sm:block" : ""}>
          <TopBar
            email={email}
            onSelectAnime={handleSelectAnime}
            onSelectManga={handleSelectManga}
            onSubmitSearch={runSearch}
            onOpenLogin={() => setShowLogin(true)}
            onOpenProfile={() => setView("profile")}
          />
        </div>

        {showLogin && (
          <LoginModal onClose={() => setShowLogin(false)} onLoggedIn={handleLoggedIn} />
        )}

        <main>
        {view === "home" && (
          <>
            {homeError && <p className="mb-4 px-4 pt-4 text-sm text-purple-400 sm:px-6">{homeError}</p>}
            {home && <HeroCarousel items={home.trending.slice(0, 6)} onSelect={handleSelectHomeItem} />}
            <div className="px-4 sm:px-6">
              {home && (
                <>
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
                </>
              )}
              <MangaRow title="Mangás" items={mangaHome} onSelect={handleSelectManga} />
            </div>
          </>
        )}

        {view === "animes" && <AnimesPage home={home} onSelect={handleSelectHomeItem} />}

        {view === "mangas" && (
          <MangasPage popular={mangaHome} latest={mangaLatest} onSelect={handleSelectManga} />
        )}

        {view === "produtos" && <ProdutosPage />}

        {view === "profile" && email && (
          <ProfilePage
            email={email}
            library={library}
            progress={libraryProgress}
            onSelectItem={openLibraryItem}
            onRemoveItem={toggleSave}
            onLogout={handleLogout}
          />
        )}

        {view === "search" && (
          <div className="px-4 py-6 sm:px-6">
            <button onClick={goHome} className="mb-4 text-sm text-neutral-400 hover:text-neutral-200">
              ← Voltar para a home
            </button>

            <div className="mb-4 flex gap-2">
              {(
                [
                  ["all", `Todos (${animeResults.length + mangaResults.length})`],
                  ["anime", `Animes (${animeResults.length})`],
                  ["manga", `Mangás (${mangaResults.length})`],
                ] as [ContentFilter, string][]
              ).map(([value, label]) => (
                <button
                  key={value}
                  onClick={() => setContentFilter(value)}
                  className={`rounded-md px-3 py-1.5 text-sm ${
                    contentFilter === value
                      ? "bg-purple-600 text-white"
                      : "bg-neutral-900 text-neutral-400 hover:text-neutral-200"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>

            {loading && <p className="mb-4 text-sm text-neutral-400">Buscando...</p>}
            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}

            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4">
              {contentFilter !== "manga" &&
                animeResults.map((anime, i) => (
                  <button
                    key={`anime-${anime.Source}-${anime.URL}-${i}`}
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

              {contentFilter !== "anime" &&
                mangaResults.map((manga) => (
                  <button
                    key={`manga-${manga.id}`}
                    onClick={() => handleSelectManga(manga)}
                    className="group text-left"
                  >
                    <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-neutral-900">
                      {manga.coverUrl ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={manga.coverUrl}
                          alt={manga.title}
                          className="h-full w-full object-cover transition group-hover:scale-105"
                        />
                      ) : (
                        <div className="flex h-full items-center justify-center text-xs text-neutral-600">
                          sem capa
                        </div>
                      )}
                      <span className="absolute right-1 top-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-neutral-200">
                        {manga.source}
                      </span>
                    </div>
                    <p className="mt-1.5 line-clamp-2 text-xs text-neutral-300">{manga.title}</p>
                  </button>
                ))}
            </div>
          </div>
        )}

        {view === "episodes" && selectedAnime && (
          <div className="px-4 py-6 sm:px-6">
            <button
              onClick={backToSearch}
              className="mb-4 text-sm text-neutral-400 hover:text-neutral-200"
            >
              ← Voltar
            </button>

            <DetailHeader
              title={selectedAnime.Name}
              coverUrl={selectedAnime.ImageURL}
              badges={[selectedAnime.Source]}
              actionLabel="Assistir"
              onAction={() => episodes[0] && handleSelectEpisode(episodes[0])}
              saved={isSaved("anime", selectedAnime.URL)}
              onToggleSave={() =>
                toggleSave({
                  kind: "anime",
                  title: selectedAnime.Name,
                  imageUrl: selectedAnime.ImageURL,
                  refId: selectedAnime.URL,
                  source: selectedAnime.Source,
                })
              }
              canSave={!!email}
            />

            <RelatedProducts query={selectedAnime.Name} />

            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
            {loading && <p className="text-sm text-neutral-400">Carregando...</p>}

            <h2 className="mb-3 text-sm font-semibold text-neutral-300">Episódios</h2>
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
          </div>
        )}

        {view === "player" && selectedAnime && selectedEpisode && (
          <div className="px-4 py-6 sm:px-6">
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

        {view === "manga-chapters" && selectedManga && (
          <div className="px-4 py-6 sm:px-6">
            <button onClick={backToSearch} className="mb-4 text-sm text-neutral-400 hover:text-neutral-200">
              ← Voltar
            </button>

            <DetailHeader
              title={selectedManga.title}
              coverUrl={selectedManga.coverUrl}
              badges={[selectedManga.source, ...(selectedManga.tags ?? [])]}
              description={selectedManga.description}
              actionLabel={mangaProgress ? `Continuar — Cap. ${mangaProgress.chapter}` : "Ler"}
              onAction={() => {
                const target =
                  chapters.find((c) => c.id === mangaProgress?.chapterId) ?? chapters[0];
                if (target) handleSelectChapter(target);
              }}
              saved={isSaved("manga", selectedManga.id)}
              onToggleSave={() =>
                toggleSave({
                  kind: "manga",
                  title: selectedManga.title,
                  imageUrl: selectedManga.coverUrl,
                  refId: selectedManga.id,
                  source: selectedManga.source,
                })
              }
              canSave={!!email}
            />

            <RelatedProducts query={selectedManga.title} />

            {error && <p className="mb-4 text-sm text-red-400">{error}</p>}
            {loading && <p className="text-sm text-neutral-400">Carregando...</p>}
            {!loading && chapters.length === 0 && !error && (
              <p className="text-sm text-neutral-400">Nenhum capítulo em português encontrado.</p>
            )}

            {chapters.length > 0 && (
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-semibold text-neutral-300">Capítulos</h2>
                <button
                  onClick={() => setChapterSortAsc((asc) => !asc)}
                  className="rounded-md border border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs text-neutral-300 hover:border-purple-500 hover:text-purple-300"
                >
                  {chapterSortAsc ? "↑ Menor primeiro" : "↓ Maior primeiro"}
                </button>
              </div>
            )}
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4">
              {sortChapters(chapters, chapterSortAsc).map((ch) => {
                const isLastRead = ch.id === mangaProgress?.chapterId;
                return (
                  <button
                    key={ch.id}
                    onClick={() => handleSelectChapter(ch)}
                    className={`rounded-md border px-4 py-2 text-left text-sm ${
                      isLastRead
                        ? "border-purple-500 bg-purple-950/40 text-purple-200"
                        : "border-neutral-800 bg-neutral-900 hover:border-neutral-600"
                    }`}
                  >
                    Cap. {ch.chapter}
                    {ch.title ? ` — ${ch.title}` : ""}
                    {isLastRead && <span className="ml-1.5 text-xs text-purple-400">●</span>}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        {view === "manga-reader" && selectedManga && selectedChapter && (
          <div className="flex h-[calc(100dvh-4rem)] flex-col px-0 py-0 sm:h-auto sm:px-6 sm:py-6">
            <div className="flex shrink-0 items-center justify-between gap-2 border-b border-neutral-800 px-3 py-2 sm:mb-4 sm:border-0 sm:px-0 sm:py-0">
              <button
                onClick={backToChapters}
                className="shrink-0 text-sm text-neutral-400 hover:text-neutral-200"
              >
                ← <span className="hidden sm:inline">Voltar para capítulos</span>
              </button>
              <h2 className="truncate text-xs font-semibold text-neutral-300 sm:text-sm">
                {selectedManga.title} — Cap. {selectedChapter.chapter}
              </h2>
            </div>

            {error && <p className="shrink-0 px-3 py-2 text-sm text-red-400 sm:px-0">{error}</p>}
            {loading && (
              <p className="shrink-0 px-3 py-2 text-sm text-neutral-400 sm:px-0">Carregando páginas...</p>
            )}

            {!loading && (
              <div className="min-h-0 flex-1">
                <MangaReader pages={chapterPages} />
              </div>
            )}
          </div>
        )}
        </main>
      </div>
    </div>
  );
}
