"use client";

import { LogOut, Trash2 } from "lucide-react";
import type { ChapterProgress, LibraryItem } from "@/lib/api";

// Mini profile: who you are, what you saved, and — for manga — where you
// stopped. No settings/history beyond that; this app has no account system
// to manage (see README "Login").
export default function ProfilePage({
  email,
  library,
  progress,
  onSelectItem,
  onRemoveItem,
  onLogout,
}: {
  email: string;
  library: LibraryItem[];
  progress: Record<string, ChapterProgress>;
  onSelectItem: (item: LibraryItem) => void;
  onRemoveItem: (item: LibraryItem) => void;
  onLogout: () => void;
}) {
  const animes = library.filter((i) => i.kind === "anime");
  const mangas = library.filter((i) => i.kind === "manga");

  return (
    <div className="px-4 py-6 sm:px-6">
      <div className="mb-8 flex items-center gap-4">
        <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-purple-600 text-xl font-bold text-white">
          {email[0]?.toUpperCase()}
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate font-semibold text-white">{email}</p>
          <p className="text-xs text-neutral-500">
            {animes.length} {animes.length === 1 ? "anime salvo" : "animes salvos"} ·{" "}
            {mangas.length} {mangas.length === 1 ? "mangá salvo" : "mangás salvos"}
          </p>
        </div>
        <button
          onClick={onLogout}
          className="flex shrink-0 items-center gap-1.5 rounded-md border border-neutral-700 px-3 py-2 text-sm text-neutral-300 hover:border-red-500 hover:text-red-400"
        >
          <LogOut size={15} />
          <span className="hidden sm:inline">Sair</span>
        </button>
      </div>

      <ProfileSection
        title="Mangás salvos"
        items={mangas}
        emptyText="Nenhum mangá salvo ainda."
        progress={progress}
        onSelectItem={onSelectItem}
        onRemoveItem={onRemoveItem}
      />

      <ProfileSection
        title="Animes salvos"
        items={animes}
        emptyText="Nenhum anime salvo ainda."
        progress={progress}
        onSelectItem={onSelectItem}
        onRemoveItem={onRemoveItem}
      />
    </div>
  );
}

function ProfileSection({
  title,
  items,
  emptyText,
  progress,
  onSelectItem,
  onRemoveItem,
}: {
  title: string;
  items: LibraryItem[];
  emptyText: string;
  progress: Record<string, ChapterProgress>;
  onSelectItem: (item: LibraryItem) => void;
  onRemoveItem: (item: LibraryItem) => void;
}) {
  return (
    <section className="mb-10">
      <h2 className="mb-3 text-lg font-bold text-neutral-100">{title}</h2>
      {items.length === 0 ? (
        <p className="text-sm text-neutral-500">{emptyText}</p>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
          {items.map((item) => {
            const p = progress[`${item.source}|${item.refId}`];
            return (
              <div key={`${item.kind}-${item.refId}`} className="group relative">
                <button onClick={() => onSelectItem(item)} className="block w-full text-left">
                  <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-neutral-900">
                    {item.imageUrl ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={item.imageUrl}
                        alt={item.title}
                        className="h-full w-full object-cover transition group-hover:scale-105"
                      />
                    ) : (
                      <div className="flex h-full items-center justify-center text-xs text-neutral-600">
                        sem capa
                      </div>
                    )}
                    <span className="absolute right-1 top-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-neutral-200">
                      {item.source}
                    </span>
                    {p && (
                      <span className="absolute inset-x-0 bottom-0 truncate bg-purple-600/90 px-1.5 py-1 text-[10px] font-medium text-white">
                        Parou no Cap. {p.chapter}
                      </span>
                    )}
                  </div>
                  <p className="mt-1.5 line-clamp-2 text-sm text-neutral-200">{item.title}</p>
                </button>
                <button
                  onClick={() => onRemoveItem(item)}
                  aria-label="Remover da biblioteca"
                  title="Remover da biblioteca"
                  className="absolute left-1 top-1 rounded-md bg-black/70 p-2 text-neutral-300 transition hover:text-red-400 sm:p-1.5 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
