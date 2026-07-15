# Production Loop

## Objetivo

El loop de produccion automatiza el camino desde una rama `feature/*` hasta una rama publicada lista para PR. El humano sigue controlando el diseno, la revision, el merge y la decision de versionar; la maquina ejecuta gates repetibles y publica artefactos cuando todo pasa.

## Uso rapido

```bash
git checkout -b feature/my-change
scripts/production-loop.sh "feat: describe the change"
```

El mensaje debe cumplir Conventional Commits.

## Que automatiza

`scripts/production-loop.sh` ejecuta:

1. Verificacion de repositorio Git valido.
2. Verificacion de branch `feature/*`.
3. Validacion del mensaje Conventional Commit.
4. `scripts/lint.sh --check`.
5. `scripts/test.sh`.
6. `scripts/build.sh`.
7. `scripts/healthcheck.sh`.
8. `graphify update .` si `graphify` esta instalado y existe `graphify-out/graph.json`.
9. `git add -A`.
10. `git commit -m "..."`.
11. `git push -u origin <branch>`.
12. Sugerencia de `gh pr create` con base `develop` si existe, si no `main`.

Despues de validar que todo pasa, el script hace commit y push automaticamente a la branch `feature/*` correspondiente.

## Intervencion humana

El humano debe intervenir en estos puntos:

- Elegir el alcance del cambio.
- Escribir el mensaje Conventional Commit.
- Revisar el diff antes de ejecutar el loop si el cambio es sensible.
- Crear o revisar el PR.
- Aprobar y mergear el PR.
- Crear un tag `v*` cuando el cambio debe convertirse en release instalable.
- Configurar branch protection y secrets de deploy en GitHub.
- Confirmar deploy manual de produccion cuando se usa `scripts/deploy-prod.sh` fuera de CI.

## Acciones destructivas

El loop no borra ramas, no fuerza pushes, no resetea historial y no toca `main`. Cualquier operacion destructiva debe hacerse fuera del loop y con aprobacion explicita.

## Graphify

Si el repositorio tiene `graphify-out/graph.json`, el loop actualiza el grafo antes del commit. Si el clon no tiene grafo, el loop lo documenta y continua. Para crear el grafo en un clon nuevo, ejecuta `/graphify` o `graphify .` segun tu entorno.

## Fallos comunes

- Branch no es `feature/*`: crea una rama feature.
- Coverage menor a 70%: agrega tests o ajusta el cambio.
- `golangci-lint` no instalado: ejecuta `scripts/setup.sh`.
- GitHub no tiene `develop`: el loop sugerira PR contra `main`.
- Deploy no hace nada: configura `DEPLOY_STAGING_COMMAND` o `DEPLOY_PROD_COMMAND` como secrets.

## Relacion con CI/CD

El loop local reduce fallos antes del PR. GitHub Actions vuelve a ejecutar los gates en un entorno limpio:

- `validate`
- `security-scan`
- `docker-build`
- `release` en tags `v*`
- `deploy-staging`
- `deploy-production`

El merge solo debe ocurrir con CI verde.

## Release real

Para publicar una version descargable por usuarios:

```bash
git checkout main
git pull --ff-only origin main
git tag v1.0.0
git push origin v1.0.0
```

El tag dispara GoReleaser en GitHub Actions. El release resultante contiene binarios para Linux, macOS y Android, mas `checksums.txt`. Para `ozsh`, este es el despliegue principal de produccion; `DEPLOY_PROD_COMMAND` queda como extension opcional para servidores o mirrors externos.
