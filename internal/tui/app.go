package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"uptui/internal/ipc"
	"uptui/internal/models"
)

// ── message types ─────────────────────────────────────────────────────────────

type tickMsg time.Time
type dataMsg struct {
	monitors []*models.MonitorStatus
	err      error
}

// ── views ─────────────────────────────────────────────────────────────────────

type view int

const (
	viewDashboard view = iota
	viewDetail
	viewAdd
)

// ── sort / filter keys ────────────────────────────────────────────────────────

const (
	sortByName   = 0
	sortByStatus = 1
	sortByUptime = 2

	filterAll      = 0
	filterDown     = 1
	filterProblems = 2
)

// ── model ─────────────────────────────────────────────────────────────────────

type Model struct {
	client   *ipc.Client
	monitors []*models.MonitorStatus
	cursor   int
	view     view
	err      string
	loading  bool
	width    int
	height   int
	styles   Styles

	// detail
	selected *models.MonitorStatus

	// add/edit form
	addInputs   []textinput.Model
	addFocus    int
	addErr      string
	editMode    bool   // true when form is used for editing
	editOldName string // the original name being edited

	// sort / filter / scroll
	sortKey      int
	filterKey    int
	detailScroll int

	// delete confirmation
	pendingDelete string // non-empty while awaiting y/N confirmation

	// edit confirmation
	pendingEdit *models.Monitor // non-nil while awaiting y/N confirmation for edit
}

func NewModel(client *ipc.Client, theme Theme) Model {
	inputs := make([]textinput.Model, 4)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "My Service"
	inputs[0].CharLimit = 60
	inputs[0].Width = 32

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "http"
	inputs[1].CharLimit = 8
	inputs[1].Width = 10

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "https://example.com"
	inputs[2].CharLimit = 200
	inputs[2].Width = 40

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "60"
	inputs[3].CharLimit = 6
	inputs[3].Width = 8

	return Model{
		client:    client,
		addInputs: inputs,
		loading:   true,
		width:     80,
		height:    24,
		styles:    NewStyles(theme),
	}
}

// ── bubbletea interface ────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchData(m.client), schedTick())
}

func fetchData(c *ipc.Client) tea.Cmd {
	return func() tea.Msg {
		monitors, err := c.List()
		return dataMsg{monitors: monitors, err: err}
	}
}

func schedTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case dataMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.monitors = msg.monitors
			// clamp cursor to visible list
			visible := m.visibleMonitors()
			if len(visible) == 0 {
				m.cursor = 0
			} else if m.cursor >= len(visible) {
				m.cursor = len(visible) - 1
			}
			// keep selected in sync for detail view (match by Name)
			if m.view == viewDetail && m.selected != nil {
				for _, ms := range m.monitors {
					if ms.Monitor.Name == m.selected.Monitor.Name {
						m.selected = ms
						break
					}
				}
			}
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchData(m.client), schedTick())

	case tea.KeyMsg:
		switch m.view {
		case viewDashboard:
			return m.updateDashboard(msg)
		case viewDetail:
			return m.updateDetail(msg)
		case viewAdd:
			return m.updateAdd(msg)
		}
	}

	if m.view == viewAdd {
		return m.forwardToInputs(msg)
	}
	return m, nil
}

// ── dashboard keys ─────────────────────────────────────────────────────────────

func (m Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ── confirmation mode: only y/n/esc are meaningful ───────────────────────
	if m.pendingDelete != "" {
		switch msg.String() {
		case "y", "Y":
			name := m.pendingDelete
			c := m.client
			m.pendingDelete = ""
			return m, func() tea.Msg {
				c.Delete(name)
				monitors, err := c.List()
				return dataMsg{monitors: monitors, err: err}
			}
		default:
			m.pendingDelete = ""
		}
		return m, nil
	}

	visible := m.visibleMonitors()

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(visible)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(visible) {
			m.selected = visible[m.cursor]
			m.detailScroll = 0
			m.view = viewDetail
		}
	case "a":
		m.editMode = false
		m.editOldName = ""
		for i := range m.addInputs {
			m.addInputs[i].SetValue("")
		}
		m.addFocus = 0
		m.addErr = ""
		m.view = viewAdd
		cmd := m.addInputs[0].Focus()
		return m, cmd
	case "e":
		if m.cursor < len(visible) {
			ms := visible[m.cursor]
			m.editMode = true
			m.editOldName = ms.Monitor.Name
			m.addInputs[0].SetValue(ms.Monitor.Name)
			m.addInputs[1].SetValue(string(ms.Monitor.Type))
			m.addInputs[2].SetValue(ms.Monitor.Target)
			m.addInputs[3].SetValue(fmt.Sprintf("%d", ms.Monitor.Interval))
			m.addFocus = 0
			m.addErr = ""
			m.view = viewAdd
			cmd := m.addInputs[0].Focus()
			return m, cmd
		}
	case "d":
		if m.cursor < len(visible) {
			m.pendingDelete = visible[m.cursor].Monitor.Name
		}
	case "p":
		if m.cursor < len(visible) {
			ms := visible[m.cursor]
			c := m.client
			name := ms.Monitor.Name
			active := ms.Monitor.Active
			return m, func() tea.Msg {
				if active {
					c.Pause(name)
				} else {
					c.Resume(name)
				}
				monitors, err := c.List()
				return dataMsg{monitors: monitors, err: err}
			}
		}
	case "s":
		m.sortKey = (m.sortKey + 1) % 3
	case "f":
		m.filterKey = (m.filterKey + 1) % 3
		newVis := m.visibleMonitors()
		if len(newVis) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(newVis) {
			m.cursor = len(newVis) - 1
		}
	case "r":
		return m, fetchData(m.client)
	}
	return m, nil
}

// ── detail keys ────────────────────────────────────────────────────────────────

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.detailScroll = 0
		m.view = viewDashboard
	case "down", "j":
		if m.selected != nil && len(m.selected.History) > 0 {
			pageSize := detailPageSize(m.height)
			total := len(m.selected.History)
			maxScroll := total - pageSize
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.detailScroll++
			if m.detailScroll > maxScroll {
				m.detailScroll = maxScroll
			}
		}
	case "up", "k":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	}
	return m, nil
}

// ── add/edit form keys ─────────────────────────────────────────────────────────

func (m Model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ── edit confirmation mode: only y/n/esc are meaningful ──────────────────
	if m.pendingEdit != nil {
		switch msg.String() {
		case "y", "Y":
			mon := *m.pendingEdit
			m.pendingEdit = nil
			_, err := m.client.Edit(m.editOldName, mon)
			if err != nil {
				m.addErr = err.Error()
				return m, nil
			}
			m.addInputs[m.addFocus].Blur()
			m.editMode = false
			m.editOldName = ""
			m.view = viewDashboard
			return m, fetchData(m.client)
		default:
			m.pendingEdit = nil
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.addInputs[m.addFocus].Blur()
		m.editMode = false
		m.editOldName = ""
		m.view = viewDashboard
		return m, nil
	case "tab", "down":
		m.addInputs[m.addFocus].Blur()
		m.addFocus = (m.addFocus + 1) % len(m.addInputs)
		cmd := m.addInputs[m.addFocus].Focus()
		return m, cmd
	case "shift+tab", "up":
		m.addInputs[m.addFocus].Blur()
		m.addFocus = (m.addFocus - 1 + len(m.addInputs)) % len(m.addInputs)
		cmd := m.addInputs[m.addFocus].Focus()
		return m, cmd
	case "enter":
		if m.addFocus == len(m.addInputs)-1 {
			return m.submitAdd()
		}
		m.addInputs[m.addFocus].Blur()
		m.addFocus = (m.addFocus + 1) % len(m.addInputs)
		cmd := m.addInputs[m.addFocus].Focus()
		return m, cmd
	}
	return m.forwardToInputs(msg)
}

func (m Model) forwardToInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for i := range m.addInputs {
		var cmd tea.Cmd
		m.addInputs[i], cmd = m.addInputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) submitAdd() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.addInputs[0].Value())
	monType := strings.ToLower(strings.TrimSpace(m.addInputs[1].Value()))
	target := strings.TrimSpace(m.addInputs[2].Value())
	intervalStr := strings.TrimSpace(m.addInputs[3].Value())

	if name == "" {
		m.addErr = "name is required"
		return m, nil
	}
	if monType == "" {
		monType = "http"
	}
	if monType != "http" && monType != "tcp" {
		m.addErr = "type must be http or tcp"
		return m, nil
	}
	if target == "" {
		m.addErr = "target is required"
		return m, nil
	}

	interval := 60
	if intervalStr != "" {
		fmt.Sscanf(intervalStr, "%d", &interval)
		if interval < 10 {
			interval = 10
		}
	}

	mon := models.Monitor{
		Name:     name,
		Type:     models.MonitorType(monType),
		Target:   target,
		Interval: interval,
		Timeout:  30,
		Active:   true,
	}

	if m.editMode {
		m.pendingEdit = &mon
		return m, nil
	}

	_, err := m.client.Add(mon)
	if err != nil {
		m.addErr = err.Error()
		return m, nil
	}

	m.addInputs[m.addFocus].Blur()
	m.view = viewDashboard
	return m, fetchData(m.client)
}

// ── View ───────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.view {
	case viewDashboard:
		return m.dashboardView()
	case viewDetail:
		return m.detailView()
	case viewAdd:
		return m.addView()
	}
	return ""
}

// ── dashboard view ─────────────────────────────────────────────────────────────

func (m Model) dashboardView() string {
	var sb strings.Builder

	// ── header ────────────────────────────────────────────────────────────────
	upCount, downCount, pendCount := 0, 0, 0
	for _, ms := range m.monitors {
		switch ms.Status {
		case models.StatusUp:
			upCount++
		case models.StatusDown:
			downCount++
		default:
			pendCount++
		}
	}

	header := m.styles.Title.Render("uptui")
	summary := ""
	if len(m.monitors) > 0 {
		parts := []string{}
		if upCount > 0 {
			parts = append(parts, m.styles.Up.Render(fmt.Sprintf("● %d up", upCount)))
		}
		if downCount > 0 {
			parts = append(parts, m.styles.Down.Render(fmt.Sprintf("● %d down", downCount)))
		}
		if pendCount > 0 {
			parts = append(parts, m.styles.Pending.Render(fmt.Sprintf("● %d pending", pendCount)))
		}
		summary = strings.Join(parts, "  ")
	}

	headerLine := header + "  " + summary
	sb.WriteString(headerLine + "\n")
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")

	// ── column widths ────────────────────────────────────────────────────────
	// Extended uptime (7d/30d) shown when terminal is wide enough
	showExtUptime := m.width >= 100
	nameW := m.width - 62
	if showExtUptime {
		nameW = m.width - 80 // extra 18 chars for 7d + 30d columns
	}
	if nameW < 12 {
		nameW = 12
	}
	if nameW > 40 {
		nameW = 40
	}

	// ── column headers ───────────────────────────────────────────────────────
	hStatus := padR(m.styles.Header.Render("STATUS"), 9)
	hName := padR(m.styles.Header.Render("NAME"), nameW)
	hType := padR(m.styles.Header.Render("TYPE"), 5)
	hLat := padR(m.styles.Header.Render("LATENCY"), 10)
	hUp := padR(m.styles.Header.Render("UPTIME"), 8)
	hHist := m.styles.Header.Render("HISTORY")
	if showExtUptime {
		h7d := padR(m.styles.Header.Render("7D"), 7)
		h30d := padR(m.styles.Header.Render("30D"), 7)
		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s  %s  %s  %s\n",
			hStatus, hName, hType, hLat, hUp, h7d, h30d, hHist))
	} else {
		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s  %s\n",
			hStatus, hName, hType, hLat, hUp, hHist))
	}
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")

	// ── rows ─────────────────────────────────────────────────────────────────
	visible := m.visibleMonitors()
	if m.loading {
		sb.WriteString("\n  " + m.styles.Pending.Render("Connecting to daemon...") + "\n")
	} else if m.err != "" {
		sb.WriteString("\n  " + m.styles.Error.Render("Error: "+m.err) + "\n")
		sb.WriteString("  " + m.styles.Muted.Render("Make sure the daemon is running: uptui daemon") + "\n")
	} else if len(m.monitors) == 0 {
		sb.WriteString("\n  " + m.styles.Muted.Render("No monitors configured.") + "\n")
		sb.WriteString("  " + m.styles.Muted.Render("Press ") + m.styles.KeyHint.Render("a") + m.styles.Muted.Render(" to add your first monitor.") + "\n")
	} else if len(visible) == 0 {
		sb.WriteString("\n  " + m.styles.Muted.Render("No monitors match the current filter.") + "\n")
	} else {
		for i, ms := range visible {
			row := m.renderRow(ms, i == m.cursor, nameW, showExtUptime)
			sb.WriteString(row + "\n")
		}
	}

	// ── footer ───────────────────────────────────────────────────────────────
	sortNames := []string{"name", "status", "uptime"}
	filterNames := []string{"all", "down", "problems"}
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")
	var footer string
	if m.pendingDelete != "" {
		footer = m.styles.Error.Render(" Delete ") +
			m.styles.Bold.Render(truncate(m.pendingDelete, 40)) +
			m.styles.Error.Render("?  ") +
			m.styles.KeyHint.Render("y") + m.styles.Muted.Render(" confirm  ") +
			m.styles.KeyHint.Render("any key") + m.styles.Muted.Render(" cancel")
	} else {
		footer = m.styles.Muted.Render(" ") +
			m.styles.KeyHint.Render("a") + m.styles.Muted.Render("dd  ") +
			m.styles.KeyHint.Render("e") + m.styles.Muted.Render("dit  ") +
			m.styles.KeyHint.Render("d") + m.styles.Muted.Render("elete  ") +
			m.styles.KeyHint.Render("p") + m.styles.Muted.Render("ause  ") +
			m.styles.KeyHint.Render("s") + m.styles.Muted.Render(":"+sortNames[m.sortKey]+"  ") +
			m.styles.KeyHint.Render("f") + m.styles.Muted.Render(":"+filterNames[m.filterKey]+"  ") +
			m.styles.KeyHint.Render("↑↓") + m.styles.Muted.Render(" nav  ") +
			m.styles.KeyHint.Render("↵") + m.styles.Muted.Render(" detail  ") +
			m.styles.KeyHint.Render("r") + m.styles.Muted.Render(" refresh  ") +
			m.styles.KeyHint.Render("q") + m.styles.Muted.Render("uit")
	}
	sb.WriteString(footer)

	return sb.String()
}

func (m Model) renderRow(ms *models.MonitorStatus, selected bool, nameW int, showExtUptime bool) string {
	cursor := "  "
	if selected {
		cursor = m.styles.Cursor.Render("▶") + " "
	}

	st := m.styles.StatusStyle(string(ms.Status))
	dot := st.Render("●")
	statusText := padR(st.Render(strings.ToUpper(string(ms.Status))), 7)

	name := truncate(ms.Monitor.Name, nameW)
	name = padR(name, nameW)

	monType := padR(string(ms.Monitor.Type), 5)

	var latency string
	if ms.Status == models.StatusDown || ms.Status == models.StatusPaused || ms.Latency == 0 {
		latency = padR(m.styles.Muted.Render("  -"), 10)
	} else {
		latency = padR(fmt.Sprintf("%d ms", ms.Latency), 10)
	}

	isPending := ms.Status == models.StatusPending
	uptime := m.renderUptimePct(ms.Uptime24h, isPending, 8)

	hist := m.sparklineStatus(ms.History)

	if showExtUptime {
		uptime7d := m.renderUptimePct(ms.Uptime7d, isPending, 7)
		uptime30d := m.renderUptimePct(ms.Uptime30d, isPending, 7)
		return fmt.Sprintf("%s%s %s  %s  %s  %s  %s  %s  %s  %s",
			cursor, dot, statusText, name, monType, latency, uptime, uptime7d, uptime30d, hist)
	}
	return fmt.Sprintf("%s%s %s  %s  %s  %s  %s  %s",
		cursor, dot, statusText, name, monType, latency, uptime, hist)
}

// ── detail view ────────────────────────────────────────────────────────────────

func (m Model) detailView() string {
	if m.selected == nil {
		return ""
	}
	ms := m.selected

	var sb strings.Builder

	// header
	back := m.styles.Muted.Render("← ")
	name := m.styles.Bold.Render(ms.Monitor.Name)
	st := m.styles.StatusStyle(string(ms.Status))
	statusBadge := st.Render("● " + strings.ToUpper(string(ms.Status)))
	latBadge := ""
	if ms.Latency > 0 {
		latBadge = "  " + m.styles.Muted.Render(fmt.Sprintf("%d ms", ms.Latency))
	}
	sb.WriteString(back + name + "\n")
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")

	// info line
	target := m.styles.Muted.Render("Target: ") + ms.Monitor.Target
	interval := m.styles.Muted.Render("  Interval: ") + fmt.Sprintf("%ds", ms.Monitor.Interval)
	uptimePct := m.styles.Up.Render(fmt.Sprintf("%.2f%%", ms.Uptime24h))
	if ms.Uptime24h < 90 {
		uptimePct = m.styles.Down.Render(fmt.Sprintf("%.2f%%", ms.Uptime24h))
	}
	uptime7dPct := m.renderUptimePct(ms.Uptime7d, ms.Status == models.StatusPending, 0)
	uptime30dPct := m.renderUptimePct(ms.Uptime30d, ms.Status == models.StatusPending, 0)
	uptime := m.styles.Muted.Render("  Uptime 24h: ") + uptimePct +
		m.styles.Muted.Render("  7d: ") + uptime7dPct +
		m.styles.Muted.Render("  30d: ") + uptime30dPct
	sb.WriteString("  " + statusBadge + latBadge + "  " + target + interval + uptime + "\n")

	lastCheck := ""
	if !ms.LastCheck.IsZero() {
		ago := time.Since(ms.LastCheck).Round(time.Second)
		lastCheck = "  " + m.styles.Muted.Render("Last check: ") + humanDuration(ago) + " ago"
	}
	if lastCheck != "" {
		sb.WriteString(lastCheck + "\n")
	}
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")

	// latency chart
	chartW := m.width - 10
	if chartW < 20 {
		chartW = 20
	}
	if chartW > 120 {
		chartW = 120
	}

	if len(ms.History) == 0 {
		sb.WriteString("\n  " + m.styles.Muted.Render("No check history yet.") + "\n")
	} else {
		sb.WriteString("\n  " + m.styles.Header.Render("Response time") + "\n")
		sb.WriteString("  " + m.latencySparkline(ms.History, chartW) + "\n")
		sb.WriteString(m.latencyStats(ms.History) + "\n")
		sb.WriteString("\n")

		// scrollable check history
		sb.WriteString("  " + m.styles.Header.Render("Check history") + "\n")
		total := len(ms.History)
		pageSize := detailPageSize(m.height)
		maxScroll := total - pageSize
		if maxScroll < 0 {
			maxScroll = 0
		}
		scroll := m.detailScroll
		if scroll > maxScroll {
			scroll = maxScroll
		}
		end := total - scroll
		start := end - pageSize
		if start < 0 {
			start = 0
		}
		if scroll > 0 {
			sb.WriteString(fmt.Sprintf("  %s\n",
				m.styles.Muted.Render(fmt.Sprintf("▲ %d newer  k/↑", scroll))))
		}
		for i := end - 1; i >= start; i-- {
			r := ms.History[i]
			rst := m.styles.StatusStyle(string(r.Status))
			dot := rst.Render("●")
			ts := m.styles.Muted.Render(r.Timestamp.Format("2006-01-02 15:04:05"))
			var lat string
			if r.Latency > 0 {
				lat = fmt.Sprintf("  %4d ms", r.Latency)
			} else {
				lat = m.styles.Muted.Render("       -")
			}
			msg := ""
			if r.Message != "" {
				msg = "  " + m.styles.Muted.Render(r.Message)
			}
			sb.WriteString(fmt.Sprintf("  %s  %s  %s%s\n", dot, ts, lat, msg))
		}
		if start > 0 {
			sb.WriteString(fmt.Sprintf("  %s\n",
				m.styles.Muted.Render(fmt.Sprintf("▼ %d older  j/↓", start))))
		}
	}

	sb.WriteString("\n" + m.styles.Border.Render(strings.Repeat("─", m.width)) + "\n")
	sb.WriteString(m.styles.KeyHint.Render("esc") + m.styles.Muted.Render(" back  ") +
		m.styles.KeyHint.Render("j/k") + m.styles.Muted.Render(" scroll  ") +
		m.styles.KeyHint.Render("q") + m.styles.Muted.Render("uit"))

	return sb.String()
}

// ── add/edit view ──────────────────────────────────────────────────────────────

func (m Model) addView() string {
	var sb strings.Builder

	title := "Add Monitor"
	if m.editMode {
		title = "Edit Monitor"
	}
	sb.WriteString(m.styles.Title.Render(title) + "\n")
	sb.WriteString(m.styles.Border.Render(strings.Repeat("─", 50)) + "\n\n")

	labels := []string{
		"Name      ",
		"Type      ",
		"Target    ",
		"Interval  ",
	}
	hints := []string{
		"",
		m.styles.Muted.Render("  (http or tcp)"),
		"",
		m.styles.Muted.Render("  seconds, min 10"),
	}

	for i, input := range m.addInputs {
		label := m.styles.Label.Render(labels[i])
		focused := ""
		if m.addFocus == i {
			focused = m.styles.Cursor.Render("▶") + " "
		} else {
			focused = "  "
		}
		sb.WriteString(fmt.Sprintf("%s%s %s%s\n\n", focused, label, input.View(), hints[i]))
	}

	if m.addErr != "" {
		sb.WriteString(m.styles.Error.Render("  ✗ "+m.addErr) + "\n\n")
	}

	if m.pendingEdit != nil {
		sb.WriteString(m.styles.Error.Render("  Save changes to ") +
			m.styles.Bold.Render(truncate(m.editOldName, 40)) +
			m.styles.Error.Render("?  ") +
			m.styles.KeyHint.Render("y") + m.styles.Muted.Render(" confirm  ") +
			m.styles.KeyHint.Render("any key") + m.styles.Muted.Render(" cancel"))
	} else {
		sb.WriteString(m.styles.Muted.Render("  ") +
			m.styles.KeyHint.Render("tab") + m.styles.Muted.Render("/") +
			m.styles.KeyHint.Render("↑↓") + m.styles.Muted.Render(" navigate  ") +
			m.styles.KeyHint.Render("enter") + m.styles.Muted.Render(" next/submit  ") +
			m.styles.KeyHint.Render("esc") + m.styles.Muted.Render(" cancel"))
	}

	return sb.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// sparklineStatus renders last 24 results as colored status blocks.
func (m Model) sparklineStatus(history []models.Result) string {
	n := 24
	start := 0
	if len(history) > n {
		start = len(history) - n
	}
	results := history[start:]

	var sb strings.Builder
	for i := 0; i < n; i++ {
		if i >= len(results) {
			sb.WriteString(m.styles.Muted.Render("·"))
			continue
		}
		r := results[i]
		switch r.Status {
		case models.StatusUp:
			sb.WriteString(m.styles.Up.Render("▓"))
		case models.StatusDown:
			sb.WriteString(m.styles.Down.Render("▓"))
		case models.StatusPaused:
			sb.WriteString(m.styles.Paused.Render("─"))
		default:
			sb.WriteString(m.styles.Muted.Render("░"))
		}
	}
	return sb.String()
}

// latencySparkline renders a wider latency chart for the detail view.
func (m Model) latencySparkline(history []models.Result, width int) string {
	n := width
	start := 0
	if len(history) > n {
		start = len(history) - n
	}
	results := history[start:]

	// find max latency among up results
	maxLat := 1
	for _, r := range results {
		if r.Status == models.StatusUp && r.Latency > maxLat {
			maxLat = r.Latency
		}
	}

	var sb strings.Builder
	for _, r := range results {
		switch r.Status {
		case models.StatusDown:
			sb.WriteString(m.styles.Down.Render("▁"))
		case models.StatusPaused:
			sb.WriteString(m.styles.Paused.Render("─"))
		case models.StatusUp:
			idx := int(math.Round(float64(r.Latency) / float64(maxLat) * 7))
			if idx > 7 {
				idx = 7
			}
			if idx < 0 {
				idx = 0
			}
			sb.WriteString(m.styles.Up.Render(string(sparkBlocks[idx])))
		default:
			sb.WriteString(m.styles.Muted.Render("░"))
		}
	}
	return sb.String()
}

func (m Model) latencyStats(history []models.Result) string {
	var latencies []int
	for _, r := range history {
		if r.Status == models.StatusUp && r.Latency > 0 {
			latencies = append(latencies, r.Latency)
		}
	}
	if len(latencies) == 0 {
		return ""
	}
	min, max, sum := latencies[0], latencies[0], 0
	for _, l := range latencies {
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
		sum += l
	}
	avg := sum / len(latencies)
	return fmt.Sprintf("  %s%d ms  %s%d ms  %s%d ms",
		m.styles.Muted.Render("min: "), min,
		m.styles.Muted.Render("avg: "), avg,
		m.styles.Muted.Render("max: "), max)
}

// visibleMonitors returns the filtered and sorted slice of monitors for display.
func (m Model) visibleMonitors() []*models.MonitorStatus {
	out := make([]*models.MonitorStatus, 0, len(m.monitors))
	for _, ms := range m.monitors {
		switch m.filterKey {
		case filterDown:
			if ms.Status != models.StatusDown {
				continue
			}
		case filterProblems:
			if ms.Status == models.StatusUp || ms.Status == models.StatusPaused {
				continue
			}
		}
		out = append(out, ms)
	}
	sort.SliceStable(out, func(i, j int) bool {
		switch m.sortKey {
		case sortByStatus:
			oi, oj := statusOrder(out[i].Status), statusOrder(out[j].Status)
			if oi != oj {
				return oi < oj
			}
			return out[i].Monitor.Name < out[j].Monitor.Name
		case sortByUptime:
			if out[i].Uptime24h != out[j].Uptime24h {
				return out[i].Uptime24h < out[j].Uptime24h
			}
			return out[i].Monitor.Name < out[j].Monitor.Name
		default: // sortByName
			return out[i].Monitor.Name < out[j].Monitor.Name
		}
	})
	return out
}

// statusOrder maps Status to a sort order: down first, up last.
func statusOrder(s models.Status) int {
	switch s {
	case models.StatusDown:
		return 0
	case models.StatusPending:
		return 1
	case models.StatusPaused:
		return 2
	default:
		return 3
	}
}

// renderUptimePct renders a padded uptime percentage. width=0 means no padding.
func (m Model) renderUptimePct(pct float64, isPending bool, width int) string {
	if isPending {
		s := m.styles.Muted.Render("  -")
		if width > 0 {
			return padR(s, width)
		}
		return s
	}
	var u lipgloss.Style
	switch {
	case pct >= 99:
		u = m.styles.Up
	case pct >= 90:
		u = m.styles.Pending
	default:
		u = m.styles.Down
	}
	s := u.Render(fmt.Sprintf("%.1f%%", pct))
	if width > 0 {
		return padR(s, width)
	}
	return s
}

// detailPageSize returns the number of history rows to show in the detail view.
func detailPageSize(height int) int {
	// Approximate overhead: title, borders, info, lastCheck, chart section, history header = ~13 lines
	n := height - 13
	if n < 5 {
		n = 5
	}
	return n
}

// padR right-pads s to width using visible rune count.
func padR(s string, width int) string {
	vis := lipgloss.Width(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// truncate cuts s to maxRunes runes.
func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if maxRunes > 3 {
		return string(runes[:maxRunes-1]) + "…"
	}
	return string(runes[:maxRunes])
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
