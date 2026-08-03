# Starship, Transparent TUI, and Extensible Plugins Design

Date: 2026-08-02
Status: Approved for planning
Repository: `SnakePilot10/ozsh`

## 1. Purpose

This design defines the next two implementation phases after the fullscreen contextual TUI merged in PR #21.

The work solves three related problems:

1. The TUI uses gray or tinted background blocks that interrupt the visual language and look especially poor in Termux.
2. The Plugins screen exposes only the curated catalog and does not provide a discoverable path for adding other Zsh plugins.
3. Native prompt themes remain structurally limited, so ozsh needs an optional Starship engine without abandoning the current native engine or breaking existing users.

The work is split into two pull requests so each change remains reviewable and testable.

- PR A: transparent TUI chrome and complete plugin management UX.
- PR B: optional Starship prompt engine with independent configuration and contextual themes.

PR B starts only after PR A is merged.

## 2. Product decisions

The following decisions are final for this design:

- The native prompt engine remains the default and fallback.
- Starship is activated manually from the Prompt screen.
- Native and Starship settings are stored independently and restored when switching engines.
- Themes are contextual to the active engine instead of mixing both engines in one list.
- Starship installation offers both `Install now` and `Show instructions`.
- Plugin management includes both a curated catalog and `Add custom plugin`.
- Custom plugin repositories are cloned first, candidate load files are detected, and the user explicitly selects the correct file.
- The TUI chrome uses a transparent terminal background. Selection and focus rely on markers, borders, typography, and accent color instead of filled gray rectangles.

## 3. Scope and pull request boundaries

### 3.1 PR A: TUI and plugin workflow

PR A includes:

- removing explicit background colors from TUI chrome;
- updating active tabs, focused panels, selected rows, badges, previews, and status regions;
- adding a visible custom-plugin action to the Plugins screen;
- adding a multi-step custom-plugin wizard;
- staging plugin downloads safely;
- detecting and ranking candidate `.plugin.zsh`, `.zsh`, and `.sh` files;
- requiring explicit trust confirmation before activation;
- exposing enable, disable, change load file, remove trust, and remove actions;
- preserving the existing curated catalog;
- adding regression tests for transparent rendering, responsive layout, staging, rollback, and plugin lifecycle operations.

PR A does not change the prompt engine or configuration schema.

### 3.2 PR B: Starship engine

PR B includes:

- adding `native` and `starship` prompt engines;
- migrating configuration while preserving existing native settings;
- adding engine selection to the Prompt screen;
- detecting and optionally installing Starship;
- generating an ozsh-managed Starship TOML file;
- adding Starship presets and engine-specific theme browsing;
- rendering live previews through the Starship binary when available;
- updating Review & Apply and the managed Zsh block;
- preserving native settings when Starship is active and preserving Starship settings when native is active;
- adding platform, migration, generation, preview, and shell-integration tests.

PR B does not replace or deprecate the native engine.

## 4. PR A architecture

### 4.1 Transparent visual system

The TUI must inherit the terminal background. No ozsh chrome component may emit a background ANSI sequence.

This applies to:

- the outer panel;
- tabs;
- preview boxes;
- selected rows;
- badges;
- status boxes;
- plugin cards;
- modal and wizard chrome.

Prompt output may still contain background colors because prompt styling is user content rather than TUI chrome.

The semantic palette remains, but `Surface` and `Panel` stop being used as backgrounds. They may be removed if no foreground or border use remains.

Selection uses all of the following when space allows:

- a leading `›` marker;
- bold text;
- accent foreground;
- a focused border for the active pane.

Compact layouts may omit the pane border marker only when necessary to preserve content width, but the `›` marker and bold accent text remain.

Active tabs use accent foreground, bold text, and underline. They do not use a background fill.

Badges render as plain bracketed text, for example:

- `[installed]`
- `[trusted]`
- `[native]`
- `[starship]`

Preview regions keep borders and padding, but their background remains transparent.

### 4.2 Visual invariants

The following invariants are testable requirements:

- Rendering the TUI chrome must not contain ANSI SGR background codes in the `40-49`, `100-107`, `48;2`, or `48;5` families.
- Prompt preview content is excluded from this assertion because a prompt theme may intentionally contain a background.
- Focus must remain identifiable in monochrome output.
- Active navigation must not depend on color alone.
- Width and height guarantees from the fullscreen layout remain unchanged.

### 4.3 Plugin screen structure

The Plugins screen has two explicit sections:

```text
Recommended plugins
  › [x] Autosuggestions
    [ ] fzf-tab
    [x] Syntax highlighting

Custom plugins
    zsh-extract              [active]
    history-substring-search [disabled]

[a] Add custom plugin
```

The details pane shows the selected plugin's:

- source type: recommended or custom;
- repository URL when available;
- managed checkout path;
- selected load file;
- installed state;
- health state;
- trust state;
- enabled state;
- activation state;
- available actions.

When no custom plugins exist, the section shows a concise empty state and the `Add custom plugin` key remains visible.

### 4.4 Custom plugin wizard

The wizard is a finite state flow with explicit steps:

1. Repository URL
2. Clone progress
3. Candidate selection
4. Trust review
5. Pending summary

The wizard must remain keyboard-first and responsive below 72 columns.

#### Step 1: repository URL

The user enters an HTTPS repository URL.

Validation rules remain aligned with the existing plugin package:

- HTTPS only;
- no embedded credentials;
- no query string;
- no fragment;
- repository-derived plugin name must match the existing safe name pattern;
- duplicates are rejected before cloning.

#### Step 2: staged clone

The repository is cloned with `--depth 1` through a cancellable command with a fixed timeout.

The clone destination is a randomly named hidden staging directory inside the managed plugin root:

```text
~/.config/ozsh/plugins/.staging-<random>
```

Using the same parent filesystem allows finalization through an atomic rename.

The staging directory and plugin root use mode `0700` where supported.

Cancel, timeout, clone failure, TUI exit, or failed finalization removes the staging directory.

#### Step 3: candidate discovery

Candidate discovery scans regular files only. Symlinks are rejected.

Accepted suffixes:

- `.plugin.zsh`
- `.zsh`
- `.sh`

The scanner excludes `.git` and ignores obvious documentation, test, example, vendor, and generated directories where practical.

The scan has bounded depth and a maximum candidate count to avoid pathological repositories.

Candidates are ranked in this order:

1. root-level `<repository-name>.plugin.zsh`;
2. other root-level `*.plugin.zsh` files;
3. root-level `*.zsh` files;
4. root-level `*.sh` files;
5. nested candidates by the same suffix priority and then path depth.

The wizard never selects a candidate silently. Even when there is one candidate, the user confirms it.

If no candidate is found, the wizard explains the accepted file types and offers Back or Cancel. It does not activate the repository.

#### Step 4: trust review

Before activation, ozsh shows:

- repository URL;
- destination plugin name;
- relative load file;
- final managed path;
- a warning that shell plugins execute code in every interactive shell.

The user must explicitly confirm trust.

Downloading a repository does not imply trust. A plugin remains disabled and untrusted until confirmation.

#### Step 5: pending summary

The selected plugin is added to the in-memory pending configuration and the staged checkout remains inactive.

Final activation occurs only through Review & Apply. Apply performs:

1. revalidate staging directory and chosen load file;
2. atomically rename staging directory to the final managed directory;
3. update the plugin item source path;
4. persist configuration;
5. generate the managed Zsh file;
6. roll back the rename and configuration if persistence or generation fails.

A successful Apply removes any obsolete staging metadata.

### 4.5 Plugin lifecycle actions

For a custom plugin, the details pane exposes:

- enable or disable;
- trust or remove trust;
- change load file;
- remove plugin.

Changing the load file runs the same candidate scanner against the installed checkout. The new file must pass the existing path and symlink validation before it can be trusted.

Removing trust immediately makes the plugin inactive in the pending model.

Removing a plugin is staged until Apply. Apply uses a quarantine rename before saving configuration so a failed save can restore the checkout.

Recommended plugins keep curated ordering and metadata. They use the same status vocabulary as custom plugins.

### 4.6 PR A error handling

All long-running work runs outside `Update()` and `View()` through `tea.Cmd`.

Errors are shown in the active workflow and preserve enough state for retry or Back.

Required error cases include:

- Git unavailable;
- invalid URL;
- duplicate plugin;
- clone timeout;
- clone cancellation;
- repository too large for bounded candidate discovery;
- no candidate files;
- unsafe symlink;
- unreadable candidate;
- destination already exists;
- save failure;
- generated file failure;
- rollback failure.

Rollback failure is reported separately and must not be hidden behind the original error.

## 5. PR B architecture

### 5.1 Configuration model

PR B increments the configuration version from 2 to 3.

The logical model becomes:

```toml
version = 3

[prompt]
engine = "native"

[prompt.native]
# existing native prompt settings

[prompt.starship]
preset = "pure-preset"
```

Exact TOML field placement may be adjusted to preserve clean Go types, but these invariants are required:

- `prompt.engine` is either `native` or `starship`;
- existing version 2 prompt settings migrate into the native configuration without value loss;
- native and Starship configurations coexist;
- switching engines never overwrites the inactive engine's state;
- missing engine defaults to `native`;
- invalid engine values fail validation with a useful message.

The migration is deterministic and idempotent.

### 5.2 Engine abstraction

Prompt behavior is separated behind a small engine interface rather than spreading `if engine == ...` across the TUI.

The interface must cover:

- engine identifier and display name;
- availability detection;
- theme or preset catalog;
- preview rendering;
- generated configuration artifacts;
- managed Zsh initialization fragment;
- diagnostics for Doctor and Review & Apply.

The native implementation wraps the current generator and preview code.

The Starship implementation owns Starship-specific TOML generation, command execution, preset metadata, and installation guidance.

### 5.3 Prompt screen

The Prompt screen adds an `Engine` row above engine-specific settings.

Selecting it opens:

```text
Choose prompt engine

› Native
  Starship
```

Changing the selected engine updates the pending model and preview immediately but writes nothing until Apply.

When Starship is selected and unavailable, the screen offers:

- Install now
- Show instructions
- Keep Native

Keeping Native reverts only the pending engine selection. It does not delete stored Starship settings.

### 5.4 Starship availability and installation

Availability detection uses `exec.LookPath("starship")` and a short version probe with timeout.

Termux is detected using reliable environment and filesystem signals already used elsewhere in ozsh. The managed command is:

```sh
pkg install starship
```

For generic Linux, ozsh offers the official installer targeted to a user-writable directory such as `~/.local/bin` and avoids `sudo` by default.

The UI always offers `Show instructions` before executing an installer.

Installation commands are represented as structured plans rather than interpolated shell strings. Commands execute directly through `exec.CommandContext`.

After installation, ozsh repeats availability detection and reports the resolved executable and version.

Failure leaves the engine selection pending but inactive until the user chooses retry, instructions, or Native.

### 5.5 Starship configuration ownership

ozsh writes:

```text
~/.config/ozsh/starship.toml
```

The managed Zsh fragment includes:

```zsh
export STARSHIP_CONFIG="$HOME/.config/ozsh/starship.toml"
eval "$(starship init zsh)"
```

This fragment is emitted only when the active engine is Starship.

Switching back to Native removes Starship initialization from the generated ozsh block but preserves both the Starship binary and `~/.config/ozsh/starship.toml`.

The Starship file begins with an ozsh ownership comment and is generated deterministically.

### 5.6 Starship themes and presets

Themes are contextual to the active engine.

- Native engine: existing ozsh theme catalog.
- Starship engine: Starship preset catalog.

The initial Starship catalog includes a compact, reviewed set drawn from official presets and ozsh-maintained adaptations:

- Pure;
- Gruvbox Rainbow;
- Pastel Powerline;
- Tokyo Night;
- Catppuccin Powerline;
- Bracketed Segments;
- No Nerd Fonts.

Each entry contains:

- stable ozsh ID;
- display name;
- short description;
- Nerd Font requirement;
- source type: official preset or ozsh adaptation;
- deterministic TOML content or generation strategy.

The catalog is embedded so theme browsing does not require network access.

Selecting a Starship theme updates the pending Starship configuration only.

### 5.7 Starship preview

When Starship is available, the preview uses the real binary with a temporary generated config.

The command runner is injected behind an interface so tests use a fake executable.

The preview command receives controlled context including current directory, status, and jobs where supported. It has a short timeout and sanitized environment.

The temporary config is removed after each preview operation.

Preview results are cached by a hash of:

- Starship settings;
- preview context;
- terminal width;
- Starship executable version.

When Starship is unavailable, ozsh renders a clearly labeled sample preview and the message:

```text
Install Starship for a live preview
```

Sample previews must never be labeled live.

### 5.8 Review & Apply

Review & Apply includes engine-specific artifacts and actions.

For Starship it shows at minimum:

- active engine;
- selected preset;
- Starship availability and version;
- `~/.config/ozsh/config.toml`;
- `~/.config/ozsh/starship.toml`;
- the generated ozsh Zsh file;
- `.zshrc` managed block changes;
- whether Starship initialization is enabled or removed;
- confirmation that Native settings are preserved.

Apply remains transactional according to the existing ozsh save and rollback rules.

### 5.9 Doctor

Doctor adds checks for:

- selected engine;
- Starship executable presence when selected;
- Starship version probe;
- managed Starship config presence and readability;
- `STARSHIP_CONFIG` ownership and path;
- duplicate Starship initialization outside the managed ozsh block;
- incompatible or missing Nerd Font requirement for the selected preset where detectable.

Warnings do not silently rewrite user-owned shell configuration.

## 6. Data flow

### 6.1 Custom plugin

```text
URL input
  -> validate URL and duplicate name
  -> clone into managed staging directory
  -> discover safe candidate files
  -> user selects load file
  -> user confirms trust
  -> pending configuration
  -> Review & Apply
  -> atomic rename + save + generated Zsh update
```

### 6.2 Engine switch

```text
Prompt screen
  -> choose engine
  -> load that engine's preserved settings
  -> render engine-specific preview
  -> browse engine-specific themes
  -> Review & Apply
  -> write engine artifacts and managed Zsh fragment
```

## 7. Testing strategy

### 7.1 PR A tests

Required tests include:

- no TUI chrome background ANSI sequences;
- active tabs and selected rows remain identifiable without color;
- responsive plugin screen at narrow and wide widths;
- custom plugin empty state and visible add action;
- URL validation;
- duplicate rejection;
- staged clone timeout and cancellation;
- candidate ranking;
- depth and count bounds;
- symlink rejection;
- explicit candidate confirmation;
- explicit trust confirmation;
- cancellation cleanup;
- Apply finalization;
- save failure rollback;
- quarantine restore on removal failure;
- custom load-file change;
- existing recommended plugin behavior;
- race detector and Termux ARM64 build.

### 7.2 PR B tests

Required tests include:

- version 2 to version 3 migration;
- idempotent migration;
- invalid engine validation;
- Native to Starship to Native state preservation;
- deterministic Starship TOML generation;
- contextual theme catalogs;
- unavailable Starship sample preview labeling;
- live preview through injected fake runner;
- preview timeout and cancellation;
- Termux installation plan;
- generic Linux user-local installation plan;
- command execution without shell interpolation;
- managed Zsh fragment for each engine;
- removal of Starship init when returning to Native;
- Review & Apply artifact list;
- Doctor checks;
- `.zshrc` smoke test;
- race detector and Termux ARM64 build.

## 8. Compatibility and non-goals

Compatibility requirements:

- Existing users remain on the native engine after upgrade.
- Existing version 2 prompt appearance remains unchanged unless the user edits it.
- Existing curated plugins continue to work.
- Existing plugin paths remain valid.
- Starship is optional and ozsh remains useful without it.

Non-goals for this phase:

- importing arbitrary existing Starship TOML into the TUI;
- converting native themes into Starship themes or the reverse;
- creating a full editor for every Starship module option;
- executing plugin installation scripts from repositories;
- automatic trust based on GitHub popularity or ownership;
- auto-updating custom plugins;
- supporting non-Zsh shells.

## 9. Delivery sequence

1. Implement and merge PR A from `feat/tui-plugins-polish`.
2. Verify PR A in real Termux with keyboard open and closed.
3. Create a fresh Starship branch from updated `main`.
4. Implement PR B using the engine abstraction and migration defined here.
5. Verify Native and Starship flows in Termux before marking PR B ready.

## 10. Acceptance criteria

The design is complete when all of the following are true:

- ozsh TUI chrome visually inherits the terminal background;
- no gray selection rectangles remain;
- the Plugins screen visibly explains how to add another plugin;
- a custom repository can be cloned, inspected, trusted, staged, applied, disabled, retargeted, and removed safely;
- Native remains the default prompt engine;
- Starship can be selected manually from Prompt;
- Starship installation offers execution or instructions;
- Native and Starship settings survive engine switching independently;
- Themes shows only themes for the active engine;
- real Starship preview is used when available and sample preview is labeled when unavailable;
- Review & Apply exposes every modified artifact;
- all tests and Termux builds pass before either PR is merged.
