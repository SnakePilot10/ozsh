# Git Workflow

## Branches

- `main`: produccion. Debe estar protegida y recibir cambios solo por PR aprobado.
- `develop`: integracion. Los pushes a esta rama disparan deploy automatico a staging.
- `feature/*`: desarrollo de features y mejoras no urgentes.
- `hotfix/*`: correcciones urgentes desde `main`.
- `release/*`: estabilizacion previa a tags `v*`.

## Conventional Commits

Formato obligatorio:

```text
tipo(scope opcional): descripcion imperativa corta
```

Tipos aceptados:

- `feat`: nueva funcionalidad.
- `fix`: correccion de bug.
- `docs`: documentacion.
- `refactor`: cambio interno sin alterar comportamiento.
- `test`: tests.
- `chore`: mantenimiento.
- `ci`: CI/CD.
- `build`: build, packaging o dependencias.
- `perf`: performance.

Ejemplos:

```text
feat: add prompt theme preset
fix(shell): preserve zshrc newline during reset
ci: add security scan job
```

## Proceso de feature

1. Actualiza `main` local.
2. Crea una rama `feature/<nombre>`.
3. Implementa cambios pequenos y revisables.
4. Ejecuta `scripts/lint.sh --check`, `scripts/test.sh`, `scripts/build.sh` y `scripts/healthcheck.sh`.
5. Ejecuta `graphify update .` si `graphify-out/graph.json` existe.
6. Crea un commit con Conventional Commits y pushea la rama.
7. Abre PR hacia `develop` si existe; si no, hacia `main` hasta crear `develop`.
8. Espera CI verde y revision antes de mergear.

## Proceso de release

1. Crea `release/vX.Y.Z` desde `develop`.
2. Ejecuta `scripts/test.sh`, `scripts/build.sh`, `scripts/healthcheck.sh`.
3. Actualiza `CHANGELOG.md` y packaging si corresponde.
4. Abre PR de `release/vX.Y.Z` a `main`.
5. Tras merge, crea tag `vX.Y.Z`.
6. El workflow `release` ejecuta GoReleaser.

## Proceso de hotfix

1. Crea `hotfix/<descripcion>` desde `main`.
2. Corrige el problema con el minimo cambio seguro.
3. Ejecuta validacion completa.
4. Abre PR hacia `main`.
5. Tras merge, propaga el fix a `develop`.

## Proteccion de main

Estado verificado con `gh`: el repositorio remoto es privado y `main` no aparece protegida. La API de branch protection devuelve limitacion de plan para repos privados sin GitHub Pro o repo publico.

Cuando la plataforma lo permita, configura manualmente en GitHub:

1. Settings -> Branches -> Add branch protection rule.
2. Branch name pattern: `main`.
3. Activar `Require a pull request before merging`.
4. Activar `Require approvals` con minimo 1 aprobacion.
5. Activar `Require status checks to pass before merging`.
6. Checks requeridos:
   - `validate`
   - `security-scan`
   - `docker-build`
7. Activar `Require branches to be up to date before merging`.
8. Activar `Do not allow bypassing the above settings` si el plan lo permite.
9. Restringir push directo a admins o deshabilitarlo por completo.

## Reglas operativas

- No pushear directo a `main`.
- No commitear `.env`, tokens, certificados ni secrets.
- No usar `git push --force` salvo aprobacion explicita.
- No omitir CI para cambios de codigo, packaging o scripts.
- No modificar `.zshrc` en tests sin HOME temporal.
