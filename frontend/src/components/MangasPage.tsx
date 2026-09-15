import type { MangaItem } from "@/lib/api";
import MangaRow from "./MangaRow";

export default function MangasPage({
  popular,
  latest,
  onSelect,
}: {
  popular: MangaItem[];
  latest: MangaItem[];
  onSelect: (item: MangaItem) => void;
}) {
  return (
    <div className="px-4 py-6 sm:px-6">
      <h1 className="mb-6 text-2xl font-bold text-white">Mangás</h1>
      {popular.length === 0 && latest.length === 0 && (
        <p className="text-sm text-neutral-400">Carregando...</p>
      )}
      <MangaRow title="Populares" items={popular} onSelect={onSelect} />
      <MangaRow title="Recém adicionados" items={latest} onSelect={onSelect} />
    </div>
  );
}
