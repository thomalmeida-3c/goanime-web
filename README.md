# goanime-web

MVP que expõe o [GoAnime](https://github.com/alvarorichard/GoAnime) (CLI de streaming de anime em Go) como uma API HTTP, com um frontend Next.js para consumi-la.

## Estrutura

- `backend/` — fork local do repositório GoAnime com um `cmd/server` novo que expõe busca/episódios/stream via HTTP, mais pequenos patches no `pkg/goanime` (veja "Patches" abaixo).
- `frontend/` — Next.js 16 + TypeScript + Tailwind, consome a API do backend.

## Rodando localmente

### Backend

```bash
cd backend
go run ./cmd/server
# escuta em :8080 (ADDR=":porta" para mudar)
```

### Frontend

```bash
cd frontend
npm install   # se ainda não rodou
cp .env.example .env.local  # já tem o valor padrão certo
npm run dev
# abre em http://localhost:3000
```

## Endpoints do backend

- `GET /api/search?q=<nome>` — busca em todas as fontes (AnimeFire, Goyabu, SuperFlix, AniDB).
- `GET /api/episodes?url=<animeUrl>&source=<Source>` — lista episódios.
- `GET /api/stream?episodeUrl=<url>&source=<Source>&quality=best&mode=sub` — resolve a URL de stream e devolve `playbackUrl` (um endpoint local, `/api/proxy/:id`, que faz o proxy do vídeo já com os headers/token que o provedor exige).
- `GET /api/proxy/:id` — proxy do vídeo (streaming, suporta `Range`, reescreve manifests HLS quando aplicável).
- `GET /api/home` — fileiras estilo Crunchyroll para a home (em alta / populares da temporada / mais populares), com pôster, banner, nota e link direto pro anime quando disponível. Ver "Home" abaixo.

## Home (fileiras "em alta" / "populares", estilo Crunchyroll)

O GoAnime (e os scrapers que ele usa) não tem nenhum dado próprio de popularidade — não sabemos quantas pessoas assistiram o quê. Pra ter fileiras tipo Crunchyroll, `/api/home` busca os rankings reais no [AniList](https://anilist.co) (API GraphQL pública, sem chave): `TRENDING_DESC` para "Em alta", `POPULARITY_DESC` filtrado pela temporada atual para "Populares da temporada", e `POPULARITY_DESC` geral para "Mais populares".

A home **não** tenta casar cada título com uma fonte (Goyabu/SuperFlix/...) de antemão — isso exigiria dezenas de buscas ao vivo só pra carregar a tela inicial. Em vez disso `/api/home` devolve só metadados do AniList (título, pôster, banner, sinopse, nota, gêneros), o que é rápido (~1s, cacheado 1h). Clicar num card — ou no hero — dispara uma busca normal em `/api/search?q=<título>`, exatamente como digitar na busca, e o usuário escolhe entre os resultados das fontes igual ao fluxo de busca manual.

O hero (`HeroCarousel.tsx`) roda um carrossel com os 6 primeiros itens de "Em alta", com setas, indicadores e avanço automático a cada 7s.

## Upscaling em tempo real (WebGL2)

O player (`UpscaledVideoPlayer.tsx`) aplica um upscale+nitidez em tempo real no navegador — não é o upscaler do GoAnime (`internal/upscaler`), que é um processo offline em lote (extrai todo frame como PNG via ffmpeg, roda Anime4K, reencoda — minutos por episódio) feito pra arquivos baixados, incompatível com um vídeo que está sendo *streamado* ao vivo através do nosso proxy.

Como funciona: o `<video>` real fica invisível (só decodifica e toca o áudio); a cada frame novo (`requestVideoFrameCallback`, com fallback pra `requestAnimationFrame`) ele é enviado como textura pra um shader WebGL2 (`lib/upscaleGL.ts`) que desenha num `<canvas>` maior que a resolução original — a amostragem bilinear da própria GPU faz o upscale, e um passo de nitidez com máscara de borda (unsharp mask que só realça onde há contraste local, pra não amplificar ruído/banding) recupera a definição que o upscale simples borra. É uma aproximação de passe único do efeito visual do Anime4K, não um port do algoritmo original (que é multi-passo).

Como o canvas cobre o vídeo, os controles nativos do navegador somem junto — por isso o player tem sua própria barra (play/pause, seek, mudo, tela cheia, e o toggle "Upscaling"). Sem WebGL2 (raro hoje em dia), cai automaticamente pro `<video controls>` normal.

**Pegadinha resolvida**: como o backend/frontend rodam em portas diferentes (8080/3000), o `<video>` precisa de `crossOrigin="anonymous"` pra o WebGL poder ler os frames como textura — sem isso o navegador lança `SecurityError` mesmo com o proxy já mandando `Access-Control-Allow-Origin: *`.

## O que funciona hoje

- **Goyabu**: busca, episódios e stream **funcionam de ponta a ponta**, incluindo a resolução de vídeos hospedados no Blogger (o backend replica o fluxo `batchexecute` que o GoAnime CLI usa antes de entregar a URL ao mpv). Testado com "Sousou no Frieren".
- **SuperFlix**: busca e episódios funcionam, mas a resolução de stream do GoAnime exige um navegador Firefox headed para resolver um desafio Cloudflare — pesado demais para este MVP, não foi ligado ao `/api/stream`.
- **AnimeFire**: o scraper do GoAnime não está retornando resultados no momento (parece ser um problema do próprio site/scraper upstream, não deste projeto).
- **AniDB**: a API upstream respondeu 503 durante os testes.

Isso é uma limitação dos scrapers de cada fonte (sites de terceiros com proteção anti-bot), não do backend em si — a infraestrutura de proxy/HLS já criada funciona para qualquer fonte que devolva uma URL de stream direta.

## Patches aplicados ao GoAnime (em `backend/`)

O `pkg/goanime` público do GoAnime só expunha a fonte AnimeFire no enum `types.Source` (comentário no código já avisava disso). Como a maioria dos resultados reais vem de Goyabu/SuperFlix/AniDB, foram feitos 3 ajustes pequenos e cirúrgicos:

1. `pkg/goanime/types/source.go` — adicionado `SourceGoyabu`, `SourceSuperFlix`, `SourceAniDB` ao enum público (o motor interno já suportava essas fontes).
2. `pkg/goanime/client.go` — `scraperKind()` não mapeava `AniDBType`; adicionado.
3. `internal/player/export_web.go` (novo arquivo) — exporta `ResolveBloggerStream()`, um wrapper fino sobre a lógica de resolução de Blogger já existente no player do CLI, para o `cmd/server` poder reusá-la sem duplicar código.

## Limitações conhecidas do MVP

- O proxy de Blogger (`internal/player`) é um singleton global — só um stream Blogger ativo por vez no backend. Ok para uso local de uma pessoa, não para múltiplos usuários simultâneos.
- Sem autenticação, sem persistência, sem histórico/continuar-assistindo.
- SuperFlix e AniDB não têm stream funcional ligado ainda (ver acima).
