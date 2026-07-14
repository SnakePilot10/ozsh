# Deployment

## Arquitectura

`ozsh` no es un servicio web persistente. Es un binario CLI/TUI distribuido por:

- instalador source-based (`install.sh`),
- artefactos GoReleaser,
- Homebrew,
- AUR,
- imagen Docker opcional para validacion y ejecucion aislada.

Por eso, el healthcheck de produccion es CLI-based: construir o resolver el binario, ejecutar `version`, `preview`, `apply` sobre un `HOME` temporal y validar el Zsh generado.

## Entornos

- Local: checkout del repo, Go 1.24+, scripts locales.
- Staging: rama `develop`; deploy parametrizado por `DEPLOY_STAGING_COMMAND` o `DEPLOY_TARGET_STAGING`.
- Produccion: rama `main` o tags `v*`; deploy parametrizado por `DEPLOY_PROD_COMMAND` o `DEPLOY_TARGET_PRODUCTION`.

## Variables

Consulta `.env.example`. Variables relevantes:

- `OZSH_INSTALL_DIR`: directorio de instalacion source-based.
- `OZSH_BIN_DIR`: destino del binario.
- `OZSH_REPO`: origen usado por `install.sh`.
- `GHCR_IMAGE`: imagen GHCR esperada.
- `DEPLOY_STAGING_COMMAND`: comando shell de deploy staging guardado como secret.
- `DEPLOY_PROD_COMMAND`: comando shell de deploy produccion guardado como secret.

No commitees `.env` reales.

## Deploy local

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```

## Deploy staging

1. Mergea un PR a `develop`.
2. GitHub Actions ejecuta `validate`, `security-scan`, `docker-build` y `deploy-staging`.
3. `scripts/deploy-staging.sh` ejecuta `DEPLOY_STAGING_COMMAND` si esta configurado.
4. `scripts/healthcheck.sh` valida el binario.

Deploy manual:

```bash
DEPLOY_STAGING_COMMAND='your deploy command' scripts/deploy-staging.sh
scripts/healthcheck.sh
```

## Deploy production

1. Mergea a `main` o crea un tag `v*`.
2. GitHub Actions ejecuta validacion, seguridad, Docker y deploy production.
3. `scripts/deploy-prod.sh` requiere confirmacion local o `OZSH_ASSUME_YES=1` en CI.
4. `scripts/healthcheck.sh` valida el binario.

Deploy manual:

```bash
DEPLOY_PROD_COMMAND='your deploy command' scripts/deploy-prod.sh
scripts/healthcheck.sh
```

## Rollback

- Binario instalado por source checkout: volver a un tag estable y reconstruir.
- GoReleaser/Homebrew/AUR: reinstalar version anterior publicada.
- Docker: volver a un tag anterior de GHCR.
- Config de usuario: `ozsh reset` elimina el bloque gestionado y conserva backups timestamped.

## Healthcheck

`scripts/healthcheck.sh` usa un `HOME` temporal y comprueba:

- `ozsh version`,
- `ozsh preview`,
- `ozsh apply`,
- existencia de `omega.zsh`,
- bloque gestionado en `.zshrc`,
- sintaxis Zsh cuando `zsh` esta disponible.

No se implementa `/health` HTTP porque el proyecto no expone servidor web.
