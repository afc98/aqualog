package plot

import (
	"encoding/json"
	"fmt"
	"html"
	"image/color"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	goplot "gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// SeriesData holds a single plottable time series.
type SeriesData struct {
	Name string
	Unit string
	X    []time.Time
	Y    []float64
}

// Options controls plot rendering.
type Options struct {
	Title  string
	Format string
	Output string
	Open   bool
	Width  float64
	Height float64
}

// Render writes a plot to disk and optionally opens it in the default viewer.
func Render(seriesList []SeriesData, opts Options) (string, error) {
	if err := validateSeries(seriesList); err != nil {
		return "", err
	}

	format := strings.ToLower(opts.Format)
	if format == "" {
		format = formatFromOutput(opts.Output)
	}
	if format == "" {
		format = "png"
	}
	if !isSupportedFormat(format) {
		return "", fmt.Errorf("unsupported plot format %q; use png, svg, pdf, jpg, jpeg, tif, tiff, or html", format)
	}

	output := opts.Output
	if output == "" {
		tmpFile, err := os.CreateTemp("", "aqualog_plot_*."+format)
		if err != nil {
			return "", fmt.Errorf("creating temp file: %w", err)
		}
		output = tmpFile.Name()
		if err := tmpFile.Close(); err != nil {
			return "", fmt.Errorf("closing temp file: %w", err)
		}
	}

	if format == "html" {
		if err := writeHTMLPlot(output, seriesList, opts); err != nil {
			return "", err
		}
	} else if err := writeStaticPlot(output, seriesList, opts); err != nil {
		return "", err
	}

	if opts.Open {
		if err := openFile(output); err != nil {
			return output, fmt.Errorf("opening viewer: %w", err)
		}
	}

	return output, nil
}

func validateSeries(seriesList []SeriesData) error {
	if len(seriesList) == 0 {
		return fmt.Errorf("no series requested")
	}
	for _, s := range seriesList {
		if s.Name == "" {
			return fmt.Errorf("series name is required")
		}
		if len(s.X) != len(s.Y) {
			return fmt.Errorf("series %q has mismatched time/value lengths", s.Name)
		}
		if len(s.X) == 0 {
			return fmt.Errorf("series %q has no plottable points", s.Name)
		}
	}
	return nil
}

func writeStaticPlot(path string, seriesList []SeriesData, opts Options) error {
	p := goplot.New()
	p.Title.Text = title(opts)
	p.X.Label.Text = "Time"
	p.Y.Label.Text = yLabel(seriesList)
	p.X.Tick.Marker = goplot.TimeTicks{Format: "2006-01-02\n15:04"}

	for i, s := range seriesList {
		pts := make(plotter.XYs, len(s.X))
		for j := range pts {
			pts[j].X = float64(s.X[j].Unix())
			pts[j].Y = s.Y[j]
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return fmt.Errorf("creating line for %s: %w", s.Name, err)
		}
		line.Color = paletteColor(i)
		p.Add(line)
		p.Legend.Add(label(s), line)
	}

	width, height := dimensions(opts)
	if err := p.Save(vg.Length(width)*vg.Inch, vg.Length(height)*vg.Inch, path); err != nil {
		return fmt.Errorf("writing plot %s: %w", path, err)
	}
	return nil
}

func writeHTMLPlot(path string, seriesList []SeriesData, opts Options) error {
	doc := htmlDocument(seriesList, opts)
	if err := os.WriteFile(path, []byte(doc), 0644); err != nil {
		return fmt.Errorf("writing html plot %s: %w", path, err)
	}
	return nil
}

func htmlDocument(seriesList []SeriesData, opts Options) string {
	type point struct {
		T string  `json:"t"`
		X int64   `json:"x"`
		Y float64 `json:"y"`
	}
	type htmlSeries struct {
		Name   string  `json:"name"`
		Unit   string  `json:"unit"`
		Color  string  `json:"color"`
		Points []point `json:"points"`
	}

	var data []htmlSeries
	minX, maxX, minY, maxY := bounds(seriesList)
	for i, s := range seriesList {
		hs := htmlSeries{Name: s.Name, Unit: s.Unit, Color: colorHex(paletteColor(i))}
		for j := range s.X {
			hs.Points = append(hs.Points, point{
				T: s.X[j].Format(time.RFC3339),
				X: s.X[j].Unix(),
				Y: s.Y[j],
			})
		}
		data = append(data, hs)
	}
	payload, _ := json.Marshal(data)
	width, height := dimensions(opts)

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
body{margin:0;font-family:Arial,sans-serif;color:#1f2933;background:#f7f9fb}
.wrap{padding:20px}
h1{font-size:20px;font-weight:600;margin:0 0 12px}
#plot{width:100%%;height:min(78vh,%dpx);background:white;border:1px solid #d8dee4}
.legend{display:flex;flex-wrap:wrap;gap:12px;margin-top:10px;font-size:13px}
.item{display:flex;align-items:center;gap:6px}
.swatch{width:22px;height:3px}
.tip{position:fixed;display:none;background:#111827;color:white;padding:7px 9px;border-radius:4px;font-size:12px;pointer-events:none;white-space:nowrap}
</style>
</head>
<body>
<div class="wrap">
<h1>%s</h1>
<svg id="plot" viewBox="0 0 %d %d" role="img" aria-label="%s"></svg>
<div id="legend" class="legend"></div>
<div id="tip" class="tip"></div>
</div>
<script>
const series=%s;
const bounds={minX:%d,maxX:%d,minY:%g,maxY:%g};
const svg=document.getElementById("plot");
const legend=document.getElementById("legend");
const tip=document.getElementById("tip");
const W=%d,H=%d,pad={l:70,r:25,t:20,b:60};
function sx(x){return pad.l+(x-bounds.minX)*(W-pad.l-pad.r)/(bounds.maxX-bounds.minX||1)}
function sy(y){return H-pad.b-(y-bounds.minY)*(H-pad.t-pad.b)/(bounds.maxY-bounds.minY||1)}
function el(n,a){const e=document.createElementNS("http://www.w3.org/2000/svg",n);for(const k in a)e.setAttribute(k,a[k]);return e}
svg.appendChild(el("line",{x1:pad.l,y1:H-pad.b,x2:W-pad.r,y2:H-pad.b,stroke:"#6b7280"}));
svg.appendChild(el("line",{x1:pad.l,y1:pad.t,x2:pad.l,y2:H-pad.b,stroke:"#6b7280"}));
for(let i=0;i<=5;i++){
 const y=bounds.minY+(bounds.maxY-bounds.minY)*i/5, yy=sy(y);
 svg.appendChild(el("line",{x1:pad.l,y1:yy,x2:W-pad.r,y2:yy,stroke:"#edf1f5"}));
 const tx=el("text",{x:pad.l-8,y:yy+4,"text-anchor":"end","font-size":"12",fill:"#4b5563"});tx.textContent=y.toFixed(2);svg.appendChild(tx);
}
for(let i=0;i<=5;i++){
 const x=bounds.minX+(bounds.maxX-bounds.minX)*i/5, xx=sx(x);
 const d=new Date(x*1000).toISOString().slice(0,10);
 const tx=el("text",{x:xx,y:H-pad.b+22,"text-anchor":"middle","font-size":"12",fill:"#4b5563"});tx.textContent=d;svg.appendChild(tx);
}
for(const s of series){
 const path=s.points.map((p,i)=>(i?"L":"M")+sx(p.x).toFixed(2)+","+sy(p.y).toFixed(2)).join(" ");
 svg.appendChild(el("path",{d:path,fill:"none",stroke:s.color,"stroke-width":"2"}));
 for(const p of s.points){
  const c=el("circle",{cx:sx(p.x),cy:sy(p.y),r:3,fill:s.color,opacity:"0"});
  c.addEventListener("mouseenter",ev=>{tip.style.display="block";tip.textContent=s.name+" "+p.t+" "+p.y.toFixed(4)+(s.unit?" "+s.unit:"")});
  c.addEventListener("mousemove",ev=>{tip.style.left=(ev.clientX+12)+"px";tip.style.top=(ev.clientY+12)+"px"});
  c.addEventListener("mouseleave",()=>tip.style.display="none");
  svg.appendChild(c);
 }
 const li=document.createElement("div");li.className="item";li.innerHTML='<span class="swatch" style="background:'+s.color+'"></span><span>'+s.name+(s.unit?' ('+s.unit+')':'')+'</span>';legend.appendChild(li);
}
</script>
</body>
</html>`,
		html.EscapeString(title(opts)),
		int(height*96),
		html.EscapeString(title(opts)),
		int(width*96),
		int(height*96),
		html.EscapeString(title(opts)),
		string(payload),
		minX, maxX, minY, maxY,
		int(width*96), int(height*96),
	)
}

func bounds(seriesList []SeriesData) (int64, int64, float64, float64) {
	minX, maxX := seriesList[0].X[0].Unix(), seriesList[0].X[0].Unix()
	minY, maxY := seriesList[0].Y[0], seriesList[0].Y[0]
	for _, s := range seriesList {
		for i := range s.X {
			x := s.X[i].Unix()
			y := s.Y[i]
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if minX == maxX {
		minX--
		maxX++
	}
	if minY == maxY {
		delta := math.Max(math.Abs(minY)*0.05, 1)
		minY -= delta
		maxY += delta
	}
	return minX, maxX, minY, maxY
}

func title(opts Options) string {
	if opts.Title != "" {
		return opts.Title
	}
	return "Aqualog Plot"
}

func yLabel(seriesList []SeriesData) string {
	units := map[string]bool{}
	for _, s := range seriesList {
		if s.Unit != "" {
			units[s.Unit] = true
		}
	}
	if len(units) == 0 {
		return "Value"
	}
	if len(units) == 1 {
		for unit := range units {
			return "Value (" + unit + ")"
		}
	}
	keys := make([]string, 0, len(units))
	for unit := range units {
		keys = append(keys, unit)
	}
	sort.Strings(keys)
	return "Value (" + strings.Join(keys, ", ") + ")"
}

func label(s SeriesData) string {
	if s.Unit == "" {
		return s.Name
	}
	return fmt.Sprintf("%s (%s)", s.Name, s.Unit)
}

func dimensions(opts Options) (float64, float64) {
	width := opts.Width
	height := opts.Height
	if width <= 0 {
		width = 10
	}
	if height <= 0 {
		height = 5
	}
	return width, height
}

func formatFromOutput(path string) string {
	if path == "" {
		return ""
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if ext == "htm" {
		return "html"
	}
	return ext
}

func isSupportedFormat(format string) bool {
	switch format {
	case "png", "svg", "pdf", "jpg", "jpeg", "tif", "tiff", "html":
		return true
	default:
		return false
	}
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

func paletteColor(i int) color.Color {
	palette := []color.Color{
		color.RGBA{31, 119, 180, 255},
		color.RGBA{255, 127, 14, 255},
		color.RGBA{44, 160, 44, 255},
		color.RGBA{214, 39, 40, 255},
		color.RGBA{148, 103, 189, 255},
		color.RGBA{140, 86, 75, 255},
		color.RGBA{227, 119, 194, 255},
		color.RGBA{127, 127, 127, 255},
	}
	return palette[i%len(palette)]
}

func colorHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
