# Estado actual

Fecha de pausa: 2026-06-12.

## Resumen

`ozsh` esta en estado release-candidate local. El CLI principal, generador,
presets, headers, plugins manuales, logging, empaquetado base y TUI Bubble Tea
ya existen.

El proyecto esta listo para una validacion externa de v1.0. La suite local,
coverage, build, seguridad, performance y smoke test pasan. Quedan tareas que
requieren entorno externo: clean install en Ubuntu/macOS/Termux reales,
auditoria de issues del repo remoto y checksums finales de paquetes tras cortar
el tag.

## Funcional

- `ozsh preview` y `ozsh preview --real`
- `ozsh apply`
- `ozsh reset`
- `ozsh doctor`
- `ozsh theme list|preview|apply`
- `ozsh header list|preview|apply`
- `ozsh plugin list|add|remove|enable|disable|trust|untrust`
- `ozsh tui`
- `ozsh version`
- `ozsh update --check`
- `ozsh update`

## TUI

La TUI incluye dashboard, builder, preview editable, apply con diff y
confirmacion, doctor con fixes basicos, temas, headers y plugins. En Termux usa
preview ligero y el builder permite desactivar segmentos pesados con `h`.

## Seguridad de plugins

Los plugins solo se sourcean si estan:

- configurados
- habilitados
- marcados como `trusted`
- apuntando a un path bajo `$HOME`
- declarando un `load` relativo `.zsh` o `.sh`
- presentes como archivo regular readable en tiempo de shell
- no son symlinks

Aun asi, sourcear plugins significa ejecutar codigo de terceros en Zsh. La
confianza sigue siendo una decision explicita del usuario.

## Tests

Comando:

```bash
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test ./...
```

Estado actual: pasa al 100%, incluyendo restore bit-perfect cuando el `.zshrc`
original no tenia salto final.

Coverage:

```bash
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -coverprofile=/tmp/ozsh_coverage.out ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go tool cover -func=/tmp/ozsh_coverage.out
```

Estado actual: 70.7% total.

## Build, seguridad y performance

- Build local: `go build -buildvcs=false ./...` pasa.
- Smoke local: `scripts/release-smoke.sh` pasa.
- Seguridad: `govulncheck ./...` reporta 0 vulnerabilidades llamadas por el codigo.
- Performance: `BenchmarkRunApply` reporta ~0.73ms/op, debajo del objetivo de 100ms.
- Binary size local: 7.2MB con TUI, debajo del limite de 20MB.

## Pendientes importantes

- Validar instalacion de usuario nuevo en menos de 5 minutos.
- Auditar issues criticos reales en GitHub cuando haya repo remoto/credenciales.
- Sustituir placeholders de Homebrew/AUR por tag y SHA256 reales al publicar.
- Ejecutar `goreleaser release --snapshot --clean` en un entorno con GoReleaser instalado.
