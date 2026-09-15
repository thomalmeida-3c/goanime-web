import type { HomeItem, HomeResponse } from "@/lib/api";
import HomeRow from "./HomeRow";

export default function AnimesPage({
  home,
  onSelect,
}: {
  home: HomeResponse | null;
  onSelect: (item: HomeItem) => void;
}) {
  return (
    <div className="px-4 py-6 sm:px-6">
      <h1 className="mb-6 text-2xl font-bold text-white">Animes</h1>
      {!home && <p className="text-sm text-neutral-400">Carregando...</p>}
      {home && (
        <>
          <HomeRow title="Em alta no Brasil" items={home.trending} onSelect={onSelect} />
          <HomeRow title="Populares da temporada" items={home.seasonPopular} onSelect={onSelect} />
          <HomeRow title="Mais populares" items={home.allTimePopular} onSelect={onSelect} />
        </>
      )}
    </div>
  );
}
