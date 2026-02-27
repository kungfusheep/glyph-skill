package main

import (
	"log"
	"math/rand/v2"
	"time"

	. "github.com/kungfusheep/glyph"
)

type Process struct {
	Name  string
	PID   int
	Usage int
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// tab state
	tab := 0

	// per-tab sparkline data
	cpuPoints := make([]float64, 20)
	memPoints := make([]float64, 20)
	netPoints := make([]float64, 20)

	// per-tab process tables
	cpuProcs := generateProcs("cpu")
	memProcs := generateProcs("mem")
	netProcs := generateProcs("net")

	// status bar: track the max usage across all tabs
	maxUsage := 0
	statusOk := "All systems nominal"
	statusWarn := "WARNING: usage exceeds 80%"

	app.SetView(
		VBox.Gap(1).Border(BorderRounded).Title("System Monitor")(
			Tabs([]string{"CPU", "Memory", "Network"}, &tab).Kind(TabsStyleUnderline),
			Switch(&tab).
				Case(0, VBox.Gap(1)(
					Sparkline(&cpuPoints).Width(60).FG(Cyan),
					AutoTable(&cpuProcs).
						Columns("Name", "PID", "Usage").
						Headers("Name", "PID", "Usage%").
						SortBy("Usage", false).
						HeaderStyle(Style{Attr: AttrBold, FG: Cyan}),
				)).
				Case(1, VBox.Gap(1)(
					Sparkline(&memPoints).Width(60).FG(Green),
					AutoTable(&memProcs).
						Columns("Name", "PID", "Usage").
						Headers("Name", "PID", "Usage%").
						SortBy("Usage", false).
						HeaderStyle(Style{Attr: AttrBold, FG: Green}),
				)).
				Case(2, VBox.Gap(1)(
					Sparkline(&netPoints).Width(60).FG(Magenta),
					AutoTable(&netProcs).
						Columns("Name", "PID", "Usage").
						Headers("Name", "PID", "Usage%").
						SortBy("Usage", false).
						HeaderStyle(Style{Attr: AttrBold, FG: Magenta}),
				)),
			HRule().FG(BrightBlack),
			IfOrd(&maxUsage).Gt(80).
				Then(Text(&statusWarn).Bold().FG(Red)).
				Else(Text(&statusOk).FG(Green)),
		),
	)

	app.Handle("1", func() { tab = 0 })
	app.Handle("2", func() { tab = 1 })
	app.Handle("3", func() { tab = 2 })
	app.Handle("q", app.Stop)

	// background data refresh
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			pushPoint(&cpuPoints, rand.Float64()*100)
			pushPoint(&memPoints, rand.Float64()*100)
			pushPoint(&netPoints, rand.Float64()*100)

			cpuProcs = generateProcs("cpu")
			memProcs = generateProcs("mem")
			netProcs = generateProcs("net")

			// compute max usage across all processes
			peak := 0
			for _, sets := range [][]Process{cpuProcs, memProcs, netProcs} {
				for _, p := range sets {
					if p.Usage > peak {
						peak = p.Usage
					}
				}
			}
			maxUsage = peak

			app.RequestRender()
		}
	}()

	app.Run()
}

func pushPoint(pts *[]float64, v float64) {
	*pts = append((*pts)[1:], v)
}

var procNames = [][]string{
	{"nginx", "postgres", "redis", "node", "go-api"},
	{"chrome", "vscode", "docker", "kubelet", "etcd"},
	{"sshd", "envoy", "haproxy", "coredns", "promtail"},
}

func generateProcs(kind string) []Process {
	var names []string
	switch kind {
	case "cpu":
		names = procNames[0]
	case "mem":
		names = procNames[1]
	default:
		names = procNames[2]
	}

	procs := make([]Process, 5)
	for i, name := range names {
		procs[i] = Process{
			Name:  name,
			PID:   1000 + rand.IntN(9000),
			Usage: rand.IntN(100),
		}
	}
	return procs
}
