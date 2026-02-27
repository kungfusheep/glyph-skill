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

	// deploy simulation — extracted so retry can reuse it
	deploy := func() {
		activeStep = steps[0]
		progress = 0
		progressLabel = "0%"
		logs = nil
		showError = false
		app.RequestRender()

		go func() {
			for i, step := range steps {
				activeStep = step
				logs = append(logs, fmt.Sprintf("starting %s...", step))
				app.RequestRender()

				// each step ~1s, progress increments smoothly across 25 ticks
				for p := i * 25; p < (i+1)*25; p++ {
					time.Sleep(40 * time.Millisecond)
					progress = p
					progressLabel = fmt.Sprintf("%d%%", p)

					// simulate failure at 75%
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
			logs = append(logs, "deployment finished!")
			app.RequestRender()
		}()
	}

	// form view
	var form *FormC
	form = Form.LabelBold().OnSubmit(func() {
		if form.ValidateAll() {
			app.Go("main")
			deploy()
		}
	})(
		Field("Release", Input(&name).Placeholder("release name").Validate(VRequired)),
		Field("Target", Radio(&target, "staging", "production")),
	)

	app.View("form",
		VBox.Border(BorderRounded).Title("Deploy Config")(form),
	).NoCounts().Handle("q", app.Stop)

	// dashboard view
	app.View("main",
		VBox.Gap(1)(
			HBox.Gap(1)(
				// left: steps with spinner on active
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

				// middle: progress
				VBox.WidthPct(0.4).Border(BorderRounded).Title("Progress")(
					Text(&progressLabel).Bold(),
					Progress(&progress).FG(Green),
				),

				// right: log lines
				VBox.WidthPct(0.3).Border(BorderRounded).Title("Log")(
					ForEach(&logs, func(line *string) any {
						return Text(line).Dim()
					}),
				),
			),

			// error overlay
			If(&showError).Then(
				Overlay.Centered().Backdrop().BackdropFG(BrightBlack)(
					VBox.Border(BorderRounded).Title("Error")(
						Text("Deployment failed at 75%").FG(Red),
						Text("[r] retry  [c] cancel").Dim(),
					),
				),
			),
		),
	).Handle("q", app.Stop).
		Handle("r", func() {
			if showError {
				showError = false
				deploy()
			}
		}).
		Handle("c", func() {
			if showError {
				showError = false
				app.Go("form")
			}
		})

	// spinner animation
	go func() {
		for range time.Tick(80 * time.Millisecond) {
			frame++
			app.RequestRender()
		}
	}()

	app.RunFrom("form")
}
