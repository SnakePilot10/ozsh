# Analisis del Proyecto

Fecha de analisis: 2026-07-14

## Resumen ejecutivo

`ozsh` es una aplicacion CLI/TUI escrita en Go para construir, previsualizar, generar y aplicar prompts declarativos de Zsh. El proyecto ya tiene una base de produccion avanzada: tests unitarios, smoke tests, empaquetado con GoReleaser, instalador, workflows de GitHub Actions y documentacion de release. El trabajo pendiente es consolidar ese estado en un loop de produccion reproducible con calidad local, CI/CD, flujo Git, scripts operativos, Docker opcional y documentacion completa.

## Stack tecnologico

- Lenguaje principal: Go.
- Version declarada: `go 1.24.2`.
- Toolchain: `go1.24.4`.
- Modulo: `github.com/snakepilot10/ozsh`.
- Gestor de dependencias: Go modules (`go.mod`, `go.sum`).
- UI terminal: Bubble Tea, Bubbles y Lip Gloss.
- Configuracion: TOML via `github.com/BurntSushi/toml`.
- Shell objetivo: Zsh.
- Packaging: GoReleaser, Homebrew formula y AUR PKGBUILD.

## Arquitectura actual

- `cmd/ozsh/main.go`: punto de entrada principal del CLI.
- `internal/config`: carga, guardado y validacion de configuracion TOML.
- `internal/prompt`: generacion y preview del prompt Zsh.
- `internal/shell`: deteccion del entorno, manejo de `.zshrc`, backups y paths.
- `internal/plugins`: gestion de plugins manuales y reglas de seguridad.
- `internal/tui`: interfaz Bubble Tea.
- `internal/logging`: logging local con rotacion.
- `presets`: temas y presets de prompt.
- `templates`: plantillas opcionales para generacion.
- `packaging`: artefactos Homebrew y AUR.
- `docs`: checklist de release, screencasts y planes.
- `scripts`: automatizaciones existentes de smoke install/release y release metadata.

## Tests y calidad existentes

- Framework de testing: `go test` nativo.
- Tests existentes: archivos `*_test.go` en `cmd/ozsh` e `internal/*`.
- Coverage gate existente: 70% en GitHub Actions.
- Smoke tests existentes:
  - `scripts/release-smoke.sh`
  - `scripts/install-smoke.sh`
- Build actual: `go build -buildvcs=false ./...`.
- Seguridad documentada: `govulncheck ./...`.
- Performance documentada: `BenchmarkRunApply` objetivo menor a 100ms/op.

## CI/CD existente

Workflows actuales:

- `.github/workflows/test.yml`: tests en Ubuntu/macOS, coverage gate, build, smoke scripts y cross-build Android/Termux.
- `.github/workflows/release.yml`: release por tags `v*` con GoReleaser.
- `.github/workflows/changelog.yml`: generacion manual de changelog con git-cliff.

Estado remoto verificado con `gh`:

- Repositorio: `SnakePilot10/ozsh`.
- Visibilidad: privado.
- Rama por defecto: `main`.
- Permisos del usuario: `ADMIN`.
- Workflows activos: `test`, `release`, `changelog`.
- Ultimo workflow `test` en `main`: exitoso.
- Branch protection: no activa; GitHub API indica que la proteccion avanzada requiere repo publico o GitHub Pro para este repo privado.

## Docker y runtime

- No habia `Dockerfile`.
- No habia `docker-compose.yml`.
- El runtime principal no es un servicio HTTP: es un binario CLI/TUI.
- El healthcheck adecuado para produccion es un smoke check CLI con HOME temporal, no un endpoint `/health`.

## Variables de entorno

No hay `.env` real ni variables obligatorias para ejecutar el binario. Variables relevantes actuales:

- `OZSH_REPO`: repo o path usado por el instalador.
- `OZSH_INSTALL_DIR`: directorio del checkout instalado.
- `OZSH_BIN_DIR`: destino del binario.
- `OZSH_YES`: modo no interactivo del instalador.
- `OZSH_APPLY`: aplicar configuracion tras instalar.
- `OZSH_UPDATE_PATH`: actualizar PATH en `.zshrc`.
- `TERMUX_VERSION` y `PREFIX`: deteccion de Termux.

## Base de datos

No aplica. El proyecto no usa base de datos.

## Estado Git local y remoto

La copia original en `/home/snake/Proyectos/Go/ozsh` tenia `.git/` vacio y `git status` fallaba. Para evitar un loop de produccion sobre un repo local corrupto, se preparo un clon limpio en `/tmp/opencode/ozsh-clean` y se trabaja en la branch `feature/production-loop`.

## Riesgos e incertidumbres

- El deploy real no tiene infraestructura declarada; los scripts de deploy deben ser parametrizados por variables de entorno y seguros por defecto.
- La proteccion real de `main` no puede aplicarse via API en este repo privado sin GitHub Pro o repo publico; queda documentada como paso manual.
- El healthcheck HTTP solicitado en plantillas genericas no aplica a este proyecto; se implementa healthcheck CLI.
- El objetivo de coverage ideal es 80%, pero el gate actual se conserva en 70% para no romper el estado release-candidate. Se documenta subirlo gradualmente.
