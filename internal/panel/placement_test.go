package panel

import "testing"

func TestPlaceAtKeepsWhereThePersonLeftIt(t *testing.T) {
	work := Rect{0, 0, 1920, 1040}
	base := Rect{560, 187, 800, 72}
	tests := []struct {
		name   string
		x, y   int
		want   Rect
		wantOK bool
	}{
		{"left where it was dragged", 100, 300, Rect{100, 300, 800, 72}, true},
		{"pulled in from the right edge", 1800, 300, Rect{1120, 300, 800, 72}, true},
		{"pulled up from the bottom edge", 100, 1030, Rect{100, 968, 800, 72}, true},
		{"on another monitor: usual place", 2500, 300, base, false},
		{"above the screen: usual place", 100, -40, base, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := placeAt(base, work, tt.x, tt.y)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("placeAt(%d, %d) = %+v, %v; want %+v, %v", tt.x, tt.y, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestPlaceCentresTheBarNearTheTopOfTheUsableArea(t *testing.T) {
	tests := []struct {
		name      string
		work      Rect
		dpi       uint32
		heightDIP int
		want      Rect
	}{
		{"1080p at 100%", Rect{0, 0, 1920, 1040}, 96, 72, Rect{560, 187, 800, 72}},
		{"1080p laptop at 150%", Rect{0, 0, 1920, 1032}, 144, 72, Rect{360, 186, 1200, 108}},
		{"4K at 200%", Rect{0, 0, 3840, 2080}, 192, 72, Rect{1120, 374, 1600, 144}},
		{"second monitor to the right", Rect{1920, 0, 2560, 1400}, 96, 72, Rect{2800, 252, 800, 72}},
		{"second monitor to the left (negative coordinates)", Rect{-1920, 100, 1920, 1040}, 96, 72, Rect{-1360, 287, 800, 72}},
		{"a result list makes it taller", Rect{0, 0, 1920, 1040}, 96, 520, Rect{560, 187, 800, 520}},
		{"never wider than the screen", Rect{0, 0, 600, 800}, 96, 72, Rect{0, 144, 600, 72}},
		{"never off the bottom", Rect{0, 0, 1920, 400}, 96, 380, Rect{560, 20, 800, 380}},
		{"never taller than the screen", Rect{0, 0, 1920, 300}, 96, 520, Rect{560, 0, 800, 300}},
		{"an unknown dpi counts as 100%", Rect{0, 0, 1920, 1040}, 0, 72, Rect{560, 187, 800, 72}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := place(tt.work, tt.dpi, tt.heightDIP); got != tt.want {
				t.Fatalf("place(%+v, dpi %d, %d) = %+v, want %+v", tt.work, tt.dpi, tt.heightDIP, got, tt.want)
			}
		})
	}
}
