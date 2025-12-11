package plot

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"runtime"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

// SeriesData holds a single time series.
type SeriesData struct {
	Name string
	X    []time.Time
	Y    []float64
}

// PlotInteractive renders a plot and opens it in the default viewer.
func PlotInteractive(seriesList []SeriesData) error {
	p := plot.New()
	p.Title.Text = "Aqualog Plot"
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Values"

	for i, s := range seriesList {
		pts := make(plotter.XYs, len(s.X))
		for j := range pts {
			pts[j].X = float64(s.X[j].Unix())
			pts[j].Y = s.Y[j]
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return err
		}

		// Auto colours
		line.Color = plotutilColor(i)
		p.Add(line)
		p.Legend.Add(s.Name, line)
	}

	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04"}

	// Render plot to an in-memory PNG
	width := 10 * vg.Inch
	height := 5 * vg.Inch

	img := vgimg.New(width, height)
	dc := draw.New(img)
	p.Draw(dc)

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "aqualog_plot_*.png")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := (&vgimg.PngCanvas{Canvas: img}).WriteTo(tmpFile); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}

	// Open using system viewer
	if err := openFile(tmpFile.Name()); err != nil {
		return fmt.Errorf("opening viewer: %w", err)
	}

	return nil
}

// openFile attempts to open a file with the system default program.
func openFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// simple color generator (gonum’s plotutil.Color without dependencies)
func plotutilColor(i int) color.Color {
	palette := []color.Color{
		color.RGBA{31, 119, 180, 255},
		color.RGBA{255, 127, 14, 255},
		color.RGBA{44, 160, 44, 255},
		color.RGBA{214, 39, 40, 255},
		color.RGBA{148, 103, 189, 255},
		color.RGBA{140, 86, 75, 255},
	}
	return palette[i%len(palette)]
}
