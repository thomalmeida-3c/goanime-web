# Deploy do backend (Fly.io)

O backend não pode ser serverless: guarda biblioteca/token do Mercado Livre em
arquivo JSON (precisa de disco que sobrevive a reinícios), mantém um resolver
de stream do Blogger como singleton global em memória, e serve vídeo via
streaming de longa duração. Fly.io roda o binário Go num processo persistente
de verdade, com disco (volume) de verdade, e tem região em São Paulo.

`flyctl` já está instalado (`brew install flyctl`). `Dockerfile`, `.dockerignore`
e `fly.toml` já estão prontos em `backend/`.

## 1. Login (você precisa fazer isso — abre o navegador)

```bash
fly auth login
```

Se ainda não tem conta, esse comando também deixa criar uma (pede cartão de
crédito, mas o uso aqui fica bem abaixo de US$3/mês — um shared-cpu-1x/256mb
rodando 24/7 em `gru`).

## 2. Criar o app e o volume (nome precisa ser único global no Fly)

```bash
cd backend
fly apps create goanime-web-backend   # troque o nome se já estiver em uso
fly volumes create goanime_data --region gru --size 1 -a goanime-web-backend
```

Se trocar o nome do app, atualize `app = "..."` em `backend/fly.toml` pra
bater com o nome escolhido.

## 3. Deploy

```bash
fly deploy
```

Isso builda o Dockerfile no builder remoto do Fly (não precisa Docker
instalado localmente) e sobe a máquina em `gru`.

## 4. Confirmar que subiu

```bash
fly status
curl https://goanime-web-backend.fly.dev/api/home
```

## 5. (Opcional, depois) Conectar o Mercado Livre em produção

```bash
fly secrets set ML_CLIENT_ID=... ML_CLIENT_SECRET=... \
  ML_REDIRECT_URI=https://goanime-web-backend.fly.dev/api/ml/callback
```

E cadastrar essa mesma `ML_REDIRECT_URI` como redirect URI autorizada na
aplicação do Mercado Livre Developers.

## Depois de rodando: aponte o frontend pra essa URL

No Vercel, defina a env var `NEXT_PUBLIC_API_URL=https://goanime-web-backend.fly.dev`
(ver `frontend/DEPLOY.md`).

## Deploys seguintes

Depois desse primeiro setup, `fly deploy` (dentro de `backend/`) é o único
comando necessário pra cada atualização.

## Custo e cold start

`min_machines_running = 1` em `fly.toml` mantém uma máquina sempre ligada —
sem cold start ao abrir o app no metro, mas custa ~US$2-3/mês rodando 24/7.
Pra economizar (aceitando um delay de ~1-2s na primeira requisição depois de
ficar parado), troque pra `auto_stop_machines = true` e `min_machines_running = 0`.
