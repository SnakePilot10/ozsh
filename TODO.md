# ozsh — TODO de Producción

> Estado actual: pausa de implementación; CLI/TUI beta funcional con suite local verde
> Meta: v1.0 estable con TUI, temas y plugins manuales
> Nota: `go test ./...` pasa actualmente al 100%; ver `STATUS.md`.

---

## Fase 1: Núcleo CLI Estable (v0.1 → v0.2)

**Prioridad: CRÍTICA — Bloquea todo lo demás**

- [x] **Tests unitarios para el generador**
  - [x] Test que `Generate()` produce Zsh sintácticamente válido
  - [x] Test que segmentos deshabilitados no aparecen en output
  - [x] Test que `show_success = true` incluye ✓ en éxito
  - [x] Test que `show_success = false` omite ✓ en éxito
  - [x] Test que `git` segmento no aparece fuera de repo (simulado)
  - [x] Test de idempotencia: correr `Generate()` dos veces da mismo output

- [x] **Tests para shell/zshrc.go**
  - [x] Test `InjectBlock()` en `.zshrc` vacío
  - [x] Test `InjectBlock()` en `.zshrc` con bloque existente (reemplazo, no duplicado)
  - [x] Test `RemoveBlock()` elimina solo el bloque ozsh
  - [x] Test `RemoveBlock()` preserva contenido original
  - [x] Test `HasBlock()` detecta bloque presente/ausente
  - [x] Test backup se crea con timestamp correcto

- [x] **Tests para config/io.go**
  - [x] Test `Load()` crea config por defecto si no existe
  - [x] Test `Load()` lee config existente correctamente
  - [x] Test `Save()` escribe TOML válido y legible
  - [x] Test manejo de permisos (directorio no escribible, etc.)

- [x] **Manejo de errores robusto**
  - [x] Validar que colores en `fg` sean valores Zsh válidos (cyan, red, #00f5ff)
  - [x] Validar que `order` no contenga segmentos inexistentes
  - [x] Validar que `order` no tenga duplicados
  - [x] Mensajes de error traducidos o al menos consistentes
  - [x] Códigos de salida correctos (0 éxito, 1 error genérico, 2 error config)

- [x] **Logging**
  - [x] Flag `--verbose` / `-v` para debug output
  - [x] Log de operaciones en `~/.config/ozsh/ozsh.log`
  - [x] Rotación de logs (máximo 5MB, 3 archivos)

---

## Fase 2: Generador Mejorado (v0.2)

**Prioridad: ALTA — Base para temas y personalización**

- [x] **Soporte de colores hexadecimales**
  - [x] Convertir `#00f5ff` a escape ANSI para preview
  - [x] Generar código Zsh para colores hex (usar `truecolor` si terminal lo soporta)
  - [x] Fallback a colores named si terminal no soporta truecolor
  - [x] Validación de formato hex (#RRGGBB)

- [x] **Segmentos adicionales (MVP extendido)**
  - [x] `host` — nombre de máquina (`%m`)
  - [x] `venv` — virtualenv activo (`$VIRTUAL_ENV`)
  - [x] `node` — versión de Node.js (solo si `package.json` presente)
  - [x] `go` — versión de Go (solo si `go.mod` presente)
  - [x] `battery` — nivel de batería (Linux/Termux)
  - [x] `jobs` — jobs en background (`%j`)

- [x] **Configuración de separadores básicos**
  - [x] Campo `separator = "  "` (espacios por defecto)
  - [x] Campo `separator = " | "` opcional
  - [x] No powerline todavía, solo strings simples

- [x] **Right prompt (RPROMPT)**
  - [x] Soporte para `right_prompt = true`
  - [x] Segmentos marcados para derecha en config
  - [x] Generar `RPROMPT` en lugar de `PROMPT` para esos segmentos

- [x] **Templates opcionales**
  - [x] Directorio `templates/` con `omega.zsh.tmpl`
  - [x] Motor de templates mínimo (Go `text/template`)
  - [x] Fallback a generador Go si template no existe

---

## Fase 3: TUI con Bubble Tea (v0.3)

**Prioridad: ALTA — Experiencia visual principal**

- [x] **Estructura Bubble Tea**
  - [x] Modelo raíz con estado global
  - [x] Sistema de navegación entre pantallas (tabs)
  - [x] Manejo de teclas globales (q salir, ? ayuda, tab cambiar pantalla)

- [x] **Pantalla Dashboard**
  - [x] Status general (config válida, bloque presente, backups disponibles)
  - [x] Quick actions: Apply, Preview, Doctor, Reset
  - [x] Info del sistema (versión ozsh, plataforma detectada)

- [x] **Pantalla Prompt Builder**
  - [x] Lista de segmentos con checkboxes (Bubbles `list` + `checkbox`)
  - [x] Toggle con espacio o enter
  - [x] Reordenar con `j/k` o flechas + `shift+j/k`
  - [x] Panel de propiedades al seleccionar segmento (color, bold, icono)
  - [x] Preview en tiempo real (abajo o lateral)

- [x] **Pantalla Preview**
  - [x] Preview simulado grande (viewport completo)
  - [x] Contexto editable (username, cwd, git branch, exit status)
  - [x] Modo "preview real" (opcional, v0.3.1)

- [x] **Pantalla Apply**
  - [x] Diff visual de cambios (qué cambió en config)
  - [x] Confirmación antes de modificar `.zshrc`
  - [x] Spinner durante generación
  - [x] Resultado: éxito / error con detalle

- [x] **Pantalla Doctor**
  - [x] Checks visuales con íconos ✓/✗/⚠
  - [x] Recomendaciones automáticas ("falta zsh, instálalo con...")
  - [x] Botón "fix" para problemas auto-resolvibles

- [x] **Estilos con Lip Gloss**
  - [x] Tema base oscuro (fondo #09090d, texto #e0e0e0)
  - [x] Acento cyan (#00f5ff) para elementos activos
  - [x] Bordes redondeados donde sea posible
  - [x] Layout responsive (paneles laterales vs apilados)

---

## Fase 4: Temas y Presets (v0.4)

**Prioridad: MEDIA — Diferenciación visual**

- [x] **Sistema de presets**
  - [x] Archivos `presets/*.toml` (cyber-cyan, neon-red, matrix-green, etc.)
  - [x] Comando `ozsh theme list`
  - [x] Comando `ozsh theme apply <name>`
  - [x] Preview de tema antes de aplicar

- [x] **Estructura de tema**
  ```toml
  [theme]
  name = "neon-red"
  accent = "#ff003c"
  background = "#09090d"
  muted = "#6b6b80"
  success = "#00ff9f"
  warning = "#ffe600"
  error = "#ff003c"
  ```

- [x] **Temas en TUI**
  - [x] Galería de temas con preview en miniatura
  - [x] Aplicar tema desde TUI
  - [x] Tema personalizado (guardar como preset nuevo)

- [x] **Temas para la propia TUI**
  - [x] Tema oscuro (default)
  - [x] Tema claro
  - [x] Tema que respete colores del terminal

---

## Fase 5: Plugins Manuales (v0.5)

**Prioridad: MEDIA — Expansión funcional**

- [x] **Motor manual**
  - [x] `ozsh plugin add <url>` — clona a `~/.config/ozsh/plugins/`
  - [x] `ozsh plugin remove <name>`
  - [x] `ozsh plugin list`
  - [x] `ozsh plugin enable/disable <name>`

- [x] **Generación de sources**
  - [x] Iterar `plugins.items` en config
  - [x] Generar `source` lines en `omega.zsh`
  - [x] Validar que archivos existen antes de sourcear
  - [x] Orden de carga respetado

- [x] **Plugins en TUI**
  - [x] Lista de plugins con estado (enabled/disabled)
  - [x] Formulario para añadir nuevo plugin (URL + archivo a sourcear)
  - [x] Validación de URL y descarga

---

## Fase 7: Termux Polish (v0.7)

**Prioridad: MEDIA — Plataforma clave**

- [x] **Detección mejorada**
  - [x] Detectar si zsh es shell por defecto en Termux
  - [x] Detectar si `termux-chroot` está activo
  - [x] Rutas correctas para prefix de Termux

- [x] **Optimizaciones Termux**
  - [x] Reducir uso de memoria en preview (celulares limitados)
  - [x] Opción para desactivar segmentos pesados (git en repos grandes)
  - [x] Soporte para teclado virtual (touch-friendly en TUI)

- [x] **Instalación en Termux**
  - [x] `install.sh` detecta Termux automáticamente
  - [x] Instala dependencias si faltan (`pkg install zsh git`)
  - [x] No intenta `chsh` (no funciona en Termux)
  - [x] Documentación específica para Termux en README

---

## Fase 8: Estabilidad y v1.0

**Prioridad: ALTA — Requisito para "producción"**

- [x] **Tests de integración**
  - [x] Script de CI (GitHub Actions) que corre tests en:
    - [x] Ubuntu latest
    - [x] macOS latest
    - [x] Termux (emulado o Docker)
  - [x] Test end-to-end: `ozsh apply` → abrir zsh → ver prompt correcto
  - [x] Test de `ozsh reset` → `.zshrc` vuelve a estado anterior

- [x] **Documentación**
  - [x] README completo con GIFs/screencasts
  - [x] Documentación de config.toml (todos los campos)
  - [x] Guía de migración desde omega-zsh-python
  - [x] Guía de contribución (CONTRIBUTING.md)
  - [x] Changelog automatizado (conventional commits)

- [x] **Empaquetado**
  - [x] Releases automáticos en GitHub (goreleaser)
  - [x] Binarios para: Linux amd64/arm64, macOS amd64/arm64, Android/Termux arm64
  - [x] Fórmula de Homebrew
  - [x] PKGBUILD para AUR (Arch Linux)
  - [x] Script de instalación one-liner (`curl | bash`)

- [x] **Seguridad**
  - [x] Validar que paths no escapan de `$HOME` (no `../../../etc/passwd`)
  - [x] Validar URLs de plugins (scheme https solo)
  - [x] No ejecutar código arbitrario de plugins
  - [x] Sanitizar input de usuario en TUI

- [x] **Performance**
  - [x] Benchmark de arranque: `ozsh apply` debe tardar < 100ms
  - [x] Memoria: TUI no debe usar más de 50MB
  - [x] Binary size: < 20MB con TUI, < 5MB sin TUI (opcional)

- [x] **UX final**
  - [x] Mensajes de error claros y accionables
  - [x] Sugerencias contextuales ("¿quisiste decir 'ozsh apply'?")
  - [x] Comando `ozsh version`
  - [x] Comando `ozsh update` (auto-update)
  - [x] Notificaciones de nueva versión

---

## Bloqueantes para v1.0

| # | Item | Por qué bloquea |
|---|------|-----------------|
| 1 | Tests unitarios del generador | Sin esto, cualquier cambio puede romper prompts |
| 2 | Tests de integración | Sin esto, no sabemos si funciona en Zsh real |
| 3 | Manejo de errores robusto | Usuario no debe quedarse con `.zshrc` roto |
| 4 | Backup y restore funcionales | Reversibilidad es promesa central del proyecto |
| 5 | Documentación completa | v1.0 implica que otros pueden usarlo sin ayuda |

---

## No-Goals (mantener fuera del scope)

- ❌ Gestor de plugins Oh My Zsh / Antidote / Zinit (solo manual)
- ❌ Powerline separators en v1.0 (evaluar para v2.0)
- ❌ Cambio automático de shell (`chsh`)
- ❌ Soporte para Bash / Fish (solo Zsh)
- ❌ Instalación de dependencias del sistema (zsh, git, go)
- ❌ Conexión a internet para temas/plugins (todo local por defecto)

---

## Métricas de éxito para v1.0

- [x] `go test ./...` pasa al 100%
- [x] Coverage > 70%
- [x] Binary compila en Linux, macOS, Termux sin advertencias
- [ ] Usuario nuevo puede instalar y tener prompt funcionando en < 5 minutos
- [x] `ozsh reset` deja `.zshrc` exactamente como estaba (bit-perfect restore)
- [ ] 0 issues críticos abiertos en GitHub
- [x] README tiene al menos 3 ejemplos de prompts reales

---

*Última actualización: 2026-06-15*
*Próxima revisión: antes de cortar tag v1.0.0*
