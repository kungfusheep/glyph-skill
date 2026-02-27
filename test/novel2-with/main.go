package main

import (
	"fmt"
	"log"

	. "github.com/kungfusheep/glyph"
)

type Todo struct {
	Text string `glyph:"render"`
	Done bool   `glyph:"checked"`
	ID   int
}

var nextID int

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	todos := []Todo{
		{Text: "Learn glyph", Done: true, ID: 0},
		{Text: "Build something cool", Done: false, ID: 1},
		{Text: "Ship it", Done: false, ID: 2},
	}
	nextID = 3

	mode := "all"
	statusLabel := ""
	var input InputState

	displayed := []Todo{}

	rebuild := func() {
		displayed = displayed[:0]
		active := 0
		for i := range todos {
			if !todos[i].Done {
				active++
			}
			switch mode {
			case "all":
				displayed = append(displayed, todos[i])
			case "active":
				if !todos[i].Done {
					displayed = append(displayed, todos[i])
				}
			case "done":
				if todos[i].Done {
					displayed = append(displayed, todos[i])
				}
			}
		}
		if active == 1 {
			statusLabel = "1 item left"
		} else {
			statusLabel = fmt.Sprintf("%d items left", active)
		}
	}
	rebuild()

	findByID := func(id int) int {
		for i := range todos {
			if todos[i].ID == id {
				return i
			}
		}
		return -1
	}

	app.SetView(
		VBox.Border(BorderRounded).Title("Todo").Gap(1)(
			HBox.Gap(1)(
				Text("Add:"),
				TextInput{Field: &input, Width: 40, Placeholder: "what needs doing?"},
			),
			HBox.Gap(2)(
				Text("[1] all").Bold(),
				Text("[2] active").Bold(),
				Text("[3] done").Bold(),
				Text("|").Dim(),
				Switch(&mode).
					Case("all", Text("showing: all").FG(Cyan)).
					Case("active", Text("showing: active").FG(Green)).
					Case("done", Text("showing: done").FG(Yellow)),
			),
			CheckList(&displayed).
				BindNav("j", "k").
				Handle("<Space>", func(item *Todo) {
					idx := findByID(item.ID)
					if idx >= 0 {
						todos[idx].Done = !todos[idx].Done
						rebuild()
					}
				}).
				Handle("d", func(item *Todo) {
					idx := findByID(item.ID)
					if idx >= 0 {
						todos = append(todos[:idx], todos[idx+1:]...)
						rebuild()
					}
				}),
			HRule().FG(BrightBlack),
			Text(&statusLabel).Dim(),
		),
	).
		Handle("<Enter>", func() {
			if input.Value != "" {
				todos = append(todos, Todo{Text: input.Value, ID: nextID})
				nextID++
				input.Clear()
				rebuild()
			}
		}).
		Handle("1", func() { mode = "all"; rebuild() }).
		Handle("2", func() { mode = "active"; rebuild() }).
		Handle("3", func() { mode = "done"; rebuild() }).
		Handle("q", app.Stop).
		BindField(&input)

	app.Run()
}
