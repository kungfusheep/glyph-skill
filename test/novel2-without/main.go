package main

import (
	"fmt"

	. "github.com/kungfusheep/glyph"
)

type Todo struct {
	Text string `glyph:"render"`
	Done bool   `glyph:"checked"`
}

func main() {
	todos := []Todo{
		{"Learn glyph", false},
		{"Build something cool", false},
	}

	// filtered views for each mode
	var active, done []Todo

	var input InputState
	var view int // 0=all, 1=active, 2=done
	var status string

	syncFilters := func() {
		active = active[:0]
		done = done[:0]
		for i := range todos {
			if !todos[i].Done {
				active = append(active, todos[i])
			} else {
				done = append(done, todos[i])
			}
		}
		remaining := 0
		for i := range todos {
			if !todos[i].Done {
				remaining++
			}
		}
		status = fmt.Sprintf("%d items left", remaining)
	}
	syncFilters()

	allList := CheckList(&todos).
		BindNav("j", "k").
		BindToggle("<space>").
		BindDelete("d")

	activeList := CheckList(&active).
		BindNav("j", "k").
		BindToggle("<space>").
		BindDelete("d")

	doneList := CheckList(&done).
		BindNav("j", "k").
		BindToggle("<space>").
		BindDelete("d")

	app, _ := NewApp()
	app.OnBeforeRender(syncFilters)
	app.SetView(
		VBox.Border(BorderRounded).Title("Todo").Gap(1)(
			HBox.Gap(1)(
				Text("Add:"),
				TextInput{Field: &input, Width: 40, Placeholder: "what needs doing?"},
			),
			HBox.Gap(2)(
				Text("1:all").Bold(),
				Text("2:active").Dim(),
				Text("3:done").Dim(),
			),
			Switch(&view).
				Case(0, allList).
				Case(1, activeList).
				Case(2, doneList).
				Default(allList),
			SpacerNode{},
			Text(&status).Dim(),
		)).
		Handle("<enter>", func() {
			if input.Value != "" {
				todos = append(todos, Todo{Text: input.Value})
				input.Clear()
			}
		}).
		Handle("1", func() { view = 0 }).
		Handle("2", func() { view = 1 }).
		Handle("3", func() { view = 2 }).
		Handle("q", app.Stop).
		BindField(&input).
		Run()
}
