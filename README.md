# glyph skill for Claude Code

A Claude Code plugin that gives Claude idiomatic knowledge of [glyph](https://useglyph.sh) — the declarative terminal UI framework for Go.

Once installed, Claude automatically applies correct glyph patterns when you're working on a glyph application. No need to prompt it.

## Install

In Claude Code:

```
/plugin install glyph
```

## Usage

Claude activates automatically when your code imports `github.com/kungfusheep/glyph` or you ask to build a terminal UI in Go. If you want to be explicit, invoke it directly before asking Claude to write any glyph code:

```
/glyph:glyph
```

## What it covers

- Core mental model — how the template/pointer system works
- Complete examples for single-view, multi-view, and inline apps
- Full API reference for layout, display, lists, tables, forms, conditionals, and styling
- Common pitfalls and how to avoid them

Full API docs at [useglyph.sh/api](https://useglyph.sh/api).
