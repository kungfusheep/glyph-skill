package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	g "github.com/kungfusheep/glyph"
)

// deployment steps
var steps = []string{"Build", "Test", "Deploy", "Verify"}

// state holds all mutable state for the app
type state struct {
	mu sync.Mutex

	// form state
	releaseName string
	envChoice   int // 0=staging, 1=production

	// dashboard state
	currentStep int
	progress    int
	spinFrame   int
	stepLabels  [4]string
	logText     string

	// overlay
	showError  bool
	errorMsg   string
	deploying  bool
	stopDeploy chan struct{}

	// log pipe
	logWriter *io.PipeWriter
	logReader *io.PipeReader
}

func main() {
	app, err := g.NewApp()
	if err != nil {
		panic(err)
	}

	s := &state{
		envChoice: 0,
	}
	for i, step := range steps {
		s.stepLabels[i] = step
	}

	// set up log pipe
	s.logReader, s.logWriter = io.Pipe()

	// form view
	submit := func() {
		if strings.TrimSpace(s.releaseName) == "" {
			return
		}
		s.startDeploy(app)
		app.Go("dashboard")
	}

	form := g.Form.LabelBold().OnSubmit(submit).Gap(1).Margin(2)(
		g.Field("Release Name", g.Input(&s.releaseName).Validate(g.VRequired, g.VOnBlur).Placeholder("e.g. v1.2.3")),
		g.Field("Environment", g.Radio(&s.envChoice, "staging", "production")),
	)

	app.View("form", g.VBox(
		g.SpaceH(1),
		g.Text("  Deployment Dashboard").Bold().FG(g.Cyan),
		g.HRule().FG(g.BrightBlack),
		g.SpaceH(1),
		form,
		g.SpaceH(1),
		g.Text("  Tab/Shift-Tab to navigate, Enter to deploy, q to quit").Dim(),
		g.Space(),
	)).NoCounts().Handle("q", func() { app.Stop() })

	// dashboard view
	app.View("dashboard", g.VBox(
		g.SpaceH(1),
		g.Text("  Deploying...").Bold().FG(g.Cyan),
		g.HRule().FG(g.BrightBlack),
		g.SpaceH(1),
		g.HBox.Gap(1).Grow(1)(
			// left: steps
			g.VBox.WidthPct(0.25).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Steps")(
				buildStepRow(s, 0),
				buildStepRow(s, 1),
				buildStepRow(s, 2),
				buildStepRow(s, 3),
			),
			// middle: progress
			g.VBox.WidthPct(0.35).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Progress")(
				g.Space(),
				g.Progress(&s.progress).Width(30).FG(g.Green),
				g.Text(&s.logText).FG(g.BrightBlack),
				g.Space(),
			),
			// right: live log
			g.VBox.Grow(1).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Log")(
				g.Log(s.logReader).Grow(1).MaxLines(500).AutoScroll(true),
			),
		),
		// overlay for error
		g.If(&s.showError).Then(
			g.Overlay.Centered().Backdrop().BackdropFG(g.BrightBlack).Size(50, 10).BG(g.Black)(
				g.VBox.Border(g.BorderRounded).BorderFG(g.Red).Grow(1).Gap(1)(
					g.SpaceH(1),
					g.Text("  Deployment Failed!").Bold().FG(g.Red),
					g.Text(&s.errorMsg).FG(g.BrightWhite).MarginTRBL(0, 2, 0, 2),
					g.Space(),
					g.HBox.Gap(4)(
						g.SpaceW(2),
						g.Text("[r] Retry").Bold().FG(g.Yellow),
						g.Text("[c] Cancel").Bold().FG(g.BrightBlack),
					),
					g.SpaceH(1),
				),
			),
		),
		g.SpaceH(1),
		g.Text("  q to quit").Dim(),
	)).Handle("q", func() {
		s.cancelDeploy()
		app.Stop()
	}).Handle("r", func() {
		if s.showError {
			s.showError = false
			// create new pipe since old reader may be consumed
			s.logReader, s.logWriter = io.Pipe()
			// rebuild the dashboard view with the new reader
			app.UpdateView("dashboard", g.VBox(
				g.SpaceH(1),
				g.Text("  Deploying...").Bold().FG(g.Cyan),
				g.HRule().FG(g.BrightBlack),
				g.SpaceH(1),
				g.HBox.Gap(1).Grow(1)(
					g.VBox.WidthPct(0.25).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Steps")(
						buildStepRow(s, 0),
						buildStepRow(s, 1),
						buildStepRow(s, 2),
						buildStepRow(s, 3),
					),
					g.VBox.WidthPct(0.35).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Progress")(
						g.Space(),
						g.Progress(&s.progress).Width(30).FG(g.Green),
						g.Text(&s.logText).FG(g.BrightBlack),
						g.Space(),
					),
					g.VBox.Grow(1).Border(g.BorderRounded).BorderFG(g.BrightBlack).Title("Log")(
						g.Log(s.logReader).Grow(1).MaxLines(500).AutoScroll(true),
					),
				),
				g.If(&s.showError).Then(
					g.Overlay.Centered().Backdrop().BackdropFG(g.BrightBlack).Size(50, 10).BG(g.Black)(
						g.VBox.Border(g.BorderRounded).BorderFG(g.Red).Grow(1).Gap(1)(
							g.SpaceH(1),
							g.Text("  Deployment Failed!").Bold().FG(g.Red),
							g.Text(&s.errorMsg).FG(g.BrightWhite).MarginTRBL(0, 2, 0, 2),
							g.Space(),
							g.HBox.Gap(4)(
								g.SpaceW(2),
								g.Text("[r] Retry").Bold().FG(g.Yellow),
								g.Text("[c] Cancel").Bold().FG(g.BrightBlack),
							),
							g.SpaceH(1),
						),
					),
				),
				g.SpaceH(1),
				g.Text("  q to quit").Dim(),
			))
			s.startDeploy(app)
			app.Go("dashboard")
		}
	}).Handle("c", func() {
		if s.showError {
			s.cancelDeploy()
			s.showError = false
			s.progress = 0
			s.currentStep = 0
			s.logText = ""
			app.Go("form")
		}
	})

	app.RunFrom("form")
}

func buildStepRow(s *state, idx int) any {
	return g.HBox.Gap(1)(
		g.If(&s.currentStep).Eq(idx).Then(
			g.Spinner(&s.spinFrame).FG(g.Cyan),
		).Else(
			g.If(&s.showError).Eq(true).Then(
				// when error is showing, mark completed steps with a check
				g.IfOrd(&s.currentStep).Gt(idx).Then(
					g.Text("✓").FG(g.Green),
				).Else(
					g.Text("○").FG(g.BrightBlack),
				),
			).Else(
				g.IfOrd(&s.currentStep).Gt(idx).Then(
					g.Text("✓").FG(g.Green),
				).Else(
					g.Text("○").FG(g.BrightBlack),
				),
			),
		),
		g.If(&s.currentStep).Eq(idx).Then(
			g.Text(&s.stepLabels[idx]).Bold().FG(g.White),
		).Else(
			g.IfOrd(&s.currentStep).Gt(idx).Then(
				g.Text(&s.stepLabels[idx]).FG(g.Green),
			).Else(
				g.Text(&s.stepLabels[idx]).Dim(),
			),
		),
	)
}

func (s *state) cancelDeploy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deploying && s.stopDeploy != nil {
		close(s.stopDeploy)
		s.deploying = false
	}
}

func (s *state) startDeploy(app *g.App) {
	s.mu.Lock()
	// reset state
	s.currentStep = 0
	s.progress = 0
	s.showError = false
	s.logText = "0%"
	s.deploying = true
	s.stopDeploy = make(chan struct{})
	stop := s.stopDeploy
	s.mu.Unlock()

	envName := "staging"
	if s.envChoice == 1 {
		envName = "production"
	}

	go func() {
		// spinner ticker
		spinTick := time.NewTicker(80 * time.Millisecond)
		defer spinTick.Stop()

		go func() {
			for {
				select {
				case <-stop:
					return
				case <-spinTick.C:
					s.spinFrame++
					app.RequestRender()
				}
			}
		}()

		writeLog := func(msg string) {
			s.logWriter.Write([]byte(msg + "\n"))
		}

		writeLog(fmt.Sprintf("Starting deployment of %q to %s", s.releaseName, envName))

		// each step: 25% of progress, ~1 second with smooth increments
		for stepIdx := 0; stepIdx < 4; stepIdx++ {
			select {
			case <-stop:
				return
			default:
			}

			s.currentStep = stepIdx
			writeLog(fmt.Sprintf("[%s] Starting %s...", time.Now().Format("15:04:05"), steps[stepIdx]))
			app.RequestRender()

			// simulate failure at 75%
			if s.progress >= 70 && stepIdx == 3 {
				time.Sleep(500 * time.Millisecond)
				s.progress = 75
				s.logText = fmt.Sprintf("%d%%", s.progress)
				writeLog(fmt.Sprintf("[%s] ERROR: %s failed - connection timeout", time.Now().Format("15:04:05"), steps[stepIdx]))
				app.RequestRender()

				time.Sleep(300 * time.Millisecond)
				s.mu.Lock()
				s.showError = true
				s.errorMsg = fmt.Sprintf("  %s step failed: connection timeout to %s", steps[stepIdx], envName)
				s.deploying = false
				s.mu.Unlock()
				app.RequestRender()
				return
			}

			// smooth progress over ~1 second
			targetPct := (stepIdx + 1) * 25
			startPct := s.progress
			ticks := 20
			for i := 1; i <= ticks; i++ {
				select {
				case <-stop:
					return
				default:
				}
				s.progress = startPct + (targetPct-startPct)*i/ticks
				s.logText = fmt.Sprintf("%d%%", s.progress)
				app.RequestRender()
				time.Sleep(50 * time.Millisecond)
			}

			writeLog(fmt.Sprintf("[%s] %s complete", time.Now().Format("15:04:05"), steps[stepIdx]))
			app.RequestRender()
		}

		// success (this path only reached if no failure)
		s.mu.Lock()
		s.deploying = false
		s.mu.Unlock()
		writeLog(fmt.Sprintf("[%s] Deployment complete!", time.Now().Format("15:04:05")))
		app.RequestRender()
	}()
}
