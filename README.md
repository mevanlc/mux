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
- mouse session selection, width-aware session rows, and clearer full-row
  selection highlighting; and
- more reliable AI CLI detection plus updated main-list keyboard help and
  behavior.

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

Session names use the space remaining after the age, AI badge, and branch
metadata instead of a fixed 18-character field. Names that do not fit are
shortened with an ellipsis according to their displayed terminal width.

### AI detection and keyboard behavior

In addition to tmux's reported command and child-process arguments, mux checks
the pane process's executable name. This preserves detection when a supported
AI tool changes the command text reported by tmux. Detection remains
case-sensitive and recognizes `claude`, `codex`, `aider`, and `gemini`.

In the main session list, `a` attaches to the selected session, window, or pane
just like `Enter`. `Esc` quits just like `q` or `Ctrl+C`, including after a
filter has been applied with `Enter`. While the filter editor is open, `Esc`
clears the filter and returns to the list; it also retains its cancel behavior
in the create, rename, and confirmation overlays.

The on-screen help bar is more compact than upstream's: it summarizes the arrow
keys as navigation and shows `a` for attach. The existing `j`/`k`,
expand/collapse, and `Enter` bindings remain available.

## building

Building requires Go 1.24.2 or newer. tmux is required at runtime; mux supports
Linux and macOS.

Install the latest tagged release of this fork with:

```bash
go install github.com/mevanlc/mux/cmd/mux@latest
```

Or build the current checkout:

```bash
make build
```

This creates `./mux`. Run `make install` to install it under
`$PREFIX/bin` (`/usr/local/bin` by default).

## license

This fork is distributed under the [MIT License](LICENSE).
