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

## Deploy (produção)

Frontend no Vercel, backend no Fly.io — não dá pra hospedar o backend em
algo serverless (persistência em arquivo, singleton de stream em memória,
streaming de vídeo de longa duração exigem um processo Go persistente com
disco de verdade). Passo a passo em `backend/DEPLOY.md` e `frontend/DEPLOY.md`.

## Endpoints do backend

- `GET /api/search?q=<nome>` — busca em todas as fontes (AnimeFire, Goyabu, SuperFlix, AniDB).
- `GET /api/episodes?url=<animeUrl>&source=<Source>` — lista episódios.
- `GET /api/stream?episodeUrl=<url>&source=<Source>&quality=best&mode=sub` — resolve a URL de stream e devolve `playbackUrl` (um endpoint local, `/api/proxy/:id`, que faz o proxy do vídeo já com os headers/token que o provedor exige).
- `GET /api/proxy/:id` — proxy do vídeo (streaming, suporta `Range`, reescreve manifests HLS quando aplicável).
- `GET /api/home` — fileiras estilo Crunchyroll para a home (em alta / populares da temporada / mais populares), com pôster, banner, nota e link direto pro anime quando disponível. Ver "Home" abaixo.
- `GET /api/skip?animeName=<>&animeUrl=<>&episodeNum=<>` — tempos de abertura/encerramento (AniSkip), pra o botão "Pular abertura". Best-effort: sem match no AniList ou sem dado no AniSkip, devolve `{"op":null,"ed":null}` em vez de erro.
- `GET /api/manga/home` — mangás populares (MangaLivre + MangaLivre.blog, em paralelo). `GET /api/manga/latest` — mangás recém-adicionados (idem), pra fileira da página dedicada de Mangás. `GET /api/manga/search?q=` — busca (idem). `GET /api/manga/chapters?id=<>&source=<>` — lista de capítulos. `GET /api/manga/pages?id=<>&source=<>` — resolve as URLs das páginas pro leitor. Ver "Mangás" abaixo.
- `POST /api/auth/login` `{email}` — login sem senha (cria o usuário se não existir). `GET /api/library?email=` — lista a biblioteca salva. `POST /api/library` `{email, item}` — salva um anime/mangá. `DELETE /api/library?email=&kind=&refId=` — remove. Ver "Login e biblioteca" abaixo.
- `GET /api/manga/progress?email=&source=&mangaId=` — último capítulo aberto de um mangá (`{found, chapterId, chapter}`). `POST /api/manga/progress` `{email, source, mangaId, chapterId, chapter}` — salva. `GET /api/manga/progress/all?email=` — todo o progresso do usuário de uma vez, keyed `"source|mangaId"` (usado pelo mini perfil). Ver "Progresso de leitura" abaixo.
- `GET /api/products?q=<termo>` — busca produtos reais no Mercado Livre (precisa de OAuth conectado, ver "Produtos" abaixo). `GET /api/ml/status`, `GET /api/ml/connect`, `GET /api/ml/callback` — fluxo OAuth do Mercado Livre.

## Mangás (MangaLivre + MangaLivre.blog)

Home, "recém adicionados" e busca de mangá mostram **duas fontes em paralelo**: `mangalivre.go` (`mangalivre.to`) e `mangalivreblog.go` (`mangalivre.blog`) — dois sites diferentes, sem relação entre si além do nome parecido (nem o mesmo tema: `.to` é Madara/WordPress, `.blog` é um tema próprio com outras classes CSS). Adicionado porque **cada um tem títulos que o outro não tem** — motivo real: o `.to` não tinha Vinland Saga, o `.blog` tinha. `fanMangaLivre` (`manga_handlers.go`) dispara os dois em paralelo e junta os resultados; se um site cair ou mudar o HTML, o outro continua funcionando (erro de um lado não derruba a resposta).

Os dois são scrapers de verdade (`goquery`), mas **bem mais simples** que qualquer fonte de anime: sem Cloudflare, sem sessão/token, `curl` puro já traz o HTML completo. A lista de capítulos vem inteira renderizada numa página só (sem paginação pra perseguir), e as imagens são arquivos estáticos no próprio domínio — sem precisar de proxy nem headers especiais, ao contrário do vídeo de anime. Cada fonte usa a própria listagem de popularidade do site (`?m_orderby=views` no `.to`, `?ordem=popular` no `.blog`) como ranking real, não um número inventado.

**Adicionar uma terceira fonte** segue o mesmo molde: um arquivo novo tipo `mangalivreblog.go` (client HTTP + parser goquery + as 5 funções search/popular/latest/chapters/pages, um `Source` string próprio pra identificar de onde veio), somar no `fanMangaLivre` dos 3 handlers de listagem, e um novo `case` nos dois `switch source` de `handleMangaChapters`/`handleChapterPages`. Cada site tem sua própria estrutura de HTML — inspecionar com `curl` antes de escrever o parser é mais rápido que adivinhar (os dois MangaLivre, apesar do nome, precisaram de seletores CSS completamente diferentes).

**Busca unificada**: `/api/search` (anime) e `/api/manga/search` (mangá) rodam em paralelo a cada busca; o frontend guarda os resultados por tipo e um filtro (Todos/Animes/Mangás) só troca o que é exibido. Os resultados de mangá são reordenados por relevância ao termo buscado (`rankMangaByRelevance`) — sem isso, uma obra derivada podia aparecer acima da série principal só por ter rankeado melhor na busca da fonte.

### MangaDex (código ainda existe, só não está ligado na listagem)

`mangadex.go` continua funcional — [API pública, gratuita, sem chave](https://api.mangadex.org). Foi a primeira fonte que integramos, mas apanha em títulos **licenciados oficialmente** (Boruto, Jujutsu Kaisen, ...): o MangaDex geralmente não tem nenhum capítulo de fato hospedado em pt-br pra esses, só stubs `externalUrl` apontando pro site oficial (que filtramos, já que não levam a nenhuma página real) — na prática, obra sem nenhum capítulo pra ler. `MangaItem.Source` já existe pra distinguir as fontes, e `/api/manga/chapters` e `/api/manga/pages` já sabem rotear por `source=MangaDex` — religar é trocar a chamada em `handleMangaHome`/`handleMangaSearch` de volta (ou somar as duas, como o `/api/search` de anime já faz com Goyabu+SuperFlix).

**Leitor** (`MangaReader.tsx`): paginado/lateral (uma página por vez, navega com as setas do teclado ou clicando nas metades esquerda/direita da imagem) — diferente do leitor padrão do MangaDex, que é scroll vertical infinito. Mais simples de implementar também: lista de páginas fixa, sem rastrear posição de scroll ou lazy-load.

## Pular abertura/encerramento (AniSkip)

Igual ao CLI, mas via HTTP: nenhuma das nossas fontes (Goyabu/SuperFlix/...) expõe o ID do MyAnimeList, que é a chave que a API do [AniSkip](https://aniskip.com) usa. Então `/api/skip` primeiro resolve o MAL id fazendo a mesma busca por título no AniList que o CLI já fazia (`internal/api.enrichAnimeData`, exportada como `EnrichAnimeMetadata` em `export_web.go`), e só então consulta o AniSkip.

O frontend dispara essa busca em paralelo com a resolução do stream (não bloqueia o play — o botão "Pular abertura ⏭" só aparece quando/se os dados chegam, e some quando o `currentTime` sai do intervalo). Testado com Kimetsu no Yaiba: pulou de 1:14 direto pra 2:33, batendo com os timestamps reais do AniSkip.

## Home (fileiras "em alta" / "populares", estilo Crunchyroll)

O GoAnime (e os scrapers que ele usa) não tem nenhum dado próprio de popularidade — não sabemos quantas pessoas assistiram o quê. Pra ter fileiras tipo Crunchyroll, `/api/home` busca os rankings reais no [AniList](https://anilist.co) (API GraphQL pública, sem chave): `TRENDING_DESC` para "Em alta", `POPULARITY_DESC` filtrado pela temporada atual para "Populares da temporada", e `POPULARITY_DESC` geral para "Mais populares".

A home **não** tenta casar cada título com uma fonte (Goyabu/SuperFlix/...) de antemão — isso exigiria dezenas de buscas ao vivo só pra carregar a tela inicial. Em vez disso `/api/home` devolve só metadados do AniList (título, pôster, banner, sinopse, nota, gêneros), o que é rápido (~1s, cacheado 1h). Clicar num card — ou no hero — dispara uma busca normal em `/api/search?q=<título>`, exatamente como digitar na busca, e o usuário escolhe entre os resultados das fontes igual ao fluxo de busca manual.

O hero (`HeroCarousel.tsx`) roda um carrossel com os 6 primeiros itens de "Em alta", com setas, indicadores e avanço automático a cada 7s.

## Upscaling em tempo real (WebGL2)

O player (`UpscaledVideoPlayer.tsx`) aplica um upscale+nitidez em tempo real no navegador — não é o upscaler do GoAnime (`internal/upscaler`), que é um processo offline em lote (extrai todo frame como PNG via ffmpeg, roda Anime4K, reencoda — minutos por episódio) feito pra arquivos baixados, incompatível com um vídeo que está sendo *streamado* ao vivo através do nosso proxy.

Como funciona: o `<video>` real fica invisível (só decodifica e toca o áudio); a cada frame novo (`requestVideoFrameCallback`, com fallback pra `requestAnimationFrame`) ele é enviado como textura pra um shader WebGL2 (`lib/upscaleGL.ts`) que desenha num `<canvas>` maior que a resolução original — a amostragem bilinear da própria GPU faz o upscale, e um passo de nitidez com máscara de borda (unsharp mask que só realça onde há contraste local, pra não amplificar ruído/banding) recupera a definição que o upscale simples borra. É uma aproximação de passe único do efeito visual do Anime4K, não um port do algoritmo original (que é multi-passo).

Como o canvas cobre o vídeo, os controles nativos do navegador somem junto — por isso o player tem sua própria barra (play/pause, seek, mudo, tela cheia, e o toggle "Upscaling"). Sem WebGL2 (raro hoje em dia), cai automaticamente pro `<video controls>` normal.

**Pegadinha resolvida**: como o backend/frontend rodam em portas diferentes (8080/3000), o `<video>` precisa de `crossOrigin="anonymous"` pra o WebGL poder ler os frames como textura — sem isso o navegador lança `SecurityError` mesmo com o proxy já mandando `Access-Control-Allow-Origin: *`.

## Interface: sidebar, busca com autocomplete, páginas de obra

A navegação (não só assistir/ler) segue o padrão Crunchyroll/MangaDex: uma sidebar fixa (`Sidebar.tsx`) com a marca e 4 seções — Início, Animes, Mangás, Produtos — substitui o header simples de antes. A busca (`SearchAutocomplete.tsx`, na `TopBar.tsx`) tem autocomplete com debounce de ~300ms, reaproveitando os mesmos `/api/search`/`/api/manga/search` já existentes — Enter ainda cai na tela de busca completa.

As páginas de obra (`DetailHeader.tsx`, usada tanto pra `view === "episodes"` quanto `"manga-chapters"`) seguem o estilo MangaDex: banner desfocado ao fundo (ou a própria capa desfocada quando não há banner, caso do MangaLivre), capa em primeiro plano, badges de fonte/gêneros/nota, botão de ação principal e um botão de salvar (biblioteca). Uma fileira compacta de "Produtos relacionados" (`RelatedProducts.tsx`) aparece embaixo, buscando pelo título da obra — fica silenciosa se a conta do Mercado Livre não estiver conectada.

Continua sendo uma SPA de state machine (o `view` no `page.tsx`), sem rotas de URL reais — decisão deliberada pra essa leva, não pendência.

## Mobile

Abaixo de `sm:` a `Sidebar` vira uma tab bar fixa no rodapé (ícone + label) em vez da rail vertical — libera a largura toda pro conteúdo, padrão de app mobile em vez de site desktop encolhido.

O leitor de mangá (`MangaReader.tsx`) ganha um modo imersivo no mobile: a topbar de busca some (não faz sentido no meio da leitura), o cabeçalho vira uma faixa fina, e a área da página usa `100dvh` menos a tab bar do rodapé. A imagem usa `object-contain` preenchendo a caixa de verdade (trocado de `max-height`, que só reduz — nunca ampliava uma página menor que o espaço disponível, deixando barras vazias). Zoom por gesto via [`react-zoom-pan-pinch`](https://github.com/BetterTyped/react-zoom-pan-pinch): pinça com dois dedos, duplo-toque alterna, arrasta quando ampliado. As zonas de toque "página anterior/próxima" são faixas finas nas bordas (15% de cada lado) em vez da metade da tela inteira — cobrindo a imagem toda elas engoliriam qualquer gesto de zoom antes dele chegar na lib; com zoom ativo elas se desligam (`pointer-events-none`) pra não brigar com o pan.

Botões de ação primários (Assistir/Ler/Continuar no `DetailHeader`) ocupam 100% da largura no mobile (`w-full sm:w-auto`) com padding maior — alvo de toque adequado em vez de um botão pequeno perdido no meio da tela.

### Login e biblioteca

Login **sem senha**: só o email, guardado no `localStorage` do navegador (`goanime:email`) e repassado em toda chamada de biblioteca — não tem cookie/sessão de verdade, é o suficiente pra reconhecer o mesmo usuário entre visitas num app pessoal. `POST /api/auth/login` cria o usuário na primeira vez. A biblioteca (animes/mangás salvos) persiste em `backend/data/users.json` (arquivo JSON com mutex, path configurável via `DATA_DIR`; **nunca commitado**, está no `.gitignore` porque tem emails).

### Progresso de leitura

"Onde você parou" por mangá — o último capítulo aberto, não uma posição de página/scroll. Salvo automaticamente (best-effort, não bloqueia a leitura) toda vez que um capítulo é aberto, keyed por `source|mangaId` em `User.Progress` (mesmo `users.json`, sem tabela nova). Na página da obra isso vira um botão "Continuar — Cap. X" (em vez de "Ler", que sempre iria pro capítulo 1) e o capítulo correspondente fica destacado em roxo na grade.

A grade de capítulos também tem um toggle de ordenação (menor→maior / maior→menor) — puramente no frontend (`sortChapters` em `page.tsx`), já que o backend sempre devolve em ordem de leitura. Números de capítulo com casa decimal (ex. `"108.100"`, splits de release) são parseados via `parseFloat`.

### Mini perfil

Clicar no avatar/email na topbar abre `ProfilePage.tsx` (em vez do login, quando já logado): animes e mangás salvos em duas fileiras, com o selo "Parou no Cap. X" nos cards de mangá que têm progresso, e um botão de remover por item. Abrir um card a partir do perfil pula a busca — vai direto pra lista de episódios/capítulos, já que um `LibraryItem` carrega `refId`+`source` suficientes pra isso (mesmo truque de reentrada que os cards da home usam).

### Produtos (Mercado Livre)

A busca pública do Mercado Livre (`/sites/MLB/search`, `/products/search`) **exige OAuth** — confirmado ao vivo (403 sem token), diferente do que a doc antiga sugeria. `ml_oauth.go` implementa o fluxo `authorization_code` completo: `GET /api/ml/connect` redireciona pro consentimento, `GET /api/ml/callback` troca o código por `access_token`/`refresh_token` (renovado automaticamente) e persiste em `backend/data/ml_token.json`. Sem conectar, `/api/products` devolve `{connected: false, products: []}` — a `ProdutosPage` mostra as instruções de setup nesse caso, sem erro.

Pra conectar: crie uma aplicação em [Mercado Livre Developers](https://developers.mercadolivre.com.br), defina `ML_CLIENT_ID`, `ML_CLIENT_SECRET` e `ML_REDIRECT_URI` (ex.: `http://localhost:8080/api/ml/callback`) no ambiente do backend, e acesse `/api/ml/connect`. O link de afiliado (`buildAffiliateLink`) é um passthrough até a env var `ML_AFFILIATE_TAG` existir — o formato exato do programa de afiliados do ML não foi inventado, fica documentado aqui como pendência até termos a conta.

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
- Login sem senha/verificação e persistência em JSON (não SQL) — ok pro porte de um app pessoal. Biblioteca salva (lista), mas sem rastrear progresso por episódio/capítulo ("continuar assistindo/lendo").
- SuperFlix e AniDB não têm stream funcional ligado ainda (ver acima).
