package main

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	. "github.com/kungfusheep/glyph"
)

type Process struct {
	Name  string
	PID   int
	Usage float64
}

func main() {
	var (
		tab      int
		maxUsage float64

		cpuData = make([]float64, 20)
		memData = make([]float64, 20)
		netData = make([]float64, 20)

		cpuProcs []Process
		memProcs []Process
		netProcs []Process

		statusText string
	)

	genProcs := func() []Process {
		names := []string{"chrome", "node", "postgres", "nginx", "redis"}
		procs := make([]Process, len(names))
		for i, name := range names {
			procs[i] = Process{
				Name:  name,
				PID:   1000 + rand.Intn(9000),
				Usage: float64(rand.Intn(95) + 5),
			}
		}
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].Usage > procs[j].Usage
		})
		return procs
	}

	// initial data
	cpuProcs = genProcs()
	memProcs = genProcs()
	netProcs = genProcs()

	updateStatus := func() {
		maxUsage = 0
		allProcs := [][]Process{cpuProcs, memProcs, netProcs}
		for _, procs := range allProcs {
			for _, p := range procs {
				if p.Usage > maxUsage {
					maxUsage = p.Usage
				}
			}
		}
		if maxUsage > 80 {
			statusText = fmt.Sprintf("WARNING: usage at %.0f%%", maxUsage)
		} else {
			statusText = "All systems nominal"
		}
	}
	updateStatus()

	shiftAppend := func(data []float64, val float64) {
		copy(data, data[1:])
		data[len(data)-1] = val
	}

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			Tabs([]string{"CPU", "Memory", "Network"}, &tab).
				ActiveStyle(Style{FG: Cyan, Attr: AttrBold}).
				InactiveStyle(Style{FG: BrightBlack}),
			HRule().FG(BrightBlack),

			Switch(&tab).
				Case(0, VBox.Grow(1).Border(BorderRounded).Title("CPU")(
					Sparkline(&cpuData).FG(Green),
					SpaceH(1),
					AutoTable(&cpuProcs).
						Columns("Name", "PID", "Usage").
						Column("Usage", Percent(1)).
						HeaderStyle(Style{Attr: AttrBold, FG: Cyan}).
						SortBy("Usage", false),
				)).
				Case(1, VBox.Grow(1).Border(BorderRounded).Title("Memory")(
					Sparkline(&memData).FG(Yellow),
					SpaceH(1),
					AutoTable(&memProcs).
						Columns("Name", "PID", "Usage").
						Column("Usage", Percent(1)).
						HeaderStyle(Style{Attr: AttrBold, FG: Yellow}).
						SortBy("Usage", false),
				)).
				Case(2, VBox.Grow(1).Border(BorderRounded).Title("Network")(
					Sparkline(&netData).FG(Magenta),
					SpaceH(1),
					AutoTable(&netProcs).
						Columns("Name", "PID", "Usage").
						Column("Usage", Percent(1)).
						HeaderStyle(Style{Attr: AttrBold, FG: Magenta}).
						SortBy("Usage", false),
				)).
				End(),

			IfOrd(&maxUsage).Gt(80).
				Then(Text(&statusText).Bold().FG(Red)).
				Else(Text(&statusText).FG(Green)),
		),
	)

	app.Handle("q", app.Stop)
	app.Handle("1", func() { tab = 0 })
	app.Handle("2", func() { tab = 1 })
	app.Handle("3", func() { tab = 2 })

	go func() {
		for range time.NewTicker(500 * time.Millisecond).C {
			cpuVal := 30 + rand.Float64()*60
			memVal := 40 + rand.Float64()*50
			netVal := 10 + rand.Float64()*80

			shiftAppend(cpuData, cpuVal)
			shiftAppend(memData, memVal)
			shiftAppend(netData, netVal)

			cpuProcs = genProcs()
			memProcs = genProcs()
			netProcs = genProcs()

			updateStatus()
			app.RequestRender()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
