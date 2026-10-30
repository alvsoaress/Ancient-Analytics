# Frontend — Ancient Analytics

React, TypeScript e Vite. A direção visual adotada é Arena / Dados em destaque em azul aço.

## Etapas de implementação

| Etapa | Entrega | Estado |
|---|---|---|
| 1 | Base visual, landing page, retratos oficiais, navegação e adaptação para celular | Implementada |
| 2 | Catálogo completo e meta com dados reais, filtros e ordenação | Implementada |
| 3 | Busca por ID de amigo e perfil público do jogador | Implementada |
| 4 | Histórico paginado e acesso às partidas | Implementada |
| 5 | Detalhes da partida, times e desempenho por jogador | Implementada |
| 6 | Revisão integrada de carregamento, erros, acessibilidade e responsividade | Implementada |
| 7 | Busca na home e páginas de builds por herói, com itens, habilidades e talentos | Implementada |

Cada etapa deve passar pela compilação, análise de código e verificação dos seus fluxos no navegador antes de avançar.

## Executar

O frontend consome a API local. Primeiro inicie o backend, na pasta `backend`:

```bash
go run ./cmd/api
```

Depois, nesta pasta:

```bash
npm install
npm run dev
```

O Vite exibe o endereço local no terminal, normalmente `http://localhost:5173`. Ele repassa as requisições de `/api` para `http://127.0.0.1:8080`, pois o backend não envia cabeçalhos CORS. O mesmo vale para `npm run preview`. Sem o backend, as seções de meta e heróis exibem uma mensagem de erro com a opção de tentar novamente.

```bash
npm run build
npm run lint
npm test
```

## Organização e responsabilidades

```text
src/
├── App.tsx                    # Página inicial, partida ou builds, conforme a URL; carrega o catálogo de heróis
├── App.css                    # Layout e adaptação aos tamanhos de tela
├── index.css                  # Tokens de cor, fontes e regras globais de acessibilidade
├── api.ts                     # Tipos da API, requisições, estados de carregamento e URL dos retratos
├── players.ts                 # Conversão do ID informado e nome das medalhas
├── matches.ts                 # Tipos do histórico e da partida, nomes de modos, filas e lados, formato da duração
├── navigation.ts              # Navegação sem recarregar e leitura de parâmetros da URL
├── heroes.ts                  # Busca normalizada, validação do herói e endereços das builds
├── builds.ts                  # Contrato de builds, formatação de tempos e resumo da amostra
└── components/
    ├── Header.tsx             # Navegação e menu móvel
    ├── StatsPreview.tsx       # Destaque do topo: herói mais escolhido no cenário profissional
    ├── MetaSection.tsx        # Meta profissional: posição, ordenação e mínimo de escolhas
    ├── HeroCatalog.tsx        # Catálogo completo, busca e filtro por atributo
    ├── PlayerSection.tsx      # Busca e perfil público do jogador
    ├── PlayerMatches.tsx      # Histórico paginado de partidas do jogador
    ├── MatchView.tsx          # Detalhes da partida: placar, times e desempenho por jogador
    ├── HeroBuildView.tsx      # Página de builds, posição, sequência selecionada e amostra
    ├── HeroBuildView.css      # Estilos da busca e das builds usando os tokens existentes
    ├── BuildPanels.tsx        # Itens iniciais, progressão completa, habilidades e talentos
    ├── BuildGuidePanels.tsx   # Inventário final, alternativas e histórico completo de compras
    ├── HeroSearch.tsx         # Busca no topo, sugestões e acesso direto à build
    ├── HeroLink.tsx           # Links de heróis com suporte a nova aba e navegação interna
    ├── HeroPortrait.tsx       # Retrato oficial e tratamento de imagem indisponível
    ├── RequestState.tsx       # Mensagens de carregamento e de erro, e o indicador de valor indisponível
    └── ErrorBoundary.tsx      # Mensagem de falha no lugar de uma tela em branco
public/
├── favicon.svg               # Marca vetorial usada na aplicação
└── fonts/                    # Fontes Barlow e Barlow Condensed (woff2) com licenças OFL
```

As cores da interface ficam como tokens em `index.css`; os estilos das builds reutilizam os mesmos tokens.

### Meta profissional

`MetaSection` consulta `GET /api/meta/heroes`, e a ordenação e o mínimo de escolhas são aplicados pelo backend.

- **Geral:** usa os totais profissionais da OpenDota, com mínimo inicial de 10 escolhas para que heróis pouco escolhidos não distorçam a taxa de vitória.
- **HC, Mid, Offlane, Sup 4 e Sup 5:** usam a amostra das 20 partidas profissionais mais recentes, com posições estimadas pela OpenDota. O mínimo inicial é 1 escolha.
- **Lista:** mostra os 10 primeiros heróis, com a opção de exibir todos.
- **Colunas:** taxa de escolha e taxa de vitória (com barras), KDA e escolhas. No celular, a coluna de escolhas fica oculta.
- **Taxa de escolha:** na visão Geral é estimada pelo backend, considerando dez escolhas por partida; nas posições usa o número exato de partidas da amostra.
- **KDA:** só aparece nas posições, porque o agregado geral da OpenDota não informa abates, mortes e assistências.
- **Valores ausentes:** taxas e KDA indisponíveis (`null`) aparecem como "—", diferente de zero.

### Consulta de jogador

`PlayerSection` consulta `GET /api/players/{accountId}`.

- **Entrada:** aceita o ID de amigo (Steam32), o SteamID64 e links `/profiles/<número>` da Steam ou `/players/<número>` da OpenDota e do Dotabuff. Tudo é convertido para o ID de amigo antes da consulta. Endereços personalizados da Steam (`/id/nome`) não podem ser convertidos sem consultar a Steam e são recusados com uma explicação.
- **URL:** a conta consultada fica em `?jogador=<ID>`, permitindo compartilhar o link e usar o botão voltar do navegador.
- **Perfil:** avatar, apelido, ID de amigo, SteamID64, links para a Steam e a OpenDota, medalha (`rankTier`: dezena = medalha, unidade = estrelas; Imortal não tem estrelas) e posição no ranking.
- **Valores ausentes:** apelido, medalha e ranking indisponíveis são informados como tal. HTTP `404` (`PLAYER_NOT_FOUND`) mostra que o perfil não está disponível na OpenDota, sem afirmar a causa; demais falhas oferecem nova tentativa.

### Histórico de partidas

Abaixo do perfil, `PlayerMatches` consulta `GET /api/players/{accountId}/matches` em páginas de 20 partidas.

- **Colunas:** herói (retrato oficial, pelo catálogo), resultado, K/D/A, duração, modo e fila, data e ID da partida. No celular, modo e ID ficam ocultos.
- **Navegação:** "Anteriores" e "Próximas" usam `offset`. "Próximas" segue `nextOffset` e fica desativado quando `hasMore` é `false`. Não há total de partidas, porque o backend não conta o histórico inteiro.
- **URL:** a página fica em `?pagina=N`, ao lado de `?jogador=ID`. Recarregar ou compartilhar o link mantém a página, e o botão voltar do navegador navega entre as páginas. A primeira página não usa o parâmetro, e uma nova busca de jogador volta para ela. Valores inválidos são tratados como a primeira página.
- **Modos e filas:** os códigos `gameMode` e `lobbyType` da OpenDota são traduzidos para os nomes do cliente em português. Códigos desconhecidos aparecem como "Modo N" ou "Fila N".
- **Valores ausentes:** resultado, K/D/A, duração e data indisponíveis aparecem como "—"; zero abates continua sendo 0.
- **Lista vazia:** a página informa que não há partidas públicas, sem afirmar se a conta não tem partidas ou não compartilha o histórico.
- **Partida:** "Detalhes" abre a página da partida na própria aplicação.

### Detalhes da partida

Com `?partida=<ID>` na URL, `App` mostra a página da partida no lugar das seções da página inicial, consultando `GET /api/matches/{matchId}`.

- **Cabeçalho:** vencedor ("Vitória dos Iluminados" ou "dos Temidos"), duração, início, modo e fila, placar e link para a OpenDota. Sem vencedor informado, o título indica resultado indisponível; ele não é deduzido pelo placar.
- **Times:** uma tabela para cada lado, com herói, apelido, nível, K/D/A, GPM, XPM, dano em heróis (com barra relativa ao maior dano da partida), dano em torres, cura e últimos golpes/negados. No celular, XPM, dano em torres, cura e últimos golpes ficam ocultos. Jogadores sem lado identificado aparecem num grupo separado.
- **Jogadores:** o apelido leva ao perfil do jogador na aplicação; contas anônimas aparecem como "Perfil anônimo".
- **Destaque:** aberta a partir do histórico, a URL também contém `jogador=<ID>`, e a linha desse jogador fica destacada.
- **Voltar:** retorna à página anterior da aplicação, mantendo a página do histórico. Num link aberto diretamente, leva ao perfil do jogador ou ao início.
- **Erros:** identificador inválido é recusado sem consultar a API; HTTP `404` informa que a partida não foi encontrada; demais falhas oferecem nova tentativa.
- **Navegação:** `navigation.ts` grava a URL com `history.pushState` e avisa os componentes pelo evento `popstate`, o mesmo usado pelo botão voltar do navegador.

### Carregamento, erros e acessibilidade

- **Carregamento:** ao trocar posição, ordenação ou mínimo de escolhas no meta, ou a página do histórico, os dados anteriores continuam visíveis e esmaecidos, com um aviso de atualização, até a nova resposta chegar (`previous` em `useApi`). Os botões de página ficam desativados nesse intervalo.
- **Erros:** cada seção trata sua própria falha e oferece nova tentativa; uma seção com erro não impede as demais. `ErrorBoundary` substitui a página por uma mensagem se um componente falhar ao exibir os dados.
- **Navegação:** o título da aba acompanha a tela (perfil do jogador ou partida). Ao abrir uma partida, o foco vai para a nova página. Na página da partida, os links do menu voltam à página inicial sem recarregar, rolando até a seção escolhida.
- **Acessibilidade:** tabelas com legenda e cabeçalhos de linha e coluna, regiões roláveis acessíveis pelo teclado, avisos de carregamento e erro anunciados por leitores de tela, link para pular ao conteúdo e respeito à preferência por menos movimento.
- **Verificação:** axe-core sem violações na página inicial, no perfil com histórico e na página da partida, em 320 px e 1440 px; sem rolagem horizontal da página em 320, 768, 1024 e 1440 px.

### Catálogo de heróis

`HeroCatalog` usa `GET /api/heroes`, ordenado por nome. A busca e o filtro por atributo principal são locais, sobre o catálogo completo. Funções do catálogo (`roles`) não são exibidas como posições reais do herói.

O topo da página (`StatsPreview`) usa `GET /api/meta/heroes?sortBy=picks&minPicks=1` e destaca o herói mais escolhido no agregado profissional: taxa de vitória no donut, taxa de escolha dos sete mais escolhidos nas barras, e vitórias e derrotas do herói em destaque.

## Imagens e referências

Usar sempre os retratos oficiais correspondentes ao ID de cada herói, com suas cores originais. O endereço é montado a partir do nome interno do herói no catálogo OpenDota (`npc_dota_hero_antimage` → `https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/antimage.png`), exposto pelo backend no campo `name` de `/api/heroes`. As imagens são carregadas da CDN Steam sob demanda. Se um retrato não carregar, a interface informa sua indisponibilidade e mantém o nome do herói, sem inventar personagens.

As fontes vêm de `google/fonts`, sob SIL Open Font License; as licenças estão junto dos arquivos. As pranchas e alternativas visuais não são versionadas. Referências colocadas em `frontend/design/` permanecem locais e são ignoradas pelo Git; os arquivos usados pela aplicação ficam em `public/`.


### Builds por herói

A página `/?heroi=<ID>` consulta `GET /api/heroes/{heroId}/builds`. `/?heroi=62&posicao=support4`, por exemplo, abre Bounty Hunter na posição de suporte 4. IDs inválidos e posições desconhecidas são recusados na interface sem consultar a rota de builds. URLs diretas, recarregamento e os botões voltar/avançar do navegador são suportados.

- **Entrada pela home:** o campo “Buscar build por herói” filtra o catálogo completo localmente, ignorando caixa, acentos, espaços e hífens. Apresenta até cinco sugestões com retrato e link para a página. Enter abre uma correspondência exata ou única; quando há várias opções, leva ao catálogo filtrado. Escape fecha as sugestões. Tab percorre os links nativos.
- **Catálogo:** a busca no topo e o filtro por nome no catálogo compartilham o texto durante a navegação nesta aplicação. O filtro por atributo continua disponível. Os retratos e nomes do catálogo, da tabela de meta e do destaque do topo abrem as builds. Ctrl/Cmd+clique e clique do meio mantêm o comportamento de links para abrir outra aba.
- **Posições:** Todas as posições, HC, Mid, Offlane, Suporte 4 e Suporte 5. A escolha fica na URL e corresponde ao parâmetro `position` do backend. Durante uma troca, os resultados anteriores deixam de ser exibidos, evitando apresentar dados da posição anterior como se fossem os novos; requisições obsoletas são canceladas.
- **Seleção:** inicialmente mostra até quatro sequências, na ordem de popularidade entregue pelo backend, com a opção de revelar todas. Cada botão mostra os itens, a quantidade de partidas e a taxa de vitória. A primeira sequência é a mais jogada da amostra, sem afirmar que é uma recomendação automática. O número de partidas acompanha a taxa de vitória.
- **Itens iniciais:** combinações exatas, quantidades, partidas e resultados, mostrando inicialmente até três inventários. Combinações adicionais ficam numa seção expansível. Inventário disponível sem compras anteriores ao início é diferente de informação indisponível.
- **Compras:** sequência dos itens identificados pelo backend e mediana de tempo, em minutos e segundos. Medianas fracionadas são arredondadas ao segundo mais próximo. A progressão não é cortada por quantidade de itens; é separada em início de jogo, itens principais e final de jogo. Componentes e consumíveis ficam no histórico completo. Os nomes permanecem visíveis se uma imagem falhar.
- **Inventário final:** mostra seis slots reais no encerramento das partidas, com seletor quando há inventários alternativos no grupo. Um espaço vazio é diferente de inventário indisponível. Não completa itens por suposição.
- **Situacionais e alternativas:** itens observados em outras progressões da mesma amostra, excluindo os itens já presentes na sequência selecionada. Mostra frequência e resultados; todas as opções podem ser expandidas.
- **Histórico completo:** seção expansível com todas as compras de uma partida real do grupo, inclusive componentes, receitas, consumíveis e repetições. Inclui link da partida na OpenDota.
- **Habilidades:** imagens na ordem de todas as melhorias registradas, legenda com nomes e seletor quando há sequências alternativas. Os números representam a ordem de melhoria, sem presumir o nível em que pontos guardados foram usados. Habilidades desconhecidas mantêm seu ID.
- **Talentos:** opções por nível e os resultados das escolhas efetivamente observadas na build. “Não observado nesta amostra” não é tratado como uma taxa de vitória igual a zero. Opções mais observadas são identificadas pelos contadores e pelo destaque visual. Quando o nome fornecido contém um marcador de valor não resolvido, a interface preserva o nome legível e informa que o valor não foi fornecido; não publica o marcador nem inventa um número.
- **Imagens:** retrato, itens e habilidades usam as URLs da CDN Steam fornecidas pela API. Falhas mostram um marcador com rótulo acessível, preservando os nomes. Não há ícones ilustrativos substituindo itens reais.
- **Amostra:** mostra quantidade de partidas, período, atualização, abrangência de patches e contagens de exclusão. O agrupamento usa a sequência completa de itens de uso/montados, sem limite de três compras ou preço mínimo de 1.000 de ouro. As fases seguem os tempos de compra. Estatísticas de milhares de partidas e filtro por patch não estão disponíveis nesta etapa.
- **Estados:** carregamento explícito, falha com “Tentar novamente”, herói não encontrado e ausência de builds com acesso às demais posições. A falha da seção de meta não bloqueia o catálogo nem a busca.
- **Navegação acessível:** título da aba por herói, foco na página ao abrir outro herói, botões com estado selecionado, links nativos e rótulos nos controles. O menu e o rodapé retornam às seções da home.

Os testes em `tests/*.test.mjs` usam o executor nativo do Node.js (22.18 ou superior para ler os arquivos TypeScript), sem dependências adicionais. Cobrem a busca normalizada, IDs e posições inválidos, URLs compartilháveis, tempos e contagem de exclusões. `npm run build` valida os contratos TypeScript e produz a versão de distribuição; `npm run lint` executa as regras existentes do projeto.
