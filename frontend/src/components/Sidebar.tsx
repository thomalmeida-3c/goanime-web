"use client";

import { BookOpen, Clapperboard, Home as HomeIcon, ShoppingBag } from "lucide-react";

export type NavView = "home" | "animes" | "mangas" | "produtos";

const items: { view: NavView; label: string; icon: React.ReactNode }[] = [
  { view: "home", label: "Início", icon: <HomeIcon size={20} /> },
  { view: "animes", label: "Animes", icon: <Clapperboard size={20} /> },
  { view: "mangas", label: "Mangás", icon: <BookOpen size={20} /> },
  { view: "produtos", label: "Produtos", icon: <ShoppingBag size={20} /> },
];

export default function Sidebar({
  active,
  onNavigate,
}: {
  active: NavView;
  onNavigate: (view: NavView) => void;
}) {
  return (
    <aside className="fixed inset-x-0 bottom-0 z-30 flex h-16 flex-row items-center justify-around gap-1 border-t border-neutral-800 bg-neutral-950 px-1 sm:sticky sm:inset-x-auto sm:top-0 sm:h-screen sm:w-56 sm:shrink-0 sm:flex-col sm:items-stretch sm:justify-start sm:gap-1 sm:border-t-0 sm:border-r sm:px-3 sm:py-4">
      <button
        onClick={() => onNavigate("home")}
        className="hidden items-center gap-2 px-2 py-2 text-lg font-bold tracking-tight text-purple-400 sm:mb-6 sm:flex"
      >
        <span className="text-xl">N</span>
        <span>nomad</span>
      </button>

      {items.map((item) => (
        <button
          key={item.view}
          onClick={() => onNavigate(item.view)}
          className={`flex flex-1 flex-col items-center justify-center gap-0.5 rounded-md px-2 py-1.5 text-[10px] font-medium transition sm:flex-none sm:flex-row sm:justify-start sm:gap-3 sm:px-3 sm:py-2.5 sm:text-sm ${
            active === item.view
              ? "bg-purple-600 text-white"
              : "text-neutral-400 hover:bg-neutral-900 hover:text-neutral-200"
          }`}
        >
          {item.icon}
          <span>{item.label}</span>
        </button>
      ))}
    </aside>
  );
}
