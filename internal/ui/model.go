package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/EwanValentine/pdf2context/internal/processor"
)

type workerStatus int

const (
	statusIdle workerStatus = iota
	statusExtracting
	statusOCR
	statusChunking
)

type workerState struct {
	id     int
	status workerStatus
	file   string
	active bool
}

type logEntry struct {
	file    string
	chunks  int
	elapsed time.Duration
	ocrd    bool
	err     error
}

// tickMsg is sent on every UI tick.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Model is the Bubble Tea model for the pdf2context TUI.
type Model struct {
	inputDir     string
	total        int
	workers      int
	completed    int
	failed       int
	ocrd         int
	chunks       int
	workerStates []workerState
	recentLog    []logEntry // keep last 6
	prog         progress.Model
	spin         spinner.Model
	startTime    time.Time
	width        int
	done         bool
	manifestPath string
	combinedPath string
	totalDuration time.Duration
}

// NewModel creates a new TUI model.
func NewModel(inputDir string, total, workers int) Model {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	states := make([]workerState, workers)
	for i := range states {
		states[i] = workerState{id: i}
	}

	return Model{
		inputDir:     inputDir,
		total:        total,
		workers:      workers,
		workerStates: states,
		prog:         prog,
		spin:         sp,
		startTime:    time.Now(),
	}
}

// Init starts the spinner and the first tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, tick())
}

// Update handles incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.prog.Width = progressBarWidth(m.width)
		return m, nil

	case tickMsg:
		return m, tick()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		pm, cmd := m.prog.Update(msg)
		m.prog = pm.(progress.Model)
		return m, cmd

	case processor.MsgWorkerStarted:
		if msg.WorkerID < len(m.workerStates) {
			m.workerStates[msg.WorkerID].active = true
			m.workerStates[msg.WorkerID].file = filepath.Base(msg.Path)
			m.workerStates[msg.WorkerID].status = statusIdle
		}
		return m, nil

	case processor.MsgWorkerExtracting:
		if msg.WorkerID < len(m.workerStates) {
			m.workerStates[msg.WorkerID].status = statusExtracting
		}
		return m, nil

	case processor.MsgWorkerOCR:
		if msg.WorkerID < len(m.workerStates) {
			m.workerStates[msg.WorkerID].status = statusOCR
		}
		return m, nil

	case processor.MsgWorkerChunking:
		if msg.WorkerID < len(m.workerStates) {
			m.workerStates[msg.WorkerID].status = statusChunking
		}
		return m, nil

	case processor.MsgWorkerDone:
		if msg.WorkerID < len(m.workerStates) {
			m.workerStates[msg.WorkerID].active = false
			m.workerStates[msg.WorkerID].status = statusIdle
			m.workerStates[msg.WorkerID].file = ""
		}
		m.completed++
		if msg.Result.Err != nil {
			m.failed++
		} else {
			m.chunks += msg.Result.Chunks
			if msg.Result.OCRd {
				m.ocrd++
			}
		}
		entry := logEntry{
			file:    filepath.Base(msg.Result.Path),
			chunks:  msg.Result.Chunks,
			elapsed: msg.Result.Duration,
			ocrd:    msg.Result.OCRd,
			err:     msg.Result.Err,
		}
		m.recentLog = append([]logEntry{entry}, m.recentLog...)
		if len(m.recentLog) > 6 {
			m.recentLog = m.recentLog[:6]
		}
		var pct float64
		if m.total > 0 {
			pct = float64(m.completed) / float64(m.total)
		}
		return m, m.prog.SetPercent(pct)

	case processor.MsgAllDone:
		m.done = true
		m.manifestPath = msg.ManifestPath
		m.combinedPath = msg.CombinedPath
		m.totalDuration = msg.Duration
		return m, tea.Quit
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.done {
		return m.doneView()
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(0, 1).
			Width(m.width - 4).
			Render(headerStyle.Render("pdf2context")),
	)
	sb.WriteString("\n\n")

	// Directory
	sb.WriteString(fmt.Sprintf("  %s %s\n\n",
		statLabelStyle.Render("Directory:"),
		m.inputDir,
	))

	// Progress bar
	var pct float64
	if m.total > 0 {
		pct = float64(m.completed) / float64(m.total)
	}
	sb.WriteString(fmt.Sprintf("  %s\n", statValueStyle.Render("Overall Progress")))
	barWidth := progressBarWidth(m.width)
	m.prog.Width = barWidth
	sb.WriteString(fmt.Sprintf("  %s  %s  %s\n\n",
		m.prog.View(),
		fmt.Sprintf("%.0f%%", pct*100),
		mutedStyle.Render(fmt.Sprintf("%d / %d files", m.completed, m.total)),
	))

	// Workers panel
	sb.WriteString(m.workersPanel())
	sb.WriteString("\n")

	// Statistics panel
	sb.WriteString(m.statsPanel())
	sb.WriteString("\n")

	// Recent log panel
	sb.WriteString(m.recentPanel())
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) doneView() string {
	var sb strings.Builder
	sb.WriteString("\n")

	succeeded := m.completed - m.failed
	summary := fmt.Sprintf("Complete — %d files, %d chunks", succeeded, m.chunks)
	if m.failed > 0 {
		summary += fmt.Sprintf(", %d failed", m.failed)
	}
	sb.WriteString(fmt.Sprintf("  %s %s\n",
		successStyle.Render("✓"),
		statValueStyle.Render(summary),
	))
	if m.manifestPath != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("Manifest:"),
			m.manifestPath,
		))
	}
	if m.combinedPath != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			statLabelStyle.Render("Combined:"),
			m.combinedPath,
		))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m Model) workersPanel() string {
	innerWidth := m.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var rows []string
	for _, w := range m.workerStates {
		var statusStr, fileStr string
		if !w.active {
			statusStr = mutedStyle.Render("·  idle")
			fileStr = ""
		} else {
			icon := m.spin.View()
			label := workerStatusLabel(w.status)
			statusStr = fmt.Sprintf("%s %s", icon, label)
			fileStr = truncate(w.file, innerWidth-20)
		}
		row := fmt.Sprintf("  %s  %-12s  %s",
			mutedStyle.Render(fmt.Sprintf("%d", w.id+1)),
			statusStr,
			fileStr,
		)
		rows = append(rows, row)
	}

	title := statLabelStyle.Render("─ Workers ")
	titleLine := title + strings.Repeat("─", intMax(0, innerWidth-len("─ Workers ")-2))

	content := "┌" + titleLine + "┐\n"
	for _, r := range rows {
		content += "│" + padRight(r, innerWidth+2) + "│\n"
	}
	content += "└" + strings.Repeat("─", innerWidth+2) + "┘"

	return lipgloss.NewStyle().MarginLeft(2).Render(content)
}

func (m Model) statsPanel() string {
	innerWidth := m.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	elapsed := time.Since(m.startTime)
	speed := 0.0
	etaStr := "–"
	if elapsed.Minutes() > 0 && m.completed > 0 {
		speed = float64(m.completed) / elapsed.Minutes()
		remaining := m.total - m.completed
		if speed > 0 {
			etaSecs := float64(remaining) / speed * 60
			eta := time.Duration(etaSecs) * time.Second
			etaStr = "~" + fmtDuration(eta)
		}
	}

	line1 := fmt.Sprintf("  %s: %s  %s: %s  %s: %s  %s: %s",
		statLabelStyle.Render("Files"), statValueStyle.Render(fmt.Sprintf("%d", m.total)),
		statLabelStyle.Render("Done"), statValueStyle.Render(fmt.Sprintf("%d", m.completed)),
		statLabelStyle.Render("Failed"), statValueStyle.Render(fmt.Sprintf("%d", m.failed)),
		statLabelStyle.Render("OCR'd"), statValueStyle.Render(fmt.Sprintf("%d", m.ocrd)),
	)
	line2 := fmt.Sprintf("  %s: %s  %s: %.1f/min  %s: %s",
		statLabelStyle.Render("Chunks"), statValueStyle.Render(fmt.Sprintf("%d", m.chunks)),
		statLabelStyle.Render("Speed"), speed,
		statLabelStyle.Render("ETA"), etaStr,
	)

	title := statLabelStyle.Render("─ Statistics ")
	titleLine := title + strings.Repeat("─", intMax(0, innerWidth-len("─ Statistics ")-2))

	content := "┌" + titleLine + "┐\n"
	content += "│" + padRight(line1, innerWidth+2) + "│\n"
	content += "│" + padRight(line2, innerWidth+2) + "│\n"
	content += "└" + strings.Repeat("─", innerWidth+2) + "┘"

	return lipgloss.NewStyle().MarginLeft(2).Render(content)
}

func (m Model) recentPanel() string {
	innerWidth := m.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	title := statLabelStyle.Render("─ Recent ")
	titleLine := title + strings.Repeat("─", intMax(0, innerWidth-len("─ Recent ")-2))

	content := "┌" + titleLine + "┐\n"

	if len(m.recentLog) == 0 {
		content += "│" + padRight(mutedStyle.Render("  (no files completed yet)"), innerWidth+2) + "│\n"
	} else {
		for _, e := range m.recentLog {
			var line string
			if e.err != nil {
				errStr := truncate(e.err.Error(), innerWidth-20)
				line = fmt.Sprintf("  %s  %-30s error: %s",
					errorStyle.Render("✗"),
					truncate(e.file, 30),
					errStr,
				)
			} else {
				ocrTag := ""
				if e.ocrd {
					ocrTag = " (OCR)"
				}
				line = fmt.Sprintf("  %s  %-30s %3d chunks  %s%s",
					successStyle.Render("✓"),
					truncate(e.file, 30),
					e.chunks,
					fmtDuration(e.elapsed),
					mutedStyle.Render(ocrTag),
				)
			}
			content += "│" + padRight(line, innerWidth+2) + "│\n"
		}
	}

	content += "└" + strings.Repeat("─", innerWidth+2) + "┘"

	return lipgloss.NewStyle().MarginLeft(2).Render(content)
}

// --- Helpers ---

func workerStatusLabel(s workerStatus) string {
	switch s {
	case statusExtracting:
		return "extracting"
	case statusOCR:
		return "ocr       "
	case statusChunking:
		return "chunking  "
	default:
		return "idle      "
	}
}

func progressBarWidth(width int) int {
	w := width - 30
	if w > 50 {
		w = 50
	}
	if w < 10 {
		w = 10
	}
	return w
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

func padRight(s string, n int) string {
	// Count visible rune width (approximate: ignore ANSI escapes).
	visible := visibleLen(s)
	if visible >= n {
		return s
	}
	return s + strings.Repeat(" ", n-visible)
}

// visibleLen returns an approximate visible length, ignoring ANSI escape codes.
func visibleLen(s string) int {
	inEsc := false
	count := 0
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		count++
	}
	return count
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
