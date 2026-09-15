"use client";

import { useEffect, useState } from "react";
import { type MLStatus, type Product, getMLStatus, mlConnectUrl, searchProducts } from "@/lib/api";

function formatBRL(price: number): string {
  return price.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

// Real Mercado Livre product search, gated behind a one-time OAuth
// connection — their old open/keyless public search API was retired (see
// README "Produtos"). Until that's connected, this shows exactly what's
// missing instead of a silently empty grid.
export default function ProdutosPage() {
  const [query, setQuery] = useState("");
  const [products, setProducts] = useState<Product[]>([]);
  const [connected, setConnected] = useState<boolean | null>(null);
  const [status, setStatus] = useState<MLStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);

  useEffect(() => {
    getMLStatus()
      .then(setStatus)
      .catch(() => {});
  }, []);

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
    try {
      const res = await searchProducts(query.trim());
      setConnected(res.connected);
      setProducts(res.products);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="px-4 py-6 sm:px-6">
      <h1 className="mb-2 text-2xl font-bold text-white">Produtos</h1>
      <p className="mb-6 text-sm text-neutral-400">
        Mangás, action figures e outros produtos relacionados, no Mercado Livre.
      </p>

      <form onSubmit={handleSearch} className="mb-6 flex max-w-md gap-2">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Ex: one piece action figure"
          className="flex-1 rounded-md border border-neutral-700 bg-neutral-900 px-4 py-2 text-sm text-white outline-none focus:border-purple-500"
        />
        <button
          type="submit"
          disabled={loading}
          className="rounded-md bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-500 disabled:opacity-50"
        >
          {loading ? "Buscando..." : "Buscar"}
        </button>
      </form>

      {searched && connected === false && (
        <div className="mb-6 max-w-lg rounded-lg border border-neutral-800 bg-neutral-900 p-5">
          <p className="mb-2 text-sm font-medium text-white">Mercado Livre ainda não conectado</p>
          {status?.configured ? (
            <>
              <p className="mb-3 text-sm text-neutral-400">
                As credenciais já estão configuradas — falta só autorizar o app uma vez.
              </p>
              <a
                href={mlConnectUrl()}
                className="inline-block rounded-md bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-500"
              >
                Conectar Mercado Livre
              </a>
            </>
          ) : (
            <p className="text-sm text-neutral-400">
              Precisa cadastrar um app no portal de desenvolvedores do Mercado Livre e configurar{" "}
              <code className="rounded bg-neutral-800 px-1.5 py-0.5 text-xs">ML_CLIENT_ID</code>,{" "}
              <code className="rounded bg-neutral-800 px-1.5 py-0.5 text-xs">ML_CLIENT_SECRET</code>{" "}
              e{" "}
              <code className="rounded bg-neutral-800 px-1.5 py-0.5 text-xs">ML_REDIRECT_URI</code>{" "}
              no backend. Veja o README.
            </p>
          )}
        </div>
      )}

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
        {products.map((p) => (
          <a key={p.id} href={p.url} target="_blank" rel="noopener noreferrer" className="group">
            <div className="aspect-square overflow-hidden rounded-lg bg-white p-3">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={p.thumbnail}
                alt={p.title}
                className="h-full w-full object-contain transition group-hover:scale-105"
              />
            </div>
            <p className="mt-2 line-clamp-2 text-xs text-neutral-300">{p.title}</p>
            <p className="text-sm font-semibold text-purple-400">{formatBRL(p.price)}</p>
          </a>
        ))}
      </div>
    </div>
  );
}
