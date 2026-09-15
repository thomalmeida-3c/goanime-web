const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type Anime = {
  Name: string;
  URL: string;
  ImageURL: string;
  Episodes: Episode[] | null;
  AnilistID: number;
  MalID: number;
  Source: string;
  Details: AniListDetails | null;
};

export type Episode = {
  Number: string;
  Num: number;
  URL: string;
  Title: TitleDetails | null;
  Aired: string;
  Duration: number;
  IsFiller: boolean;
  IsRecap: boolean;
  Synopsis: string;
  SkipTimes: SkipTimes | null;
};

export type TitleDetails = {
  Romaji: string;
  English: string;
  Japanese: string;
};

export type SkipTimes = {
  Op: SkipTime | null;
  Ed: SkipTime | null;
};

export type SkipTime = {
  Start: number;
  End: number;
};

export type AniListDetails = {
  ID: number;
  IDMal: number;
  Title: { Romaji: string; English: string } | null;
  Description: string;
  Genres: string[] | null;
  AverageScore: number;
  Episodes: number;
  Status: string;
  CoverImage: { Large: string; Medium: string } | null;
};

export type StreamResponse = {
  id: string;
  playbackUrl: string;
  streamUrl: string;
  metadata: Record<string, string> | null;
};

export type HomeItem = {
  anilistId: number;
  title: string;
  imageUrl: string;
  bannerUrl?: string;
  description?: string;
  score?: number;
  genres?: string[];
};

export type HomeResponse = {
  trending: HomeItem[];
  seasonPopular: HomeItem[];
  allTimePopular: HomeItem[];
  generatedAt: string;
};

type ApiError = { error: string };

async function request<T>(path: string, params: Record<string, string>): Promise<T> {
  const url = new URL(API_BASE + path);
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value);
  }

  const res = await fetch(url.toString());
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiError | null;
    throw new Error(body?.error ?? `request failed with status ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export function searchAnime(query: string): Promise<Anime[]> {
  return request<Anime[]>("/api/search", { q: query });
}

export function getEpisodes(animeUrl: string, source: string): Promise<Episode[]> {
  return request<Episode[]>("/api/episodes", { url: animeUrl, source });
}

export function getStream(episodeUrl: string, source: string): Promise<StreamResponse> {
  return request<StreamResponse>("/api/stream", { episodeUrl, source });
}

export function getHome(): Promise<HomeResponse> {
  return request<HomeResponse>("/api/home", {});
}

export type SkipInterval = { start: number; end: number };
export type SkipTimesResponse = { op: SkipInterval | null; ed: SkipInterval | null };

// getSkipTimes resolves AniSkip's opening/ending timestamps for one episode
// (best-effort: comes back {op: null, ed: null} rather than an error when
// the title or episode has no data, not just on a hard failure).
export function getSkipTimes(
  animeName: string,
  animeUrl: string,
  episodeNum: number,
): Promise<SkipTimesResponse> {
  return request<SkipTimesResponse>("/api/skip", {
    animeName,
    animeUrl,
    episodeNum: String(episodeNum),
  });
}

export function playbackUrl(path: string): string {
  return API_BASE + path;
}

export type MangaItem = {
  id: string;
  title: string;
  description?: string;
  coverUrl?: string;
  tags?: string[];
  contentRating?: string;
  year?: number;
  source: string;
};

export type ChapterItem = {
  id: string;
  chapter: string;
  title?: string;
};

export type ChapterPagesResponse = { pages: string[] };

export function getMangaHome(): Promise<{ popular: MangaItem[] }> {
  return request("/api/manga/home", {});
}

export function searchManga(query: string): Promise<MangaItem[]> {
  return request<MangaItem[]>("/api/manga/search", { q: query });
}

// getMangaChapters/getChapterPages need `source` because MangaDex and
// MangaLivre ids come from unrelated schemes (a MangaDex UUID vs a
// MangaLivre page URL) — the backend uses it to route to the right scraper.
export function getMangaChapters(mangaId: string, source: string): Promise<ChapterItem[]> {
  return request<ChapterItem[]>("/api/manga/chapters", { id: mangaId, source });
}

export function getChapterPages(chapterId: string, source: string): Promise<ChapterPagesResponse> {
  return request<ChapterPagesResponse>("/api/manga/pages", { id: chapterId, source });
}

export function getMangaLatest(): Promise<{ latest: MangaItem[] }> {
  return request("/api/manga/latest", {});
}

// --- Progresso de leitura ---------------------------------------------------
// "Onde você parou" por obra — o último capítulo aberto, não uma posição de
// página. Salvo automaticamente ao abrir um capítulo; sem login, é no-op.

export type ChapterProgress = { found: boolean; chapterId: string; chapter: string; updatedAt?: string };

export function getMangaProgress(email: string, source: string, mangaId: string): Promise<ChapterProgress> {
  return request<ChapterProgress>("/api/manga/progress", { email, source, mangaId });
}

// Keyed "source|mangaId" — matches every saved manga to its progress (if
// any) without one request per library item, for the profile page.
export function getAllMangaProgress(email: string): Promise<Record<string, ChapterProgress>> {
  return request<Record<string, ChapterProgress>>("/api/manga/progress/all", { email });
}

export function saveMangaProgress(
  email: string,
  source: string,
  mangaId: string,
  chapterId: string,
  chapter: string,
): Promise<{ ok: boolean }> {
  return requestJSON<{ ok: boolean }>(
    "/api/manga/progress",
    "POST",
    {},
    { email, source, mangaId, chapterId, chapter },
  );
}

async function requestJSON<T>(
  path: string,
  method: "POST" | "DELETE",
  params: Record<string, string>,
  body?: unknown,
): Promise<T> {
  const url = new URL(API_BASE + path);
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value);
  }
  const res = await fetch(url.toString(), {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const errBody = (await res.json().catch(() => null)) as ApiError | null;
    throw new Error(errBody?.error ?? `request failed with status ${res.status}`);
  }
  return res.json() as Promise<T>;
}

// --- Auth + biblioteca -----------------------------------------------------
// Deliberately not real authentication: email only, no password, no
// verification email (this app has no email-sending infra). Good enough to
// remember a personal watch/read list across visits, not a security
// boundary — see README "Login".

export type LibraryItem = {
  kind: "anime" | "manga";
  title: string;
  imageUrl?: string;
  refId: string;
  source: string;
  addedAt?: string;
};

export type AuthUser = { email: string; library: LibraryItem[] };

export function login(email: string): Promise<AuthUser> {
  return requestJSON<AuthUser>("/api/auth/login", "POST", {}, { email });
}

export function getLibrary(email: string): Promise<LibraryItem[]> {
  return request<LibraryItem[]>("/api/library", { email });
}

export function addToLibrary(email: string, item: LibraryItem): Promise<{ ok: boolean }> {
  return requestJSON<{ ok: boolean }>("/api/library", "POST", {}, { email, item });
}

export function removeFromLibrary(
  email: string,
  kind: string,
  refId: string,
): Promise<{ ok: boolean }> {
  return requestJSON<{ ok: boolean }>("/api/library", "DELETE", { email, kind, refId });
}

// --- Produtos (Mercado Livre) -----------------------------------------------
// Real product search, gated behind a one-time OAuth connection (Mercado
// Livre has no open/keyless search anymore) — see README "Produtos".

export type Product = {
  id: string;
  title: string;
  price: number;
  thumbnail: string;
  url: string;
};

export type ProductsResponse = { connected: boolean; products: Product[] };
export type MLStatus = { configured: boolean; connected: boolean };

export function searchProducts(query: string): Promise<ProductsResponse> {
  return request<ProductsResponse>("/api/products", { q: query });
}

export function getMLStatus(): Promise<MLStatus> {
  return request<MLStatus>("/api/ml/status", {});
}

export function mlConnectUrl(): string {
  return API_BASE + "/api/ml/connect";
}
