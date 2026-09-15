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
  animeUrl?: string;
  source?: string;
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

export function playbackUrl(path: string): string {
  return API_BASE + path;
}
