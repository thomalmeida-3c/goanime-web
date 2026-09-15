# Deploy do frontend (Vercel)

Next.js puro, zero config — o Vercel detecta tudo sozinho. `vercel` CLI já
está instalado globalmente (`npm install -g vercel`).

## 1. Login (você precisa fazer isso — abre o navegador)

```bash
vercel login
```

## 2. Deploy

Rode dentro de `frontend/` (importante: esse repo é um monorepo com
`backend/` e `frontend/` lado a lado — o comando precisa rodar de dentro de
`frontend/`, não da raiz):

```bash
cd frontend
vercel        # primeira vez: pergunta nome do projeto, confirma o diretório
```

Isso cria um deploy de preview. Pra produção:

```bash
vercel --prod
```

## 3. Variável de ambiente (essencial)

O frontend só sabe a URL do backend via `NEXT_PUBLIC_API_URL` (default,
localhost:8080, só serve pra dev local). Configure a URL de produção do
backend (Fly.io — ver `backend/DEPLOY.md`) **antes** do build de produção,
porque é uma env var `NEXT_PUBLIC_*` (embutida no bundle do navegador em
build time, não lida em runtime):

```bash
vercel env add NEXT_PUBLIC_API_URL production
# cole: https://goanime-web-backend.fly.dev  (ou o nome que você escolheu)
```

Depois de adicionar a env var, rode `vercel --prod` de novo pra rebuildar
com o valor certo.

## Deploys seguintes

`vercel --prod` (dentro de `frontend/`) redeploya com o código atual.

## Alternativa: auto-deploy via GitHub

Se preferir que cada `git push` faça deploy automático, suba este repo pro
GitHub e conecte no [dashboard do Vercel](https://vercel.com/new) — só
lembre de setar **Root Directory = `frontend`** nas configurações do projeto
(é um monorepo) e a mesma env var `NEXT_PUBLIC_API_URL` lá.
