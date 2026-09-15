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
    <section className="mb-10">
      <h2 className="mb-3 text-lg font-bold text-neutral-100">{title}</h2>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {items.map((item) => (
          <button
            key={item.anilistId}
            onClick={() => onSelect(item)}
            className="group w-36 shrink-0 text-left sm:w-44"
          >
            <div className="relative aspect-[2/3] overflow-hidden rounded-lg bg-neutral-900">
              {item.imageUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={item.imageUrl}
                  alt={item.title}
                  className="h-full w-full object-cover transition duration-200 group-hover:scale-105"
                />
              ) : (
                <div className="flex h-full items-center justify-center text-xs text-neutral-600">
                  sem imagem
                </div>
              )}
            </div>
            <p className="mt-2 line-clamp-2 text-sm text-neutral-200">{item.title}</p>
          </button>
        ))}
      </div>
    </section>
  );
}
