---
name: glyph
description: Idiomatic patterns for building terminal UIs with the glyph Go framework. Use when helping a developer write, debug, or extend a glyph application.
user-invocable: true
---

# glyph

Declarative, reactive terminal UI framework for Go. This skill is self-contained — you do not need to read the framework source code to write correct glyph applications.

```go
import . "github.com/kungfusheep/glyph"
```

The dot-import is idiomatic — it keeps view code readable.

## How glyph works

You declare a view tree once. glyph compiles it to a template. Every frame, it dereferences your pointers to read current state. No diffing, no rebuild.

**Rules:**
- Call `SetView` or `app.View()` once — never in a loop
- Bind dynamic values with pointers (`&name`, `&items`)
- Mutate the pointed-to value, then call `app.RequestRender()` from goroutines (handlers auto-render)
- Use `If`/`Switch`/`ForEach` inside the tree for conditional/dynamic content — not Go control flow around `SetView`

## Complete example: single-view app

```go
package main

import (
    "fmt"
    "log"
    "time"
    . "github.com/kungfusheep/glyph"
)

func main() {
    app, err := NewApp()
    if err != nil {
        log.Fatal(err)
    }

    count := 0
    countLabel := "0"
    frame := 0
    status := "ready"

    app.SetView(
        VBox.Border(BorderRounded).Title("Demo")(
            HBox.Gap(2)(
                Text("Count:"),
                Text(&countLabel).Bold().FG(Cyan),
                Spinner(&frame).Frames(SpinnerDots),
            ),
            Progress(&count).Width(30).FG(Green),
            If(&status).Eq("done").
                Then(Text("Complete!").FG(Green)).
                Else(Text(&status).Dim()),
        ),
    )

    app.Handle("j", func() { count++; countLabel = fmt.Sprintf("%d", count) })
    app.Handle("k", func() { count--; countLabel = fmt.Sprintf("%d", count) })
    app.Handle("q", app.Stop)

    go func() {
        for range time.Tick(80 * time.Millisecond) {
            frame++
            app.RequestRender()
        }
    }()

    app.Run()
}
```

## Complete example: multi-view app with form

```go
package main

import (
    "fmt"
    "log"
    "time"
    . "github.com/kungfusheep/glyph"
)

func main() {
    app, err := NewApp()
    if err != nil {
        log.Fatal(err)
    }

    // form state
    var name string
    var target int

    // deploy state
    steps := []string{"Build", "Test", "Deploy", "Verify"}
    activeStep := ""
    progress := 0
    progressLabel := "0%"
    frame := 0
    logs := []string{}
    showError := false

    // extract background work into a named function — reuse from submit and retry
    deploy := func() {
        activeStep = steps[0]
        progress = 0
        progressLabel = "0%"
        logs = nil
        app.RequestRender()
        go func() {
            for i, step := range steps {
                activeStep = step
                for p := i * 25; p < (i+1)*25; p++ {
                    time.Sleep(50 * time.Millisecond)
                    progress = p
                    progressLabel = fmt.Sprintf("%d%%", p)
                    if p == 75 {
                        logs = append(logs, "ERROR: health check failed")
                        showError = true
                        app.RequestRender()
                        return
                    }
                    app.RequestRender()
                }
                logs = append(logs, fmt.Sprintf("%s complete", step))
                app.RequestRender()
            }
            progress = 100
            progressLabel = "100%"
            logs = append(logs, "deployment complete!")
            app.RequestRender()
        }()
    }

    var form *FormC
    form = Form.LabelBold().OnSubmit(func() {
        if form.ValidateAll() {
            app.Go("main")
            deploy()
        }
    })(
        Field("Name", Input(&name).Placeholder("release name").Validate(VRequired)),
        Field("Target", Radio(&target, "staging", "production")),
    )

    app.View("form",
        VBox.Border(BorderRounded).Title("Config")(form),
    ).NoCounts().Handle("q", app.Stop)

    app.View("main",
        VBox.Gap(1)(
            HBox.Gap(1)(
                // ForEach with conditional per-item rendering:
                // compare external state pointer against each item
                VBox.WidthPct(0.3).Border(BorderRounded).Title("Steps")(
                    ForEach(&steps, func(step *string) any {
                        return If(&activeStep).Eq(*step).
                            Then(HBox.Gap(1)(
                                Spinner(&frame).Frames(SpinnerDots).FG(Cyan),
                                Text(step).Bold(),
                            )).
                            Else(Text(step).Dim())
                    }),
                ),
                VBox.WidthPct(0.4).Border(BorderRounded).Title("Progress")(
                    Text(&progressLabel).Bold(),
                    Progress(&progress).FG(Green),
                ),
                VBox.WidthPct(0.3).Border(BorderRounded).Title("Log")(
                    ForEach(&logs, func(line *string) any {
                        return Text(line).Dim()
                    }),
                ),
            ),
            If(&showError).Then(
                Overlay.Centered().Backdrop().BackdropFG(BrightBlack)(
                    VBox.Border(BorderRounded).Title("Error")(
                        Text("Failed at 75%").FG(Red),
                        Text("[r] retry  [c] cancel").Dim(),
                    ),
                ),
            ),
        ),
    ).Handle("q", app.Stop).
        Handle("r", func() { if showError { showError = false; deploy() } }).
        Handle("c", func() { if showError { showError = false; app.Go("form") } })

    // spinner animation — always running
    go func() {
        for range time.Tick(80 * time.Millisecond) {
            frame++
            app.RequestRender()
        }
    }()

    app.RunFrom("form")
}
```

## Complete example: inline prompt

```go
var name string
app, _ := NewInlineApp()
app.SetView(Form.LabelBold()(
    Field("Name", Input(&name).Placeholder("enter name")),
))
app.ClearOnExit(true)
app.Run()
fmt.Println("Hello,", name)
```

## Pitfalls

| Mistake | Correct |
|---|---|
| `Text(&myInt)` — renders nothing | `Text` only accepts `string` or `*string`. Format numbers into a `*string` first. |
| `WidthPct(30)` — means 3000% | `WidthPct(0.3)` — scale is 0.0–1.0 |
| `Progress(&myFloat)` | `Progress(&myInt)` — takes `*int`, range 0–100 |
| Expecting spinner to animate | You must increment `frame` yourself in a goroutine |
| Calling `SetView` in a loop | Call once, mutate pointers, call `RequestRender()` |
| Go `if/else` around `SetView` | Use `If(&val).Then(...).Else(...)` inside the tree |
| `RequestRender()` in a handler | Not needed — handlers auto-render |
| Forgetting `RequestRender()` in goroutine | Required — goroutines don't auto-render |
| Digits not reaching text input | Add `.NoCounts()` to the view |
| Importing `riffkey` directly | Not needed — use `func()` handler signatures. `riffkey` is an internal dependency. |

## API reference

### App

| Method | Purpose |
|---|---|
| `NewApp()` | fullscreen app (alternate buffer) |
| `NewInlineApp()` | inline app (renders at cursor) |
| `app.SetView(tree)` | set single view |
| `app.View(name, tree) *ViewBuilder` | register named view |
| `app.Handle(key, fn)` | key binding — handler is `func()` |
| `app.Run()` | block until Stop |
| `app.RunFrom(name)` | start on named view |
| `app.RunNonInteractive()` | render-only, no input loop |
| `app.Go(name)` | switch view |
| `app.PushView(name)` / `app.PopView()` | modal view stack |
| `app.Stop()` | exit |
| `app.RequestRender()` | schedule re-render (thread-safe) |
| `app.RenderNow()` | immediate render |
| `app.JumpKey(key)` | enable easymotion-style jump labels |
| `app.Height(h)` | set inline app height |
| `app.ClearOnExit(bool)` | clear on exit (inline) |

ViewBuilder: `.Handle(key, fn)`, `.NoCounts()`.

### Layout

| Component | Constructor | Key options |
|---|---|---|
| Vertical stack | `VBox(children...)` | `.Gap(n)` `.Border(style)` `.Title(s)` `.BorderFG(c)` `.Width(n)` `.Height(n)` `.WidthPct(0-1)` `.Grow(n)` `.FitContent()` `.Fill(c)` `.CascadeStyle(&s)` `.Margin(n)` `.MarginVH(v,h)` `.MarginTRBL(t,r,b,l)` |
| Horizontal stack | `HBox(children...)` | same as VBox |
| Modal overlay | `Overlay(children...)` | `.Centered()` `.Backdrop()` `.BackdropFG(c)` `.BG(c)` `.Size(w,h)` `.At(x,y)` |
| Custom layout | `Arrange(layoutFn)(children...)` | layout returns `[]Rect` |
| Flexible space | `Space()` | `.Grow(n)` `.Char(r)` |
| Fixed vertical gap | `SpaceH(n)` | |
| Fixed horizontal gap | `SpaceW(n)` | |
| Horizontal rule | `HRule()` | `.Char(r)` `.FG(c)` |
| Vertical rule | `VRule()` | `.Height(n)` `.Char(r)` `.FG(c)` |

Border styles: `BorderRounded`, `BorderSingle`, `BorderDouble`.

### Display

| Component | Constructor | State type | Key options |
|---|---|---|---|
| Static text | `Text("str")` | — | `.FG(c)` `.BG(c)` `.Bold()` `.Dim()` `.Italic()` `.Underline()` `.Inverse()` `.Width(n)` |
| Dynamic text | `Text(&str)` | `*string` only | same |
| Rich text | `Textf(parts...)` | mix of `string`, `*string`, `Bold(x)`, `Dim(x)`, `FG(x,c)` | — |
| Progress bar | `Progress(&val)` | `*int` (0–100) | `.Width(n)` `.FG(c)` `.BG(c)` |
| Spinner | `Spinner(&frame)` | `*int` (manual increment) | `.Frames([]string)` `.FG(c)` |
| Sparkline | `Sparkline(&vals)` | `*[]float64` | `.Width(n)` `.Range(min,max)` `.FG(c)` |
| Leader dots | `Leader(label, &val)` | `*string` | `.Fill(r)` `.FG(c)` |
| Tabs | `Tabs(labels, &idx)` | `*int` | `.Kind(style)` `.ActiveStyle(s)` `.InactiveStyle(s)` `.Gap(n)` |

**Text only accepts `string` or `*string`.** To display a number, format it: `label := fmt.Sprintf("%d%%", pct)` then `Text(&label)`.

Spinner frame sets: `SpinnerBraille`, `SpinnerDots`, `SpinnerLine`, `SpinnerCircle`.
Tab styles: `TabsStyleUnderline`, `TabsStyleBox`, `TabsStyleBracket`.

### Lists

| Component | Constructor | State type | Key options |
|---|---|---|---|
| Navigable list | `List[T](&items)` | `*[]T` | `.Selection(&int)` `.Render(fn)` `.OnSelect(fn)` `.Marker(s)` `.MarkerStyle(s)` `.MaxVisible(n)` `.SelectedStyle(s)` `.BindNav(down,up)` `.BindVimNav()` |
| Filterable list | `FilterList[T](&items, extractFn)` | `*[]T` | `.Placeholder(s)` `.Render(fn)` `.MaxVisible(n)` `.Handle(key, fn)` `.HandleClear(key, fn)` |
| Checkbox list | `CheckList[T](&items)` | `*[]T` | `.Render(fn)` `.BindNav(d,u)` `.BindToggle(key)` `.BindDelete(key)` |
| Iteration | `ForEach[T](&items, fn)` | `*[]T` | — |

FilterList query syntax: `foo` fuzzy, `'exact`, `^prefix`, `suffix$`, `!negate`, `a b` AND, `a | b` OR.

### Tables

| Component | Constructor | Key options |
|---|---|---|
| Auto table | `AutoTable(&rows)` | `.Columns(names...)` `.Headers(names...)` `.Column(name, formatter)` `.HeaderStyle(s)` `.RowStyle(s)` `.AltRowStyle(s)` `.Sortable()` `.SortBy(field, asc)` `.Scrollable(n)` `.BindVimNav()` `.Gap(n)` `.Border(style)` |

Column formatters: `Number(decimals)`, `Percent(decimals)`, `Currency(symbol, decimals)`, `PercentChange(decimals)`, `Bytes()`, `Bool(yes, no)`.

### Forms

| Component | Constructor | State type | Key options |
|---|---|---|---|
| Form container | `Form(fields...)` | — | `.LabelBold()` `.LabelFG(c)` `.Gap(n)` `.OnSubmit(fn)` `.ValidateAll() bool` |
| Form field | `Field(label, component)` | — | — |
| Text input | `Input(&str)` | `*string` | `.Placeholder(s)` `.Width(n)` `.Mask(r)` `.Validate(fn, when...)` `.Bind()` `.ManagedBy(&fm)` |
| Checkbox | `Checkbox(&bool, label)` | `*bool` | `.Marks(on, off)` `.BindToggle(key)` `.Validate(fn, when...)` |
| Radio | `Radio(&idx, options...)` | `*int` | `.Marks(on, off)` `.Gap(n)` `.Horizontal()` `.BindNav(next, prev)` |

Validators: `VRequired`, `VEmail`, `VMinLen(n)`, `VMaxLen(n)`, `VMatch(regex)`, `VTrue`.
Trigger flags: `VOnChange`, `VOnBlur`, `VOnSubmit`.

### Conditionals

| Pattern | Usage |
|---|---|
| Bool toggle | `If(&boolVar).Then(a).Else(b)` |
| Equality | `If(&strVar).Eq("val").Then(a).Else(b)` |
| Ordered comparison | `IfOrd(&intVar).Gte(10).Then(a).Else(b)` |
| Multi-way | `Switch(&strVar).Case("a", viewA).Case("b", viewB).Default(viewC)` |

IfOrd operators: `.Gt()`, `.Lt()`, `.Gte()`, `.Lte()`, `.Eq()`, `.Ne()`.

### Styling

Colors: `Black` `Red` `Green` `Yellow` `Blue` `Magenta` `Cyan` `White` (and `Bright` variants), `PaletteColor(0-255)`, `RGB(r,g,b)`, `Hex(0xRRGGBB)`, `LerpColor(a, b, t)`.

Style struct: `Style{FG: c, BG: c, Fill: c, Attr: AttrBold | AttrItalic, Align: AlignCenter}`.

Attributes: `AttrBold`, `AttrDim`, `AttrItalic`, `AttrUnderline`, `AttrInverse`, `AttrStrikethrough`.

CascadeStyle applies a style to a container — children inherit, can override:
```go
theme := Style{FG: Cyan}
VBox.CascadeStyle(&theme)(children...)
```

Built-in themes: `ThemeDark`, `ThemeLight`, `ThemeMonochrome`.

### Key binding patterns

```
"q"          single key
"gg"         sequence
"<Enter>"    special key
"<C-c>"      ctrl+c
"<S-Tab>"    shift+tab
"<Up>"       arrow key
```

Special keys: `<Enter>`, `<Escape>`, `<Tab>`, `<S-Tab>`, `<Space>`, `<Backspace>`, `<Delete>`, `<Up>`, `<Down>`, `<Left>`, `<Right>`, `<PageUp>`, `<PageDown>`, `<Home>`, `<End>`, `<C-x>` (ctrl), `<S-x>` (shift), `<A-x>` (alt).

### Advanced

| Component | Constructor | Purpose |
|---|---|---|
| Jump target | `Jump(child, onSelect)` | easymotion-style label |
| Layer view | `LayerView(&layer)` | scrollable pre-rendered buffer |
| Custom widget | `Widget(measureFn, renderFn)` | fully custom rendering |
| Scoped block | `Define(func() any { ... })` | local helpers at compile time |
| Log viewer | `Log(reader)` | streaming text from `io.Reader` |
| Filterable log | `FilterLog(reader)` | log with fzf search |
