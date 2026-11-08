package glass

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	_ "image/gif" // registers the decoders image.Decode picks from
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"strconv"
	"sync"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// When the system will not blur (Windows' energy saver), the glass is the wallpaper under the window, blurred here: what Windows'
// own Mica shows. It is tinted by the wallpaper rather than by the windows behind, and costs one small picture per wallpaper.

// fit is how a wallpaper is laid over the screen.
type fit int

const (
	fitFill    fit = iota // covers the area, cut to it, centred
	fitContain            // inside the area whole, centred, the colour around it
	fitStretch            // the area's shape exactly
	fitCenter             // its own size, centred
	fitTile               // its own size, repeated from the area's top left
)

const (
	shrink        = 16          // the screen is drawn at a 16th of its size: small to send, and the page's scaling up softens it further
	blurRadius    = 2           // in shrunk pixels, three passes: a blur of about 100 screen pixels
	maxPixels     = 100_000_000 // a wallpaper larger than this is refused rather than decoded into gigabytes
	cachedScreens = 8
)

// desktop is how the wallpaper lies on the screen a window is on, in the screen's pixels.
type desktop struct {
	path     string // the wallpaper's file; empty for a plain colour
	fallback string // Windows' own copy of the current wallpaper, read when path cannot be (a slideshow's folder, a moved file)
	fit      fit
	color    color.RGBA      // the colour around and under the picture
	screen   image.Rectangle // the window's screen
	area     image.Rectangle // what the picture is laid over: the screen, or every screen for a picture spanning them
	origin   image.Point     // the window's top left (its client area)
	scale    float64         // screen pixels per CSS pixel
}

var (
	pictures   = map[string]string{} // rendered screens as data: URLs, by what they were made from
	picturesMu sync.Mutex
)

// wallpaperGlass is the glass for a window on d: its screen's wallpaper blurred, and where the screen lies from the window.
func wallpaperGlass(d desktop) (Glass, error) {
	file, key := "", fmt.Sprint(d.fit, d.color, d.screen, d.area)
	if d.path != "" {
		for _, p := range []string{d.path, d.fallback} {
			if info, err := os.Stat(p); p != "" && err == nil && info.Mode().IsRegular() {
				file, key = p, key+p+strconv.FormatInt(info.Size(), 10)+strconv.FormatInt(info.ModTime().UnixNano(), 10)
				break
			}
		}
		if file == "" {
			return Glass{}, fmt.Errorf("glass: the wallpaper %s cannot be read", d.path)
		}
	}
	picturesMu.Lock()
	url, ok := pictures[key]
	picturesMu.Unlock()
	if !ok {
		var pic image.Image
		if file != "" {
			var err error
			if pic, err = decode(file); err != nil {
				return Glass{}, fmt.Errorf("glass: the wallpaper: %w", err)
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(d, pic)); err != nil {
			return Glass{}, fmt.Errorf("glass: encode the wallpaper: %w", err)
		}
		url = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		picturesMu.Lock()
		if len(pictures) >= cachedScreens {
			clear(pictures) // a wallpaper changed several times, or many screens: start again
		}
		pictures[key] = url
		picturesMu.Unlock()
	}
	return Glass{
		Look:   Wallpaper,
		Image:  url,
		Left:   float64(d.screen.Min.X-d.origin.X) / d.scale,
		Top:    float64(d.screen.Min.Y-d.origin.Y) / d.scale,
		Width:  float64(d.screen.Dx()) / d.scale,
		Height: float64(d.screen.Dy()) / d.scale,
	}, nil
}

func decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, fmt.Errorf("%d×%d is too large", cfg.Width, cfg.Height)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	pic, _, err := image.Decode(f)
	return pic, err
}

// render draws d's screen at a 16th of its size, the picture laid on its colour as Windows lays it, and blurs it.
func render(d desktop, pic image.Image) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, max(1, d.screen.Dx()/shrink), max(1, d.screen.Dy()/shrink)))
	stddraw.Draw(out, out.Bounds(), &image.Uniform{d.color}, image.Point{}, stddraw.Src)
	if pic != nil {
		down := func(v, from int) int { return int(math.Floor(float64(v-from) / shrink)) }
		for _, at := range placements(d.fit, pic.Bounds().Size(), d.area) {
			r := image.Rect(down(at.Min.X, d.screen.Min.X), down(at.Min.Y, d.screen.Min.Y), down(at.Max.X, d.screen.Min.X), down(at.Max.Y, d.screen.Min.Y))
			if r.Empty() {
				r.Max = r.Min.Add(image.Pt(1, 1)) // a tile smaller than a shrunk pixel still adds its colour
			}
			draw.BiLinear.Scale(out, r, pic, pic.Bounds(), draw.Over, nil)
		}
	}
	for range 3 { // three box blurs are close to a Gaussian
		boxBlur(out, blurRadius)
	}
	return out
}

// placements is where a picture of size goes over area, in screen pixels: one rectangle, or one per tile.
func placements(f fit, size image.Point, area image.Rectangle) []image.Rectangle {
	if size.X <= 0 || size.Y <= 0 || area.Empty() {
		return nil
	}
	centred := func(s float64) []image.Rectangle {
		w, h := int(math.Round(float64(size.X)*s)), int(math.Round(float64(size.Y)*s))
		min := area.Min.Add(image.Pt((area.Dx()-w)/2, (area.Dy()-h)/2))
		return []image.Rectangle{{Min: min, Max: min.Add(image.Pt(w, h))}}
	}
	sx, sy := float64(area.Dx())/float64(size.X), float64(area.Dy())/float64(size.Y)
	switch f {
	case fitContain:
		return centred(math.Min(sx, sy))
	case fitStretch:
		return []image.Rectangle{area}
	case fitCenter:
		return centred(1)
	case fitTile:
		var tiles []image.Rectangle
		for y := area.Min.Y; y < area.Max.Y; y += size.Y {
			for x := area.Min.X; x < area.Max.X; x += size.X {
				tiles = append(tiles, image.Rect(x, y, x+size.X, y+size.Y))
			}
		}
		return tiles
	default:
		return centred(math.Max(sx, sy))
	}
}

// boxBlur averages each pixel with its neighbours within r, across then down; the edges repeat their last pixel.
func boxBlur(img *image.RGBA, r int) {
	b := img.Bounds()
	line := make([][4]int, max(b.Dx(), b.Dy()))
	pass := func(n int, at func(i int) int) {
		for i := range n {
			p := at(i)
			line[i] = [4]int{int(img.Pix[p]), int(img.Pix[p+1]), int(img.Pix[p+2]), int(img.Pix[p+3])}
		}
		for i := range n {
			var sum [4]int
			for k := i - r; k <= i+r; k++ {
				c := line[min(max(k, 0), n-1)]
				for ch := range 4 {
					sum[ch] += c[ch]
				}
			}
			p := at(i)
			for ch := range 4 {
				img.Pix[p+ch] = uint8(sum[ch] / (2*r + 1))
			}
		}
	}
	for y := range b.Dy() {
		pass(b.Dx(), func(x int) int { return y*img.Stride + x*4 })
	}
	for x := range b.Dx() {
		pass(b.Dy(), func(y int) int { return y*img.Stride + x*4 })
	}
}
