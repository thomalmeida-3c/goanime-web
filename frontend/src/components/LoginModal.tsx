"use client";

import { useState } from "react";
import { login } from "@/lib/api";

// Deliberately not real authentication: email only, no password, no
// verification — this app has no email-sending infrastructure. Good enough
// to remember a personal watch/read list across visits.
export default function LoginModal({
  onClose,
  onLoggedIn,
}: {
  onClose: () => void;
  onLoggedIn: (email: string) => void;
}) {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const user = await login(email);
      onLoggedIn(user.email);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao entrar");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
      onClick={onClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-sm rounded-lg border border-neutral-800 bg-neutral-900 p-6"
      >
        <h2 className="mb-1 text-lg font-semibold text-white">Entrar</h2>
        <p className="mb-4 text-sm text-neutral-400">
          Só seu email, sem senha — pra salvar animes e mangás.
        </p>
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="seu@email.com"
            className="rounded-md border border-neutral-700 bg-neutral-950 px-3 py-2 text-sm text-white outline-none focus:border-purple-500"
          />
          {error && <p className="text-xs text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="rounded-md bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-500 disabled:opacity-50"
          >
            {loading ? "Entrando..." : "Entrar"}
          </button>
        </form>
        <button
          onClick={onClose}
          className="mt-3 w-full text-center text-xs text-neutral-500 hover:text-neutral-300"
        >
          Cancelar
        </button>
      </div>
    </div>
  );
}
