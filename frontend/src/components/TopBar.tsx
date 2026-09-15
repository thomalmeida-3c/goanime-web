"use client";

import { User } from "lucide-react";
import type { Anime, MangaItem } from "@/lib/api";
import SearchAutocomplete from "./SearchAutocomplete";

export default function TopBar({
  email,
  onSelectAnime,
  onSelectManga,
  onSubmitSearch,
  onOpenLogin,
}: {
  email: string | null;
  onSelectAnime: (anime: Anime) => void;
  onSelectManga: (manga: MangaItem) => void;
  onSubmitSearch: (query: string) => void;
  onOpenLogin: () => void;
}) {
  return (
    <header className="sticky top-0 z-10 flex items-center gap-4 border-b border-neutral-800 bg-neutral-950/90 px-4 py-3 backdrop-blur sm:px-6">
      <SearchAutocomplete
        onSelectAnime={onSelectAnime}
        onSelectManga={onSelectManga}
        onSubmit={onSubmitSearch}
      />
      <span className="flex-1" />
      <button
        onClick={onOpenLogin}
        className="flex items-center gap-2 rounded-full border border-neutral-700 px-3 py-1.5 text-sm text-neutral-300 hover:border-purple-500 hover:text-white"
      >
        <User size={16} />
        <span className="hidden max-w-[10rem] truncate sm:inline">{email ?? "Entrar"}</span>
      </button>
    </header>
  );
}
