"use client";

import { useEffect, useState } from "react";
import { type Product, searchProducts } from "@/lib/api";

function formatBRL(price: number): string {
  return price.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

// Compact product row on anime/manga detail pages, searched by the title.
// Renders nothing until Mercado Livre is connected (see ProdutosPage) —
// no broken-looking placeholder scattered across every detail page while
// that one-time setup hasn't happened yet.
export default function RelatedProducts({ query }: { query: string }) {
  const [products, setProducts] = useState<Product[]>([]);

  useEffect(() => {
    let cancelled = false;
    searchProducts(query)
      .then((res) => {
        if (cancelled) return;
        setProducts(res.connected ? res.products : []);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [query]);

  if (products.length === 0) return null;

  return (
    <section className="mb-10">
      <h2 className="mb-3 text-lg font-bold text-neutral-100">Produtos relacionados</h2>
      <div className="flex gap-4 overflow-x-auto pb-2">
        {products.slice(0, 10).map((p) => (
          <a
            key={p.id}
            href={p.url}
            target="_blank"
            rel="noopener noreferrer"
            className="w-36 shrink-0"
          >
            <div className="aspect-square overflow-hidden rounded-lg bg-white p-2">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={p.thumbnail} alt={p.title} className="h-full w-full object-contain" />
            </div>
            <p className="mt-2 line-clamp-2 text-xs text-neutral-300">{p.title}</p>
            <p className="text-sm font-semibold text-purple-400">{formatBRL(p.price)}</p>
          </a>
        ))}
      </div>
    </section>
  );
}
