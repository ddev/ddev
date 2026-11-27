package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayState is what the dot on the tray icon shows.
type trayState int

const (
	// trayIdle has no dot: every project is stopped.
	trayIdle trayState = iota
	// trayRunning is a green dot: a project runs.
	trayRunning
	// trayBusy is a yellow dot: an action runs, or a project is starting.
	trayBusy
)

var dotColors = map[trayState]color.NRGBA{
	trayRunning: {0x2e, 0xcc, 0x71, 0xff},
	trayBusy:    {0xf5, 0xb0, 0x1a, 0xff},
}

// stateOf sums the projects and the running jobs up in one state.
func stateOf(projects []project, runningJobs int) trayState {
	if runningJobs > 0 {
		return trayBusy
	}
	state := trayIdle
	for _, p := range projects {
		if strings.HasPrefix(p.Status, "starting") {
			return trayBusy
		}
		if p.Up {
			state = trayRunning
		}
	}
	return state
}

// withDot returns the PNG icon with a dot of c at its bottom right, cut
// out of the mark by a transparent ring so that it reads at tray size.
func withDot(icon []byte, c color.NRGBA) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	img := image.NewNRGBA(b)
	draw.Draw(img, b, src, b.Min, draw.Src)
	size := float64(min(b.Dx(), b.Dy()))
	r := size * 0.18
	ring := r + size*0.07
	cx, cy := float64(b.Max.X)-r-1, float64(b.Max.Y)-r-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			// Coverage, with a pixel of antialiasing at each edge.
			dot := clamp01(r + 0.5 - d)
			keep := clamp01(d - ring + 0.5)
			if dot == 0 && keep == 1 {
				continue
			}
			p := img.NRGBAAt(x, y)
			under := float64(p.A) / 255 * keep * (1 - dot)
			a := dot + under
			if a == 0 {
				img.SetNRGBA(x, y, color.NRGBA{})
				continue
			}
			mix := func(dc, pc uint8) uint8 {
				return uint8(math.Round((float64(dc)*dot + float64(pc)*under) / a))
			}
			img.SetNRGBA(x, y, color.NRGBA{mix(c.R, p.R), mix(c.G, p.G), mix(c.B, p.B), uint8(math.Round(a * 255))})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// scaleDown returns the PNG icon at size by size pixels, each the average
// of the source area it covers. Shrinking thins lines into faint partial
// pixels, so their opacity rises with the factor, from 2x shrinking on.
func scaleDown(icon []byte, size int) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	fx, fy := float64(b.Dx())/float64(size), float64(b.Dy())/float64(size)
	if fx < 1 || fy < 1 {
		return icon, nil
	}
	boost := max(1, fx/2)
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			var r, g, bl, a, area float64
			for sy := int(float64(y) * fy); float64(sy) < float64(y+1)*fy; sy++ {
				wy := min(float64(sy+1), float64(y+1)*fy) - max(float64(sy), float64(y)*fy)
				for sx := int(float64(x) * fx); float64(sx) < float64(x+1)*fx; sx++ {
					wx := min(float64(sx+1), float64(x+1)*fx) - max(float64(sx), float64(x)*fx)
					w := wx * wy
					// Premultiplied, so that clear pixels add no color.
					c := color.NRGBAModel.Convert(src.At(b.Min.X+sx, b.Min.Y+sy)).(color.NRGBA)
					ca := float64(c.A) / 255
					r += float64(c.R) * ca * w
					g += float64(c.G) * ca * w
					bl += float64(c.B) * ca * w
					a += ca * w
					area += w
				}
			}
			if a == 0 {
				continue
			}
			alpha := min(1, a/area*boost)
			img.SetNRGBA(x, y, color.NRGBA{uint8(math.Round(r / a)), uint8(math.Round(g / a)), uint8(math.Round(bl / a)), uint8(math.Round(alpha * 255))})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func clamp01(v float64) float64 {
	return max(0, min(1, v))
}

// trayIcons sets the tray's icon for each state. The Windows and Linux
// trays show the color mark, which reads on light and dark taskbars. macOS
// draws a menu bar icon from its shape alone, as a template, so there the
// dot is a colored label beside the icon instead.
type trayIcons struct {
	tray  *application.SystemTray
	goos  string
	icons map[trayState][]byte
	state trayState
}

// newTrayIcons sets the tray's first icon. set replaces it once the app
// runs.
func newTrayIcons(tray *application.SystemTray, goos string) *trayIcons {
	t := &trayIcons{tray: tray, goos: goos, state: -1}
	if goos == "darwin" {
		tray.SetTemplateIcon(trayIcon)
		application.SystemTrayLabelParser = parseDotLabel
		return t
	}
	tray.SetIcon(trayIconColor)
	return t
}

// load makes the icon of each state. Windows shrinks a tray icon without
// smoothing, which breaks the mark's thin lines, so there each is of the
// size it shows, which Windows tells only once the app runs.
func (t *trayIcons) load() {
	t.icons = map[trayState][]byte{trayIdle: trayIconColor}
	for s, c := range dotColors {
		var err error
		if t.icons[s], err = withDot(trayIconColor, c); err != nil {
			t.icons[s] = trayIconColor
		}
	}
	if size := traySize(); size > 0 {
		for s, icon := range t.icons {
			if small, err := scaleDown(icon, size); err == nil {
				t.icons[s] = small
			}
		}
	}
}

// set shows state, once the app runs.
func (t *trayIcons) set(state trayState) {
	if state == t.state {
		return
	}
	t.state = state
	if t.goos != "darwin" {
		if t.icons == nil {
			t.load()
		}
		t.tray.SetIcon(t.icons[state])
		return
	}
	label := ""
	if c, ok := dotColors[state]; ok {
		label = fmt.Sprintf("\033[38;2;%d;%d;%dm●\033[0m", c.R, c.G, c.B)
	}
	t.tray.SetLabel(label)
}

// parseDotLabel reads the colored dot that set gives the macOS tray.
// Wails' own parser drops the colors.
func parseDotLabel(label string) ([]application.SystemTrayLabelPart, error) {
	var r, g, b int
	var text string
	if _, err := fmt.Sscanf(label, "\033[38;2;%d;%d;%dm%s", &r, &g, &b, &text); err != nil {
		return nil, err
	}
	return []application.SystemTrayLabelPart{{
		Text:    strings.TrimSuffix(text, "\033[0m"),
		FgColor: fmt.Sprintf("#%02x%02x%02x", r, g, b),
	}}, nil
}
