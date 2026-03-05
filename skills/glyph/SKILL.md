---
name: glyph
description: Idiomatic patterns for the glyph Go TUI framework. Use when the user is writing, debugging, or extending a glyph application, when code imports github.com/kungfusheep/glyph, or when asked to build a terminal UI in Go.
user-invocable: true
---

# glyph

Declarative terminal UI framework for Go. Always refer to useglyph.sh/api for the full API reference before assuming an API does not exist.

Use dot-import:

```go
import . "github.com/kungfusheep/glyph"
```

## How glyph works

You declare a view tree once. glyph compiles it to a template. Every frame, it dereferences your pointers to read current state.

**Rules:**
- Call `SetView` or `app.View()` once. Never in a loop.
- Bind dynamic values with pointers (`&name`, `&items`)
- Mutate the pointed-to value, then call `app.RequestRender()` from goroutines (handlers auto-render)
- Use `If`/`Switch`/`ForEach` inside the tree for conditional/dynamic content, not Go control flow around `SetView`

## Guidelines for writing glyph code

**Use the functional API, not the struct API**
glyph has two APIs: a legacy struct API (`VBoxNode{Children: []any{...}}`, `TextNode{Content: &name}`) and the current functional API (`VBox(children...)`, `Text(&name).Bold()`). Always use the functional API. The struct types still exist in the package but are no longer idiomatic.

```go
// wrong: struct API
VBoxNode{Children: []any{TextNode{Content: &name}}}

// correct: functional API
VBox(Text(&name))
```

**Use text alignment styles**
Instead of manually padding strings, use `Style{Align: AlignCenter}` or `Style{Align: AlignRight}` with a fixed-width container:

```go
Text("centered").Style(Style{Align: AlignCenter}).Width(40)
Text("right-aligned").Style(Style{Align: AlignRight}).Width(40)
```

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

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
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

    // named func; reused from submit and retry handlers
    deploy := func() {
        progress = 0
        progressLabel = "0%"
        logs = nil
        showError = false
        app.RequestRender()
        go func() {
            for i, step := range steps {
                activeStep = step
                app.RequestRender()
                time.Sleep(500 * time.Millisecond)
                if i == 2 {
                    logs = append(logs, "ERROR: health check failed")
                    showError = true
                    app.RequestRender()
                    return
                }
                progress = (i + 1) * 25
                progressLabel = fmt.Sprintf("%d%%", progress)
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

    // spinner animation; always running
    go func() {
        for range time.Tick(80 * time.Millisecond) {
            frame++
            app.RequestRender()
        }
    }()

    if err := app.RunFrom("form"); err != nil {
        log.Fatal(err)
    }
}
```

## Complete example: inline prompt

`Input()` without a pointer returns an `*InputC`. Read the value after `Run()` via `.Value()`.

```go
package main

import (
    "fmt"
    "log"
    . "github.com/kungfusheep/glyph"
)

func main() {
    name := Input().Placeholder("your name").Width(30)
    token := Input().Placeholder("ghp_...").Width(30).Mask('*')

    app, err := NewInlineApp()
    if err != nil {
        log.Fatal(err)
    }

    app.ClearOnExit(true).
        SetView(Form.LabelFG(Cyan)(
            Field("Name", name),
            Field("Token", token),
        )).
        Handle("<Enter>", app.Stop).
        Handle("<Escape>", app.Stop)

    if err := app.Run(); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("name=%q token=%q\n", name.Value(), token.Value())
}
```

## Pitfalls

| Mistake | Correct |
|---|---|
| `Text(&myInt)` renders nothing | `Text` only accepts `string` or `*string`. Format numbers into a `*string` first. |
| `WidthPct(30)` means 3000% | `WidthPct(0.3)`, scale is 0.0–1.0 |
| `Progress(&myFloat)` | `Progress(&myInt)` takes `*int`, range 0–100 |
| Expecting spinner to animate | Increment `frame` yourself in a goroutine |
| Calling `SetView` in a loop | Call once, mutate pointers, call `RequestRender()` |
| Go `if/else` around `SetView` | Use `If(&val).Then(...).Else(...)` inside the tree |
| `RequestRender()` in a handler | handlers auto-render |
| Forgetting `RequestRender()` in goroutine | goroutines don't auto-render |
| Digits not reaching text input | Add `.NoCounts()` to the view |
| Importing `riffkey` directly | use `func()` handler signatures; `riffkey` is an internal dependency |

## API reference

### App

| Method | Purpose |
|---|---|
| `NewApp()` | fullscreen app (alternate buffer) |
| `NewInlineApp()` | inline app (renders at cursor) |
| `app.SetView(tree)` | set single view |
| `app.View(name, tree) *ViewBuilder` | register named view |
| `app.Handle(key, fn)` | key binding; handler is `func()` |
| `app.Run() error` | block until Stop |
| `app.RunFrom(name) error` | start on named view |
| `app.RunNonInteractive() error` | render-only, no input loop (inline only) |
| `app.Go(name)` | switch view |
| `app.Back()` | return to previous view |
| `app.PushView(name)` / `app.PopView()` | modal view stack |
| `app.Stop()` | exit |
| `app.RequestRender()` | schedule re-render (thread-safe) |
| `app.JumpKey(key)` | enable easymotion-style jump labels |
| `app.Height(h)` | set inline app height |
| `app.ClearOnExit(bool)` | clear on exit (inline) |
| `app.OnBeforeRender(fn)` | callback before each render. Use for derived state. |
| `vb.Handle(key, fn)` / `vb.NoCounts()` | ViewBuilder; returned by `app.View()` |

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

**String display helpers** return `string`. Use inside `Text()` or format into a `*string`:

| Function | Signature | Output |
|---|---|---|
| LED indicator | `LED(on bool) string` | `●` or `○` |
| Multiple LEDs | `LEDs(states ...bool) string` | `●●○` |
| Bracketed LEDs | `LEDsBracket(states ...bool) string` | `[●●○]` |
| Segmented bar | `Bar(filled, total int) string` | `▮▮▮▯▯` |
| Bracketed bar | `BarBracket(filled, total int) string` | `[▮▮▮▯▯]` |
| Analog meter | `Meter(value, max, width int) string` | `├──●──┤` |

### Lists

| Component | Constructor | State type | Key options |
|---|---|---|---|
| Navigable list | `List[T](&items)` | `*[]T` | `.Selection(&int)` `.Render(fn)` `.OnSelect(fn)` `.Marker(s)` `.MarkerStyle(s)` `.MaxVisible(n)` `.SelectedStyle(s)` `.BindNav(down,up)` `.BindVimNav()` `.Ref(fn)` |
| Filterable list | `FilterList[T](&items, extractFn)` | `*[]T` | `.Placeholder(s)` `.Render(fn)` `.MaxVisible(n)` `.Border(style)` `.Title(s)` `.Handle(key, fn)` `.HandleClear(key, fn)` `.Ref(fn)` |
| Checkbox list | `CheckList[T](&items)` | `*[]T` | `.Render(fn)` `.Checked(fn)` `.BindNav(d,u)` `.BindToggle(key)` `.BindDelete(key)` `.Ref(fn)` |
| Iteration | `ForEach[T](&items, fn)` | `*[]T` | — |

FilterList query syntax: `foo` fuzzy, `'exact`, `^prefix`, `suffix$`, `!negate`, `a b` AND, `a | b` OR.

**CheckList struct tags**: tag struct fields and glyph resolves `.Checked()` and `.Render()` from them:

```go
type Task struct {
    Name    string `glyph:"render"`
    Done    bool   `glyph:"checked"`
}

CheckList(&tasks)  // render and checked resolved via struct tags
```

Without struct tags, configure manually: `.Render(func(t *Task) any { return Text(&t.Name) })` and `.Checked(func(t *Task) *bool { return &t.Done })`.

**`.Ref()`** captures a component handle at build time:

```go
var myList *ListC[Item]
List(&items).Render(fn).Ref(func(l *ListC[Item]) { myList = l })
```

`.Ref()` is available on `List`, `CheckList`, `FilterList`, `Log`, `FilterLog`, `Input`, `Checkbox`, `Radio`, and `App`/`ViewBuilder`.

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

**TextInput outside forms.** Use `TextInput` + `app.BindField()` for standalone text input:

```go
var field InputState

app.BindField(&field)  // routes unmatched keystrokes to this field

// in view tree:
TextInput{Field: &field, Placeholder: "search..."}
```

`InputState` bundles `Value string` and `Cursor int`. Call `field.Clear()` to reset. For multiple focusable fields share a `FocusGroup` and set `FocusIndex` on each `TextInput`.

### Conditionals

| Pattern | Usage |
|---|---|
| Bool toggle | `If(&boolVar).Then(a).Else(b)` |
| Equality | `If(&strVar).Eq("val").Then(a).Else(b)` |
| Ordered comparison | `IfOrd(&intVar).Gte(10).Then(a).Else(b)` |
| Multi-way | `Switch(&strVar).Case("a", viewA).Case("b", viewB).Default(viewC)` |

IfOrd operators: `.Gt()`, `.Lt()`, `.Gte()`, `.Lte()`, `.Eq()`, `.Ne()`.

`Switch` requires `.End()` when it is not the last child in a container; omitting it is a compile error.

### Styling

Colors: `Black` `Red` `Green` `Yellow` `Blue` `Magenta` `Cyan` `White` (and `Bright` variants), `PaletteColor(0-255)`, `RGB(r,g,b)`, `Hex(0xRRGGBB)`, `LerpColor(a, b, t)`.

Style struct: `Style{FG: c, BG: c, Fill: c, Attr: AttrBold | AttrItalic, Align: AlignCenter}`.

Attributes: `AttrBold`, `AttrDim`, `AttrItalic`, `AttrUnderline`, `AttrInverse`, `AttrStrikethrough`.

CascadeStyle applies a style to a container. Children inherit it and can override:
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
| Log viewer | `Log(reader)` | streaming text from `io.Reader`. `.BindVimNav()` to scroll with j/k/ctrl-d/ctrl-u |
| Filterable log | `FilterLog(reader)` | log with fzf search. `.BindVimNav()` to scroll |

Check useglyph.sh/api for anything not listed here.
