# URL-Shortener-on-Go
# 1. Features:

- API: Quick link shortening via JSON POST requests.

- Web UI: Modern user interface (HTML/JS).

- Console UI: Management and statistics directly in the terminal (`stat`, `top` commands).

- Validation: Link verification for correctness and prevention of self-citation.

- Statistics: Tracking clicks, IP addresses, and visitor User-Agents.

# 2. Technical details:

- Go: Clean code without heavy external frameworks.

- Concurrent Storage: Thread-safe storage in RAM (In-Memory).

- Vite/Vanilla JS: For frontend (`static` folder).

# 3. Console:

- `stat` — general statistics.

- `top 5` — top 5 popular links.

- `just paste the URL` — shortens the link directly in the console.

# 4. Project structure:

- `main_block/` — entry point (main.go).

- `internal/api/` — HTTP server logic and handlers.

- `internal/storage/` — data storage logic and click counter.

- `internal/console/` — interactive console interface.

- `internal/utils/` — code generation and URL validation.

- `static/` — frontend files (HTML, CSS, JS).
