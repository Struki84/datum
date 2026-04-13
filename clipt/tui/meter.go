package tui

import (
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type vuTickMsg struct{}

func vuTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(_ time.Time) tea.Msg {
		return vuTickMsg{}
	})
}

const (
	zoneOrange = 0.70
	zoneRed    = 0.90
)

const (
	peakHoldDuration = 1400 * time.Millisecond
	peakDecayRate    = 0.035
)

const blockFull = "█"

var (
	vuWhite  = lipgloss.NewStyle().Background(lipgloss.Color("#11111B")).Foreground(lipgloss.Color("#FFF"))
	vuOrange = lipgloss.NewStyle().Background(lipgloss.Color("#11111B")).Foreground(lipgloss.Color("#FFF"))
	vuRed    = lipgloss.NewStyle().Background(lipgloss.Color("#11111B")).Foreground(lipgloss.Color("#F38BA8"))
	vuEmpty  = lipgloss.NewStyle().Background(lipgloss.Color("#11111B"))
)

type VUMeter struct {
	Width   int
	LevelCh <-chan float64

	level  float64
	peak   float64
	peakAt time.Time
	active bool

	atomicLevel uint64
}

func NewVUMeter(width int, levelCh <-chan float64) *VUMeter {
	return &VUMeter{
		Width:   width,
		LevelCh: levelCh,
	}
}

func (v *VUMeter) Activate() tea.Cmd {
	if v.active {
		return nil
	}
	v.active = true
	v.level = 0
	v.peak = 0

	ch := v.LevelCh
	go func() {
		for val := range ch {
			bits := math.Float64bits(val)
			atomic.StoreUint64((*uint64)(unsafe.Pointer(&v.atomicLevel)), bits)
		}
	}()

	return vuTick()
}

func (v *VUMeter) Deactivate() {
	v.active = false
	v.level = 0
	v.peak = 0
}

func (v *VUMeter) Update(msg tea.Msg) tea.Cmd {
	if !v.active {
		return nil
	}
	if _, ok := msg.(vuTickMsg); !ok {
		return nil
	}

	bits := atomic.LoadUint64((*uint64)(unsafe.Pointer(&v.atomicLevel)))
	target := math.Float64frombits(bits)
	target = math.Max(0, math.Min(1, target))

	if target > v.level {
		v.level += (target - v.level) * 0.6
	} else {
		v.level += (target - v.level) * 0.15
	}

	if v.level >= v.peak {
		v.peak = v.level
		v.peakAt = time.Now()
	} else if time.Since(v.peakAt) > peakHoldDuration {
		v.peak -= peakDecayRate
		if v.peak < 0 {
			v.peak = 0
		}
	}

	return vuTick()
}

func (v *VUMeter) View() string {
	if v.Width < 4 {
		return strings.Repeat(" ", v.Width)
	}

	halfW := v.Width / 2
	left := v.renderHalf(halfW, true)
	right := v.renderHalf(halfW, false)

	return left + right
}

func (v *VUMeter) renderHalf(width int, mirrored bool) string {
	filled := int(math.Round(v.level * float64(width)))
	if filled > width {
		filled = width
	}

	cols := make([]string, width)
	for i := 0; i < width; i++ {
		fraction := float64(i+1) / float64(width)
		active := i < filled

		var ch string
		switch {
		case !active:
			ch = vuEmpty.Render(" ")
		case fraction <= zoneOrange:
			ch = vuWhite.Render(blockFull)
		case fraction <= zoneRed:
			ch = vuOrange.Render(blockFull)
		default:
			ch = vuRed.Render(blockFull)
		}
		cols[i] = ch
	}

	if mirrored {
		for l, r := 0, width-1; l < r; l, r = l+1, r-1 {
			cols[l], cols[r] = cols[r], cols[l]
		}
	}

	return strings.Join(cols, "")
}
