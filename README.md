# autodoc-tetris

Tetris, as an [AutoDoc](https://github.com/yongjohnlee80/autodoc) plugin: a dialog over the page,
drawn through AutoDoc's plugin SDK. It plays by the guideline:
- a 7-bag of the seven tetrominoes;
- Super Rotation System turns with wall kicks;
- gravity that quickens every ten lines;
- scoring of 100, 300, 500 and 800 times the level for one to four lines, plus 1 a row for a soft drop and 2 a row for a hard drop;
- a ghost piece, the next piece, pause, and a best score kept between games.

It is also the example of an AutoDoc plugin: a separate program, in its own repository, that AutoDoc
installs from a git URL and runs in a dialog.

## Install

In AutoDoc (`autodoc --ui`): **Plugins › Add from a git URL…**, then

```
https://github.com/yongjohnlee80/autodoc-tetris
```

AutoDoc clones it, then asks before anything runs. It shows the build (`go build …`) and the
command it would start. It is at your own risk, as every plugin is: a plugin runs as you. On
**Yes, at my own risk**, it builds the plugin and lists **Plugins › Tetris**.

It needs git and Go 1.25 or newer, since the build is `go build`. **Plugins › Manage plugins…**
updates it or removes it.

By hand, the same thing:

```sh
git clone https://github.com/yongjohnlee80/autodoc-tetris ~/.config/autodoc/plugins/tetris
cd ~/.config/autodoc/plugins/tetris && go build -o bin/autodoc-tetris .
```

## Keys

| Key | |
|---|---|
| ← → (or h, l) | move |
| ↑, x (or k) | rotate clockwise |
| z | rotate anticlockwise |
| ↓ (or j) | soft drop |
| Space | hard drop |
| p | pause |
| Enter | a new game, once one is lost |
| q, or Esc | quit (Esc is AutoDoc's: it closes any plugin's dialog) |

The best score is kept in `$XDG_STATE_HOME/autodoc-tetris/best` (`~/.local/state/…`). It is not kept
in the plugin's directory, since that is a git clone an update resets.

## How it is made

- `plugin.toml` is the manifest AutoDoc reads. It declares a dialog plugin of plugin protocol 1, the
  dialog's size (44×21), the `command` it starts, and the `[install] build` that adding it runs.
- `game/` is the game, with no screen in it, tested with a seed.
- `render.go` draws a game into a `plugin.Frame`. The colours are in the themes' vocabulary (`"cyan"`,
  `"#ff8700"`). In the mono theme every piece is bright white, and the ghost tells them from the floor.
- `main.go` is the `plugin.Handler`: `plugin.Serve` runs it, gravity ticks on a goroutine of its
  own, and every change is a frame.

The protocol and the SDK are AutoDoc's package
[`plugin`](https://github.com/yongjohnlee80/autodoc/tree/main/plugin) (AutoDoc's ADR 0209).

## License

[Apache-2.0](LICENSE). Copyright 2026 Yong Sung John Lee.
