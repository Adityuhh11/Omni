package glass

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlacements(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	tests := []struct {
		name string
		fit  fit
		size image.Point
		area image.Rectangle
		want []image.Rectangle
	}{
		{"fill cuts a square to the screen's width", fitFill, image.Pt(1000, 1000), screen, []image.Rectangle{image.Rect(0, -420, 1920, 1500)}},
		{"contain leaves bands either side", fitContain, image.Pt(1000, 1000), screen, []image.Rectangle{image.Rect(420, 0, 1500, 1080)}},
		{"stretch is the area", fitStretch, image.Pt(10, 10), screen, []image.Rectangle{screen}},
		{"center keeps the size", fitCenter, image.Pt(100, 50), screen, []image.Rectangle{image.Rect(910, 515, 1010, 565)}},
		{"tile repeats from the top left", fitTile, image.Pt(1000, 600), screen, []image.Rectangle{
			image.Rect(0, 0, 1000, 600), image.Rect(1000, 0, 2000, 600),
			image.Rect(0, 600, 1000, 1200), image.Rect(1000, 600, 2000, 1200),
		}},
		{"a second screen's area starts where it is", fitStretch, image.Pt(10, 10), image.Rect(1920, 0, 3840, 1080), []image.Rectangle{image.Rect(1920, 0, 3840, 1080)}},
		{"an empty picture is nowhere", fitFill, image.Pt(0, 0), screen, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := placements(tt.fit, tt.size, tt.area); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("placements = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderKeepsTheScreenShapeAndColour(t *testing.T) {
	red := color.RGBA{200, 10, 10, 255}
	pic := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range pic.Pix {
		pic.Pix[i] = []uint8{red.R, red.G, red.B, red.A}[i%4]
	}
	d := desktop{fit: fitFill, color: color.RGBA{0, 0, 255, 255}, screen: image.Rect(0, 0, 1920, 1080), area: image.Rect(0, 0, 1920, 1080)}
	out := render(d, pic)
	if got := out.Bounds().Size(); got != image.Pt(120, 67) {
		t.Fatalf("size %v, want 120×67", got)
	}
	if got := out.RGBAAt(60, 33); got != red {
		t.Errorf("middle %v, want the picture's %v", got, red)
	}
	plain := render(desktop{color: d.color, screen: d.screen, area: d.area}, nil)
	if got := plain.RGBAAt(5, 5); got != d.color {
		t.Errorf("a plain colour desktop is %v, want %v", got, d.color)
	}
}

func TestWallpaperGlassPlacesTheScreenFromTheWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wall.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	d := desktop{
		path: path, fit: fitFill, screen: image.Rect(1920, 0, 3840, 1080), area: image.Rect(1920, 0, 3840, 1080),
		origin: image.Pt(2120, 300), scale: 1.25,
	}
	g, err := wallpaperGlass(d)
	if err != nil {
		t.Fatal(err)
	}
	if g.Look != Wallpaper || !strings.HasPrefix(g.Image, "data:image/png;base64,") {
		t.Fatalf("got %q with image %.30q", g.Look, g.Image)
	}
	if g.Left != -160 || g.Top != -240 || g.Width != 1536 || g.Height != 864 {
		t.Errorf("screen at %v,%v size %v×%v; want -160,-240 size 1536×864", g.Left, g.Top, g.Width, g.Height)
	}
	d.path, d.fallback = filepath.Join(t.TempDir(), "gone.jpg"), ""
	if _, err := wallpaperGlass(d); err == nil {
		t.Error("a missing wallpaper with no fallback gave glass")
	}
	d.fallback = path
	if _, err := wallpaperGlass(d); err != nil {
		t.Errorf("the fallback was not read: %v", err)
	}
}
