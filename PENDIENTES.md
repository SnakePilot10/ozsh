# Pendientes

## Infraestructura de deploy

No hay infraestructura real declarada para staging o produccion. Los scripts `scripts/deploy-staging.sh` y `scripts/deploy-prod.sh` aceptan comandos via variables de entorno y hacen skip seguro si no estan configuradas.

## Branch protection

El repositorio remoto es privado y `main` no aparece protegida. La API de GitHub respondio que la proteccion avanzada requiere GitHub Pro o repo publico para este caso. `GIT_WORKFLOW.md` documenta la configuracion manual cuando este disponible.

## Coverage

El gate actual se mantiene en 70% porque el estado documentado del proyecto es 70.7%. Objetivo recomendado: subir gradualmente a 80%.

## Healthcheck HTTP

No se implementa endpoint `/health` porque `ozsh` no es servidor HTTP. Se implementa healthcheck CLI mediante `scripts/healthcheck.sh`.

## Graphify

El clon limpio no trae `graphify-out/` porque esta ignorado. Si se requiere grafo persistente en esta copia, ejecutar `/graphify` o copiar el grafo vigente antes de usar `graphify update .`.

## Docker local

El Dockerfile y `docker-compose.yml` quedaron configurados, pero en el entorno de implementacion actual no se pudo validar `docker build` porque el usuario no tiene permiso sobre `/var/run/docker.sock`. CI debe validar el build con `docker-build`.
