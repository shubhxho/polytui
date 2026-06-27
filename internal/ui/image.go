package ui

// Kitty terminal graphics support.
//
// Polymarket events carry a square cover image; on a terminal that speaks the
// Kitty graphics protocol (kitty, Ghostty, WezTerm, Konsole) we render it as a
// little thumbnail in the detail header. The integration uses the protocol's
// *Unicode placeholder* mode so the image lives inside the normal text grid:
//
//   1. The image bytes are transmitted once as a "virtual placement" (U=1) bound
//      to an image id — this neither moves the cursor nor draws anything.
//   2. A block of placeholder cells (U+10EEEE, one per cell) references that id
//      via the cell's foreground colour, with row/column combining diacritics
//      telling the terminal which slice of the image each cell shows.
//
// Because the placeholder cells are real width-1 grid cells, Lip Gloss lays them
// out like any other text and Bubble Tea's line-diff renderer only writes the
// heavy transmit blob once (it sits on an otherwise-static header line).

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"shubhxho/polytui/internal/api"
)

const (
	imgCols      = 16  // thumbnail width in terminal cells
	imgRows      = 8   // thumbnail height in terminal cells
	imgMaxPixels = 256 // longest source edge after downscale (keeps transmits small)

	// placeholderRune is U+10EEEE, the Kitty Unicode image-placeholder code point.
	placeholderRune = '\U0010EEEE'
)

// imageHTTP is a dedicated client tuned for thumbnail fetches: keep-alives and a
// fat per-host idle pool so the many same-host S3 images (Polymarket serves them
// all from one bucket) reuse warm TLS connections instead of dialing per request.
var imageHTTP = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   16,
		MaxConnsPerHost:       16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ExpectContinueTimeout: time.Second,
	},
}

// imageSem bounds how many thumbnail fetch+decode jobs run at once, so a fast
// scroll that warms many events doesn't spawn dozens of goroutines or hammer the
// CDN. Acquired inside loadImage's command goroutine.
var imageSem = make(chan struct{}, 6)

// kittyImage is a fully-built thumbnail: the transmit escape prepended to a grid
// of placeholder cells, ready to drop straight into a Lip Gloss layout.
type kittyImage struct {
	block      string
	cols, rows int
}

// kittyEnabled reports whether the host terminal understands the Kitty graphics
// protocol. POLYTUI_IMAGES forces it on/off; otherwise we sniff well-known
// terminals that implement it.
func kittyEnabled() bool {
	switch strings.ToLower(os.Getenv("POLYTUI_IMAGES")) {
	case "1", "on", "true", "yes":
		return true
	case "0", "off", "false", "no":
		return false
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" ||
		os.Getenv("WEZTERM_PANE") != "" {
		return true
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "ghostty", "wezterm":
		return true
	}
	term := os.Getenv("TERM")
	return strings.Contains(term, "kitty") || strings.Contains(term, "ghostty")
}

// eventImageURL picks the best cover art for an event: the full image, then the
// icon, then anything a contained market offers.
func eventImageURL(e *api.Event) string {
	if e == nil {
		return ""
	}
	if e.Image != "" {
		return e.Image
	}
	if e.Icon != "" {
		return e.Icon
	}
	for _, m := range e.Markets {
		if m.Image != "" {
			return m.Image
		}
		if m.Icon != "" {
			return m.Icon
		}
	}
	return ""
}

// fetchAndPreparePNG downloads an image, downscales it to fit imgMaxPixels, and
// re-encodes it as PNG so the Kitty transmit always uses one known format.
func fetchAndPreparePNG(ctx context.Context, url string, maxDim int) ([]byte, error) {
	url = strings.ReplaceAll(url, " ", "%20") // some Polymarket asset keys contain spaces
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")
	req.Header.Set("User-Agent", "polytui/1.0")
	resp, err := imageHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image http %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	src, _, err := image.Decode(bytes.NewReader(raw)) // png/jpeg/gif via blank imports
	if err != nil {
		return nil, err
	}
	small := scaleToFit(src, maxDim)
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, small); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// scaleToFit returns src bilinearly resampled so its longest edge is at most
// maxDim (never upscales). The source is normalised to a zero-origin RGBA once
// (one O(n) pass, also handling JPEG's YCbCr) so the inner loop indexes the
// pixel buffer directly instead of paying an interface call + colour conversion
// per sample — the difference between milliseconds and tens of milliseconds.
func scaleToFit(src image.Image, maxDim int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw < 1 || sh < 1 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	rgba, ok := src.(*image.RGBA)
	if !ok || b.Min != (image.Point{}) {
		conv := image.NewRGBA(image.Rect(0, 0, sw, sh))
		draw.Draw(conv, conv.Bounds(), src, b.Min, draw.Src)
		rgba = conv
	}

	scale := 1.0
	if longest := sw; sh > longest {
		longest = sh
		if longest > maxDim {
			scale = float64(maxDim) / float64(longest)
		}
	} else if longest > maxDim {
		scale = float64(maxDim) / float64(longest)
	}
	if scale >= 1.0 {
		return rgba // already within budget; skip the resample entirely
	}
	dw := max1(int(float64(sw)*scale + 0.5))
	dh := max1(int(float64(sh)*scale + 0.5))

	pix, stride := rgba.Pix, rgba.Stride
	sample := func(x, y, ch int) float64 {
		if x < 0 {
			x = 0
		} else if x >= sw {
			x = sw - 1
		}
		if y < 0 {
			y = 0
		} else if y >= sh {
			y = sh - 1
		}
		return float64(pix[y*stride+x*4+ch])
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		fy := (float64(y)+0.5)/float64(dh)*float64(sh) - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		row := y * dst.Stride
		for x := 0; x < dw; x++ {
			fx := (float64(x)+0.5)/float64(dw)*float64(sw) - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			o := row + x*4
			for ch := 0; ch < 4; ch++ {
				c00 := sample(x0, y0, ch)
				c10 := sample(x0+1, y0, ch)
				c01 := sample(x0, y0+1, ch)
				c11 := sample(x0+1, y0+1, ch)
				dst.Pix[o+ch] = blerp(c00, c10, c01, c11, tx, ty)
			}
		}
	}
	return dst
}

func blerp(c00, c10, c01, c11, tx, ty float64) uint8 {
	top := c00 + (c10-c00)*tx
	bot := c01 + (c11-c01)*tx
	v := top + (bot-top)*ty
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	return uint8(v + 0.5)
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// buildKittyImage assembles the transmit escape plus a cols×rows grid of
// placeholder cells for the given image id.
func buildKittyImage(id uint32, pngData []byte, cols, rows int) *kittyImage {
	lines := placeholderGrid(id, cols, rows)
	lines[0] = transmitSeq(id, pngData, cols, rows) + lines[0]
	return &kittyImage{block: strings.Join(lines, "\n"), cols: cols, rows: rows}
}

// transmitSeq builds a chunked Kitty transmit-and-virtual-place command (a=T,
// U=1) for a PNG (f=100). q=2 suppresses the terminal's reply so it never leaks
// into Bubble Tea's input. The placement is virtual: nothing is drawn and the
// cursor does not move until placeholder cells reference image id.
func transmitSeq(id uint32, pngData []byte, cols, rows int) string {
	b64 := base64.StdEncoding.EncodeToString(pngData)
	const chunk = 4096
	var sb strings.Builder
	first := true
	for len(b64) > 0 {
		n := chunk
		if n > len(b64) {
			n = len(b64)
		}
		part := b64[:n]
		b64 = b64[n:]
		more := 0
		if len(b64) > 0 {
			more = 1
		}
		sb.WriteString("\x1b_G")
		if first {
			fmt.Fprintf(&sb, "a=T,U=1,i=%d,f=100,t=d,c=%d,r=%d,q=2,m=%d", id, cols, rows, more)
			first = false
		} else {
			fmt.Fprintf(&sb, "m=%d", more)
		}
		sb.WriteByte(';')
		sb.WriteString(part)
		sb.WriteString("\x1b\\")
	}
	return sb.String()
}

// placeholderGrid renders rows lines of cols placeholder cells. The image id is
// carried in each cell's 24-bit foreground colour; row/column diacritics tell
// the terminal which slice of the image the cell shows.
func placeholderGrid(id uint32, cols, rows int) []string {
	r, g, b := byte(id>>16), byte(id>>8), byte(id)
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var sb strings.Builder
		fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm", r, g, b)
		for x := 0; x < cols; x++ {
			sb.WriteRune(placeholderRune)
			sb.WriteRune(rowColumnDiacritics[y])
			sb.WriteRune(rowColumnDiacritics[x])
		}
		sb.WriteString("\x1b[39m")
		lines[y] = sb.String()
	}
	return lines
}

// rowColumnDiacritics is the canonical Kitty row/column encoding table (the
// first 64 entries — enough for any thumbnail we draw). Cell (row, col) appends
// rowColumnDiacritics[row] then rowColumnDiacritics[col] after the placeholder.
var rowColumnDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
	0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F, 0x0483, 0x0484,
	0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1,
	0x05A8, 0x05A9, 0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611,
	0x0612, 0x0613, 0x0614, 0x0615, 0x0616, 0x0617, 0x0657, 0x0658,
}
