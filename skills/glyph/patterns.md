# glyph patterns

Additional recipes beyond the examples in SKILL.md.

## Dashboard with live data

```go
cpu := 0
mem := 0
frame := 0
sparkData := []float64{3, 5, 2, 8, 4}

app.SetView(
    VBox.Gap(1)(
        HBox.Gap(1)(
            VBox.Border(BorderRounded).Title("CPU").Grow(1)(
                Progress(&cpu).FG(Green),
            ),
            VBox.Border(BorderRounded).Title("Memory").Grow(1)(
                Progress(&mem).FG(Yellow),
            ),
            VBox.Border(BorderRounded).Title("Trend").Grow(1)(
                Sparkline(&sparkData).FG(Cyan),
            ),
        ),
        HBox.Gap(1)(
            Spinner(&frame).Frames(SpinnerDots),
            Text("Monitoring...").Dim(),
        ),
    ),
)

go func() {
    for range time.Tick(80 * time.Millisecond) {
        frame++
        app.RequestRender()
    }
}()

go func() {
    for range time.Tick(time.Second) {
        cpu = getCPU()
        mem = getMem()
        sparkData = append(sparkData, float64(cpu))
        app.RequestRender()
    }
}()
```

## Command palette (FilterList)

```go
type Command struct {
    Name string
    Desc string
}

commands := []Command{
    {Name: "Open File", Desc: "Browse and open"},
    {Name: "Save", Desc: "Save current file"},
    {Name: "Quit", Desc: "Exit the editor"},
}

app.SetView(
    FilterList(&commands, func(c *Command) string { return c.Name }).
        Placeholder("type a command...").
        Render(func(c *Command) any {
            return HBox.Gap(2)(
                Text(&c.Name).Bold(),
                Text(&c.Desc).Dim(),
            )
        }).
        Handle("<Enter>", func(c *Command) {
            execute(c)
            app.Stop()
        }).
        HandleClear("<Esc>", app.Stop),
)
```

## Todo list (CheckList)

```go
type Task struct {
    Name string
    Done bool
}

tasks := []Task{
    {Name: "Write docs"},
    {Name: "Add tests"},
    {Name: "Ship it"},
}

app.SetView(
    VBox.Border(BorderRounded).Title("Tasks")(
        CheckList(&tasks).
            Render(func(t *Task) any { return Text(&t.Name) }).
            BindNav("j", "k").
            BindToggle(" ").
            BindDelete("d"),
    ),
)
```

## Tabbed views with Switch

```go
mode := "list"
tabIdx := 0

app.SetView(VBox(
    Tabs([]string{"List", "Grid", "Detail"}, &tabIdx),
    Switch(&mode).
        Case("list", listView).
        Case("grid", gridView).
        Case("detail", detailView).
        End(),
))

app.Handle("1", func() { mode = "list"; tabIdx = 0 })
app.Handle("2", func() { mode = "grid"; tabIdx = 1 })
app.Handle("3", func() { mode = "detail"; tabIdx = 2 })
```

## Custom widget (gradient bar)

```go
Widget(
    func(availW int16) (w, h int16) {
        return availW, 1
    },
    func(buf *Buffer, x, y, w, h int16) {
        for i := int16(0); i < w; i++ {
            t := float64(i) / float64(w)
            c := LerpColor(Blue, Cyan, t)
            buf.Set(int(x+i), int(y), Cell{Rune: '█', Style: Style{FG: c}})
        }
    },
)
```

## Inline spinner with completion

```go
frame := 0
done := false
status := "Working..."

app, _ := NewInlineApp()
app.SetView(HBox.Gap(1)(
    If(&done).
        Then(Text("✓").FG(Green)).
        Else(Spinner(&frame).Frames(SpinnerDots)),
    Text(&status),
))
app.RunNonInteractive()

go func() {
    for range time.Tick(80 * time.Millisecond) {
        frame++
        app.RequestRender()
    }
}()

doWork()
status = "Done!"
done = true
app.RequestRender()
time.Sleep(300 * time.Millisecond)
app.Stop()
```
