"use client";

import { Bookmark, BookmarkCheck } from "lucide-react";

// The MangaDex-style title page block: blurred banner behind, cover art in
// front, big title, badges, description, a primary action (assistir/ler)
// and a save toggle. Shared by the anime and manga detail views — they only
// differ in which fields they have (anime has no description most of the
// time; manga has no numeric score) — everything here is optional except
// the title and the primary action.
export default function DetailHeader({
  title,
  bannerUrl,
  coverUrl,
  badges = [],
  score,
  description,
  actionLabel,
  onAction,
  saved,
  onToggleSave,
  canSave,
}: {
  title: string;
  bannerUrl?: string;
  coverUrl?: string;
  badges?: string[];
  score?: number;
  description?: string;
  actionLabel: string;
  onAction: () => void;
  saved: boolean;
  onToggleSave: () => void;
  canSave: boolean;
}) {
  const bg = bannerUrl || coverUrl;

  return (
    <div className="relative mb-8 overflow-hidden rounded-xl bg-neutral-900">
      {bg && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={bg}
          alt=""
          aria-hidden
          className="absolute inset-0 h-full w-full scale-110 object-cover opacity-40 blur-2xl"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-t from-neutral-950 via-neutral-950/70 to-neutral-950/40" />

      <div className="relative flex flex-col gap-5 p-6 sm:flex-row sm:items-end sm:p-8">
        {coverUrl && (
          <div className="h-48 w-32 shrink-0 overflow-hidden rounded-lg shadow-2xl ring-1 ring-white/10 sm:h-64 sm:w-44">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={coverUrl} alt={title} className="h-full w-full object-cover" />
          </div>
        )}

        <div className="min-w-0 flex-1">
          {(badges.length > 0 || score) && (
            <div className="mb-2 flex flex-wrap gap-1.5">
              {score ? (
                <span className="rounded bg-purple-600/80 px-2 py-0.5 text-[11px] font-semibold text-white">
                  ★ {(score / 10).toFixed(1)}
                </span>
              ) : null}
              {badges.map((b) => (
                <span
                  key={b}
                  className="rounded bg-white/10 px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide text-neutral-200"
                >
                  {b}
                </span>
              ))}
            </div>
          )}

          <h1 className="mb-2 text-2xl font-bold text-white sm:text-4xl">{title}</h1>

          {description && (
            <p className="mb-4 line-clamp-3 max-w-2xl text-sm text-neutral-300">{description}</p>
          )}

          <div className="flex items-center gap-2">
            <button
              onClick={onAction}
              className="rounded-md bg-purple-600 px-5 py-2 text-sm font-semibold text-white hover:bg-purple-500"
            >
              ▶ {actionLabel}
            </button>
            {canSave && (
              <button
                onClick={onToggleSave}
                aria-label={saved ? "Remover da biblioteca" : "Salvar"}
                title={saved ? "Remover da biblioteca" : "Salvar"}
                className="flex items-center justify-center rounded-md border border-neutral-600 p-2.5 text-white transition hover:border-purple-500"
              >
                {saved ? (
                  <BookmarkCheck size={18} className="text-purple-400" />
                ) : (
                  <Bookmark size={18} />
                )}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
