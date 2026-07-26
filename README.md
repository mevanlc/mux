# a mux fork

This is a fork of [mux](https://github.com/lunemis/mux), a terminal UI for
previewing, managing, and switching between tmux sessions. See the
[upstream repository](https://github.com/lunemis/mux) for general installation
and usage documentation.

## about this fork

This fork adds:

- responsive stacked and side-by-side layouts, with explicit layout overrides;
- keyboard- and mouse-adjustable panel splits that persist independently for
  each layout;
- mouse session selection and clearer full-row selection highlighting; and
- more reliable AI CLI detection plus `Esc` as a main-list quit key.

### responsive layouts

The default `auto` mode chooses between a horizontal, side-by-side layout and a
vertical layout with the session list stacked above the preview. It uses the
terminal's aspect ratio, accounting for terminal cells being taller than they
are wide: near-square terminals stack the panels, while terminals closer to
16:9 keep them side by side.

Use `--layout` or `-l` to override the choice. The flag is case-insensitive and
accepts these values:

```text
auto       a
horizontal h
vertical   v
```

For example:

```bash
mux -l vertical
mux popup --layout horizontal
```

The layout flag controls only panel orientation. It does not change the popup's
size.

### adjustable panel splits

The session list initially receives 40% of the available width in horizontal
layout or 40% of the available height in vertical layout. Resize it one terminal
cell at a time with `Shift+Left`/`Shift+Up` to shrink it and
`Shift+Right`/`Shift+Down` to grow it. You can also drag the divider with the
left mouse button.

mux keeps separate split positions for horizontal and vertical layouts and
saves changes in `mux/config.json` under the operating system's user
configuration directory. Each panel is kept at least five cells wide or high
when the available space permits.

### mouse and selection behavior

Click a visible session row to select it and refresh its preview. The selected
session's background now spans the full list width, including the padding around
AI badges, which makes the active row unambiguous.

Mouse clicks select session rows only; window and pane rows still use keyboard
navigation.

### AI detection and quitting

In addition to tmux's reported command and child-process arguments, mux checks
the pane process's executable name. This preserves detection when a supported
AI tool changes the command text reported by tmux. Detection remains
case-sensitive and recognizes `claude`, `codex`, `aider`, and `gemini`.

In the main session list, `Esc` now quits just like `q` or `Ctrl+C`. In filters,
editors, and confirmation overlays, `Esc` retains its cancel behavior.

## building

Building requires Go 1.24.2 or newer. tmux is required at runtime; mux supports
Linux and macOS.

```bash
make build
```

This creates `./mux`. Run `make install` to install it under
`$PREFIX/bin` (`/usr/local/bin` by default).

## license

This fork is distributed under the [MIT License](LICENSE).
