# Ancient Analytics

Bem-vindo ao repositório oficial do **Ancient Analytics**, uma plataforma modular avançada projetada para elevar o nível competitivo de jogadores de Dota 2. Através da extração de dados e inteligência artificial, o sistema processa estatísticas em tempo real para ditar o meta e prever resultados de partidas antes mesmo delas começarem.

O projeto intercepta as decisões cruciais antes e durante o jogo, unificando desde a análise preditiva do draft até o cruzamento espacial do comportamento dos jogadores.

---

## Objetivo e Funcionalidades

O sistema foi arquitetado para abrigar um guarda-chuva de ferramentas essenciais sob uma única plataforma centralizada:

1. **Motor de IA para Matchups e Partidas:** Um modelo de aprendizado de máquina treinado com milhares de partidas em ranks altos para analisar as interações entre os heróis (matchups de "bonecos"). Ele identifica quem tem a vantagem estatística em rotas e prever as chances de vitória de composições inteiras.
2. **Dashboard de Tendências do "Meta":** Mapeamento constante de taxas de vitória, escolhas e banimentos cruzados por *Ranks* do jogo para determinar heróis e itens dominantes em cada patch.
3. **Analisador de Draft e Sinergia em Tempo Real:** Ferramenta interativa que consome os dados do motor de IA e exibe a probabilidade dinâmica de vitória durante a tela de escolhas (Draft).
4. **Avaliador de Desempenho e Curva de Aprendizado (Player Profiler):** Análise isolada de contas, rastreando médias de impacto em ouro/experiência e evolução do jogador.
5. **Heatmaps Espaciais e Análise de Rotações:** Plotagem visual de eventos no minimapa do Dota 2 utilizando dados de replays para estudo de controle de território.

---

## Arquitetura do Sistema (Microserviços)

O **Ancient Analytics** adota o padrão de **Microsserviços**. Isso permite separar o processamento rápido em memória (em Go) dos cálculos pesados de Machine Learning (em Python).

- **API Gateway (Go):** O maestro do sistema. Atua como um proxy reverso que roteia as requisições do front-end e garante a proteção do sistema via Heartbeats nativos (Service Registry em memória).
- **Meta Service (Go):** Responsável por consumir e armazenar em cache os dados crus das APIs externas (heróis, itens, winrates globais).
- **Draft Service (Go):** Gerencia as sessões de escolha em tempo real através de WebSockets.
- **AI Matchup Service (Python):** O cérebro preditivo. Um microserviço isolado que recebe uma composição de heróis do Gateway, processa no modelo de regressão/classificação, e devolve a probabilidade de vitória e os melhores *counters*.

---

## Tech Stack

A escolha de tecnologias foi rigorosamente selecionada para lidar com pesadas requisições de rede, alta concorrência nas respostas web e processamento avançado de dados preditivos.

### Inteligência Artificial & Data Science (AI Matchup Service)
Para treinar a IA capaz de prever os matchups e desfechos de partidas, utilizamos o padrão ouro da ciência de dados:
*   **Python:** A linguagem base do microserviço analítico.
*   **FastAPI:** Expondo o modelo de Machine Learning através de uma API REST ultrarrápida para que o backend em Go consiga consumi-la instantaneamente.
*   **Pandas & NumPy:** Para a manipulação, limpeza e estruturação do gigantesco *dataset* de partidas extraído da API.
*   **XGBoost / Scikit-Learn:** Algoritmos de aprendizado de máquina (*Gradient Boosting* e *Random Forest*) ideais para encontrar padrões vencedores no cruzamento de atributos e sinergias de heróis. 
*   **Integração LLM (Opcional):** Uso da API do Gemini/OpenAI para traduzir números frios da IA em dicas em texto natural (Ex: *"Como jogar de Anti-Mage contra Axe neste patch"*).

### Backend Central
*   **Golang (Go):** A espinha dorsal do sistema web. A concorrência nativa (goroutines) suporta milhares de conexões WebSocket para o *Analisador de Draft*.
*   **Go Padrão (`net/http` e `sync`):** Orquestração e cache em memória (RAM) nativos, substituindo o Redis para máxima leveza via *Multi-stage builds* no Docker.

### Frontend
*   **Next.js (React) + TypeScript:** Combina renderização no servidor (SSR) para SEO no *Dashboard do Meta* e renderização no cliente (CSR) para a reatividade da tela de *Draft*.
*   **HTML5 Canvas API / Deck.gl:** Usado diretamente no navegador para desenhar fluidamente as coordenadas espaciais (Heatmaps) sobre o mapa oficial do jogo a 60 FPS.
*   **Recharts:** Plotagem estatística flexível para renderizar curvas de progressão do jogador.

### Banco de Dados
*   **PostgreSQL / ClickHouse:** O motor de armazenamento do *dataset* da IA. Responsável por salvar milhões de *match_ids* históricos e realizar agregações analíticas (OLAP) em frações de segundo.

### Integração Externa de Dados (Data Sources)
- **OpenDota API (REST):** Focada na coleta em massa para popular o banco de dados da IA (IDs de partidas, heróis, itens) e nas métricas globais de rank.
- **STRATZ API (GraphQL):** Utilizada sob demanda para consultar históricos isolados de jogadores na ferramenta *Player Profiler*.

---