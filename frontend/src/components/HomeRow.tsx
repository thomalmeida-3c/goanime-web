import type { HomeItem } from "@/lib/api";

export default function HomeRow({
  title,
  items,
  onSelect,
}: {
  title: string;
  items: HomeItem[];
  onSelect: (item: HomeItem) => void;
}) {
  if (items.length === 0) return null;

  return (
    <section className="mb-8">
      <h2 className="mb-3 text-lg font-semibold text-neutral-100">{title}</h2>
      <div className="flex gap-3 overflow-x-auto pb-2">
        {items.map((item) => {
          const playable = Boolean(item.animeUrl && item.source);
          return (
            <button
              key={item.anilistId}
              onClick={() => playable && onSelect(item)}
              disabled={!playable}
              title={playable ? item.title : `${item.title} (indisponível no momento)`}
              className="group w-32 shrink-0 text-left sm:w-36"
            >
              <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-neutral-900">
                {item.imageUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={item.imageUrl}
                    alt={item.title}
                    className={`h-full w-full object-cover transition ${
                      playable ? "group-hover:scale-105" : "opacity-40 grayscale"
                    }`}
                  />
                ) : (
                  <div className="flex h-full items-center justify-center text-xs text-neutral-600">
                    sem imagem
                  </div>
                )}
                {item.score ? (
                  <span className="absolute left-1 top-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-amber-400">
                    ★ {(item.score / 10).toFixed(1)}
                  </span>
                ) : null}
                {!playable && (
                  <span className="absolute inset-x-0 bottom-0 bg-black/80 px-1.5 py-1 text-center text-[10px] text-neutral-300">
                    indisponível
                  </span>
                )}
              </div>
              <p className="mt-1.5 line-clamp-2 text-xs text-neutral-300">{item.title}</p>
            </button>
          );
        })}
      </div>
    </section>
  );
}
