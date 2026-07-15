## Proyecto

`ozsh` es una aplicacion Go CLI/TUI para construir, previsualizar, generar y aplicar prompts Zsh declarativos.

Stack:

- Go 1.24+.
- Bubble Tea, Bubbles y Lip Gloss para TUI.
- TOML para configuracion.
- GoReleaser, Homebrew y AUR para packaging.
- GitHub Actions para CI/CD.

## Reglas para agentes

- Para preguntas de arquitectura o relaciones, usa graphify si `graphify-out/graph.json` existe: `graphify query "<pregunta>"`.
- Despues de modificar codigo o documentacion, ejecuta `graphify update .` si graphify esta disponible y el grafo existe.
- No hardcodees secrets, tokens, rutas privadas ni credenciales.
- No pushees directo a `main`.
- No uses `git push --force` sin aprobacion explicita.
- No cambies comportamiento de `.zshrc` sin actualizar o ejecutar smoke tests.
- Toda prueba que toque HOME, `.zshrc` o plugins debe usar un HOME temporal.
- Mantener Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `ci:`, `build:`.
- Preferir cambios pequenos y verificables.
- Si hay incertidumbre, documentarla en `PENDIENTES.md`.

## Comandos de validacion

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```
