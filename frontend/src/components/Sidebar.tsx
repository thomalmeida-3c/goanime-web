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
    <aside className="sticky top-0 flex h-screen w-16 shrink-0 flex-col gap-1 border-r border-neutral-800 bg-neutral-950 px-2 py-4 sm:w-56 sm:px-3">
      <button
        onClick={() => onNavigate("home")}
        className="mb-6 flex items-center justify-center gap-2 px-2 py-2 text-lg font-bold tracking-tight text-purple-400 sm:justify-start"
      >
        <span className="text-xl">N</span>
        <span className="hidden sm:inline">nomad</span>
      </button>

      {items.map((item) => (
        <button
          key={item.view}
          onClick={() => onNavigate(item.view)}
          className={`flex items-center justify-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition sm:justify-start ${
            active === item.view
              ? "bg-purple-600 text-white"
              : "text-neutral-400 hover:bg-neutral-900 hover:text-neutral-200"
          }`}
        >
          {item.icon}
          <span className="hidden sm:inline">{item.label}</span>
        </button>
      ))}
    </aside>
  );
}
