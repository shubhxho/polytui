package ui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"shubhxho/polytui/internal/api"
)

// TestPrepareRealImage hits a live Polymarket event image end to end. Gated on
// NET=1 like the other integration tests.
func TestPrepareRealImage(t *testing.T) {
	if os.Getenv("NET") == "" {
		t.Skip("set NET=1 for live image fetch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	evs, err := api.New().Events(ctx, api.EventQuery{Limit: 10, Order: "volume24hr"})
	if err != nil {
		t.Fatal(err)
	}
	var url string
	for i := range evs {
		if url = eventImageURL(&evs[i]); url != "" {
			break
		}
	}
	if url == "" {
		t.Skip("no event image url found")
	}
	png, err := fetchAndPreparePNG(ctx, url, imgMaxPixels)
	if err != nil {
		t.Fatalf("fetch %s: %v", url, err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		t.Fatalf("prepared image not decodable: %v", err)
	}
	if format != "png" {
		t.Errorf("prepared format = %q, want png", format)
	}
	if cfg.Width > imgMaxPixels || cfg.Height > imgMaxPixels {
		t.Errorf("prepared size %dx%d exceeds %d", cfg.Width, cfg.Height, imgMaxPixels)
	}
	t.Logf("prepared %s -> %s %dx%d (%d bytes)", url, format, cfg.Width, cfg.Height, len(png))
}

// synthPNG builds a w×h test PNG with a simple gradient.
func synthPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestScaleToFit(t *testing.T) {
	src, _, err := image.Decode(bytes.NewReader(synthPNG(t, 600, 300)))
	if err != nil {
		t.Fatal(err)
	}
	dst := scaleToFit(src, 256)
	if got := dst.Bounds().Dx(); got != 256 {
		t.Errorf("width = %d, want 256 (longest edge clamp)", got)
	}
	if got := dst.Bounds().Dy(); got != 128 {
		t.Errorf("height = %d, want 128 (aspect preserved)", got)
	}
	// Never upscale.
	small := scaleToFit(src, 2000)
	if small.Bounds().Dx() != 600 {
		t.Errorf("upscaled to %d, want original 600", small.Bounds().Dx())
	}
}

func TestBuildKittyImage(t *testing.T) {
	ki := buildKittyImage(7, synthPNG(t, 64, 64), imgCols, imgRows)
	if ki.cols != imgCols || ki.rows != imgRows {
		t.Fatalf("dims = %dx%d, want %dx%d", ki.cols, ki.rows, imgCols, imgRows)
	}
	lines := strings.Split(ki.block, "\n")
	if len(lines) != imgRows {
		t.Fatalf("block has %d rows, want %d", len(lines), imgRows)
	}
	// The heavy transmit blob rides on the first line only.
	if !strings.Contains(lines[0], "\x1b_Ga=T,U=1,i=7") {
		t.Errorf("line 0 missing transmit header: %q", lines[0][:min(60, len(lines[0]))])
	}
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "\x1b_G") {
			t.Errorf("line %d unexpectedly carries a transmit escape", i)
		}
	}
	// Every line is exactly imgCols display cells wide (placeholders are width 1,
	// the transmit + colour escapes are zero-width).
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w != imgCols {
			t.Errorf("line %d width = %d, want %d", i, w, imgCols)
		}
		if n := strings.Count(ln, string(placeholderRune)); n != imgCols {
			t.Errorf("line %d has %d placeholder cells, want %d", i, n, imgCols)
		}
	}
	// Image id is encoded in the cell foreground colour (id 7 -> 0;0;7).
	if !strings.Contains(ki.block, "\x1b[38;2;0;0;7m") {
		t.Errorf("foreground does not encode image id 7")
	}
}

// TestDetailViewKeepsGraphics drives the whole detail View (which runs
// zone.Scan) and confirms the Kitty transmit + placeholders survive intact.
func TestDetailViewKeepsGraphics(t *testing.T) {
	m := New()
	m.imgOK = true
	m = drive(m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		eventsMsg{events: sampleEvents()},
	)
	m.cursor = 0
	m.events[0].Image = "http://img"
	nm, _ := m.handleBrowseKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(model)
	// Simulate the thumbnail fetch completing.
	url := eventImageURL(m.detail)
	m = drive(m, imageMsg{url: url, png: synthPNG(t, 64, 64)})

	out := m.View()
	if !strings.Contains(out, "\x1b_Ga=T,U=1") {
		t.Error("View dropped the Kitty transmit escape")
	}
	if !strings.Contains(out, string(placeholderRune)) {
		t.Error("View dropped the image placeholders")
	}
}

func TestDetailHeaderWithThumb(t *testing.T) {
	m := New()
	m.width, m.height = 120, 40
	m.imgOK = true
	ev := api.Event{ID: "x", Title: "Will it rain tomorrow?", Image: "http://img"}
	m.detail = &ev
	m.imgCache["http://img"] = buildKittyImage(3, synthPNG(t, 48, 48), imgCols, imgRows)

	header := m.detailHeader(m.width - 4)
	if !strings.Contains(header, string(placeholderRune)) {
		t.Error("header missing image placeholders")
	}
	if !strings.Contains(header, "Will it rain") {
		t.Error("header missing title text alongside image")
	}
	if h := lipgloss.Height(header); h < imgRows {
		t.Errorf("header height %d shorter than thumbnail (%d)", h, imgRows)
	}

	// Without a cached image the header degrades to plain text.
	m.imgCache = map[string]*kittyImage{}
	plain := m.detailHeader(m.width - 4)
	if strings.Contains(plain, string(placeholderRune)) {
		t.Error("plain header should not contain placeholders")
	}
}
