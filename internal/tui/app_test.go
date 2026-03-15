package tui

// White-box tests: same package gives access to unexported types and functions.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"uptui/internal/ipc"
	"uptui/internal/models"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestModel() Model {
	return NewModel(ipc.NewClient("127.0.0.1:29374"), DefaultTheme())
}

func mustModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	got, ok := m.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", m)
	}
	return got
}

func key(k tea.KeyType) tea.KeyMsg  { return tea.KeyMsg{Type: k} }
func rune_(r rune) tea.KeyMsg       { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

var monitors3 = []*models.MonitorStatus{
	{Monitor: models.Monitor{Name: "alpha"}, Status: models.StatusUp},
	{Monitor: models.Monitor{Name: "beta"}, Status: models.StatusDown},
	{Monitor: models.Monitor{Name: "gamma"}, Status: models.StatusPending},
}

// ── NewModel ──────────────────────────────────────────────────────────────────

func TestNewModelDefaults(t *testing.T) {
	m := newTestModel()
	if !m.loading {
		t.Error("loading should be true initially")
	}
	if m.width != 80 {
		t.Errorf("width = %d, want 80", m.width)
	}
	if m.height != 24 {
		t.Errorf("height = %d, want 24", m.height)
	}
	if m.view != viewDashboard {
		t.Errorf("view = %v, want dashboard", m.view)
	}
	if len(m.addInputs) != 5 {
		t.Errorf("addInputs len = %d, want 5", len(m.addInputs))
	}
}

// ── dataMsg ───────────────────────────────────────────────────────────────────

func TestDataMsgSetsMonitors(t *testing.T) {
	m := newTestModel()
	m2, _ := m.Update(dataMsg{monitors: monitors3})
	got := mustModel(t, m2)

	if got.loading {
		t.Error("loading should be false after data")
	}
	if got.err != "" {
		t.Errorf("err = %q, want empty", got.err)
	}
	if len(got.monitors) != 3 {
		t.Errorf("len(monitors) = %d, want 3", len(got.monitors))
	}
}

func TestDataMsgError(t *testing.T) {
	m := newTestModel()
	m2, _ := m.Update(dataMsg{err: fmt.Errorf("connection refused")})
	got := mustModel(t, m2)

	if got.loading {
		t.Error("loading should be false after error data")
	}
	if got.err == "" {
		t.Error("expected err to be set")
	}
}

func TestDataMsgClampsCursor(t *testing.T) {
	m := newTestModel()
	m.cursor = 5
	m.monitors = monitors3

	// Receive only 2 monitors — cursor must clamp to 1
	two := monitors3[:2]
	m2, _ := m.Update(dataMsg{monitors: two})
	got := mustModel(t, m2)

	if got.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", got.cursor)
	}
}

func TestDataMsgEmptyClampsCursorToZero(t *testing.T) {
	m := newTestModel()
	m.cursor = 2

	m2, _ := m.Update(dataMsg{monitors: nil})
	got := mustModel(t, m2)

	if got.cursor != 0 {
		t.Errorf("cursor = %d, want 0 for empty list", got.cursor)
	}
}

// ── WindowSizeMsg ─────────────────────────────────────────────────────────────

func TestWindowSizeMsg(t *testing.T) {
	m := newTestModel()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	got := mustModel(t, m2)

	if got.width != 140 || got.height != 40 {
		t.Errorf("size = %dx%d, want 140x40", got.width, got.height)
	}
}

// ── tickMsg ───────────────────────────────────────────────────────────────────

func TestTickProducesCmd(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("tick should return a cmd to re-fetch and re-tick")
	}
}

// ── dashboard navigation ──────────────────────────────────────────────────────

func TestCursorDown(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3

	m2, _ := m.Update(key(tea.KeyDown))
	got := mustModel(t, m2)
	if got.cursor != 1 {
		t.Errorf("cursor = %d, want 1", got.cursor)
	}
}

func TestCursorDownAlias(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3

	m2, _ := m.Update(rune_('j'))
	got := mustModel(t, m2)
	if got.cursor != 1 {
		t.Errorf("'j': cursor = %d, want 1", got.cursor)
	}
}

func TestCursorDownAtEnd(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 2

	m2, _ := m.Update(key(tea.KeyDown))
	got := mustModel(t, m2)
	if got.cursor != 2 {
		t.Errorf("down at end: cursor = %d, want 2 (unchanged)", got.cursor)
	}
}

func TestCursorUp(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 2

	m2, _ := m.Update(key(tea.KeyUp))
	got := mustModel(t, m2)
	if got.cursor != 1 {
		t.Errorf("cursor = %d, want 1", got.cursor)
	}
}

func TestCursorUpAlias(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 1

	m2, _ := m.Update(rune_('k'))
	got := mustModel(t, m2)
	if got.cursor != 0 {
		t.Errorf("'k': cursor = %d, want 0", got.cursor)
	}
}

func TestCursorUpAtTop(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 0

	m2, _ := m.Update(key(tea.KeyUp))
	got := mustModel(t, m2)
	if got.cursor != 0 {
		t.Errorf("up at top: cursor = %d, want 0 (unchanged)", got.cursor)
	}
}

// ── view switching ────────────────────────────────────────────────────────────

func TestEnterOpensDetail(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 1

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.view != viewDetail {
		t.Errorf("view = %v, want detail", got.view)
	}
	if got.selected == nil {
		t.Fatal("selected is nil")
	}
	if got.selected.Monitor.Name != "beta" {
		t.Errorf("selected.Name = %q, want beta", got.selected.Monitor.Name)
	}
}

func TestEnterNoopOnEmpty(t *testing.T) {
	m := newTestModel()
	// No monitors
	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.view != viewDashboard {
		t.Errorf("enter on empty: view = %v, want dashboard", got.view)
	}
}

func TestAOpensAdd(t *testing.T) {
	m := newTestModel()
	m2, _ := m.Update(rune_('a'))
	got := mustModel(t, m2)

	if got.view != viewAdd {
		t.Errorf("view = %v, want add", got.view)
	}
	// Inputs should be cleared
	for i, inp := range got.addInputs {
		if inp.Value() != "" {
			t.Errorf("addInputs[%d].Value() = %q, want empty", i, inp.Value())
		}
	}
	if got.editMode {
		t.Error("editMode should be false after 'a'")
	}
}

func TestEOpensEditPreFilled(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 0 // alpha

	m2, _ := m.Update(rune_('e'))
	got := mustModel(t, m2)

	if got.view != viewAdd {
		t.Errorf("view = %v, want add (edit mode)", got.view)
	}
	if !got.editMode {
		t.Error("editMode should be true after 'e'")
	}
	if got.editOldName != "alpha" {
		t.Errorf("editOldName = %q, want alpha", got.editOldName)
	}
	if got.addInputs[0].Value() != "alpha" {
		t.Errorf("name field = %q, want alpha", got.addInputs[0].Value())
	}
}

func TestDPrimesPendingDelete(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 1 // beta

	m2, _ := m.Update(rune_('d'))
	got := mustModel(t, m2)

	if got.pendingDelete != "beta" {
		t.Errorf("pendingDelete = %q, want beta", got.pendingDelete)
	}
	if got.view != viewDashboard {
		t.Error("should stay on dashboard while confirming")
	}
}

func TestDConfirmYDeletes(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.pendingDelete = "beta"

	_, cmd := m.Update(rune_('y'))
	if cmd == nil {
		t.Error("confirming delete should return a cmd")
	}
	// pendingDelete should be cleared
	m2, _ := m.Update(rune_('y'))
	got := mustModel(t, m2)
	if got.pendingDelete != "" {
		t.Errorf("pendingDelete = %q, want empty after confirm", got.pendingDelete)
	}
}

func TestDCancelOnOtherKey(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.pendingDelete = "beta"

	m2, cmd := m.Update(rune_('n'))
	got := mustModel(t, m2)

	if got.pendingDelete != "" {
		t.Errorf("pendingDelete = %q, want empty after cancel", got.pendingDelete)
	}
	if cmd != nil {
		t.Error("cancelling delete should return no cmd")
	}
}

func TestDCancelOnEsc(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.pendingDelete = "alpha"

	m2, _ := m.Update(key(tea.KeyEsc))
	got := mustModel(t, m2)

	if got.pendingDelete != "" {
		t.Errorf("pendingDelete = %q, want empty after esc", got.pendingDelete)
	}
}

func TestDNoopOnEmpty(t *testing.T) {
	m := newTestModel()
	// No monitors — d should not set pendingDelete
	m2, _ := m.Update(rune_('d'))
	got := mustModel(t, m2)

	if got.pendingDelete != "" {
		t.Errorf("d on empty: pendingDelete = %q, want empty", got.pendingDelete)
	}
}

func TestENoopOnEmpty(t *testing.T) {
	m := newTestModel()
	// No monitors
	m2, _ := m.Update(rune_('e'))
	got := mustModel(t, m2)

	if got.view != viewDashboard {
		t.Errorf("e on empty: view = %v, want dashboard", got.view)
	}
}

func TestEscFromDetail(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}}

	m2, _ := m.Update(key(tea.KeyEsc))
	got := mustModel(t, m2)

	if got.view != viewDashboard {
		t.Errorf("esc from detail: view = %v, want dashboard", got.view)
	}
}

func TestEscFromAdd(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd

	m2, _ := m.Update(key(tea.KeyEsc))
	got := mustModel(t, m2)

	if got.view != viewDashboard {
		t.Errorf("esc from add: view = %v, want dashboard", got.view)
	}
	if got.editMode {
		t.Error("editMode should be cleared after esc")
	}
}

func TestDetailSelectedKeepsUpToDate(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.view = viewDetail
	m.selected = monitors3[0] // name="alpha"

	// Receive fresh data where alpha is now down
	updated := []*models.MonitorStatus{
		{Monitor: models.Monitor{Name: "alpha"}, Status: models.StatusDown},
		{Monitor: models.Monitor{Name: "beta"}, Status: models.StatusUp},
	}
	m2, _ := m.Update(dataMsg{monitors: updated})
	got := mustModel(t, m2)

	if got.selected == nil {
		t.Fatal("selected is nil after data update")
	}
	if got.selected.Status != models.StatusDown {
		t.Errorf("selected.Status = %q, want down after update", got.selected.Status)
	}
}

// ── add-form tab navigation ───────────────────────────────────────────────────

func TestAddFormTabAdvancesFocus(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 0

	m2, _ := m.Update(key(tea.KeyTab))
	got := mustModel(t, m2)

	if got.addFocus != 1 {
		t.Errorf("after tab: addFocus = %d, want 1", got.addFocus)
	}
}

func TestAddFormShiftTabGoesBack(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 2

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	got := mustModel(t, m2)

	if got.addFocus != 1 {
		t.Errorf("after shift-tab: addFocus = %d, want 1", got.addFocus)
	}
}

func TestAddFormTabWraps(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4 // last field

	m2, _ := m.Update(key(tea.KeyTab))
	got := mustModel(t, m2)

	if got.addFocus != 0 {
		t.Errorf("tab wrap: addFocus = %d, want 0", got.addFocus)
	}
}

// ── add-form validation ───────────────────────────────────────────────────────

func TestSubmitAddEmptyName(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4 // focus on last field so Enter triggers submit

	// Leave name empty, set required fields
	m.addInputs[1].SetValue("http")
	m.addInputs[2].SetValue("http://example.com")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for empty name")
	}
	if got.view != viewAdd {
		t.Error("should stay on add view after validation error")
	}
}

func TestSubmitAddBadType(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4

	m.addInputs[0].SetValue("my service")
	m.addInputs[1].SetValue("ftp")
	m.addInputs[2].SetValue("ftp://example.com")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for invalid type")
	}
}

func TestSubmitAddEmptyTarget(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4

	m.addInputs[0].SetValue("my service")
	m.addInputs[1].SetValue("http")
	// target left empty

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for empty target")
	}
}

// ── render helpers ────────────────────────────────────────────────────────────

func TestPadR(t *testing.T) {
	tests := []struct {
		s    string
		w    int
		want int // expected visual width
	}{
		{"hello", 10, 10},
		{"hello", 3, 5},  // longer than width: return as-is
		{"", 5, 5},
	}
	for _, tt := range tests {
		got := lipgloss.Width(padR(tt.s, tt.w))
		if got != tt.want {
			t.Errorf("padR(%q, %d): width = %d, want %d", tt.s, tt.w, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello w…"},
		{"hello world", 3, "hel"}, // maxRunes<=3: no room for ellipsis, return first N runes
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.d)
		if got != tt.want {
			t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestSparklineStatusLength(t *testing.T) {
	m := newTestModel()
	history := []models.Result{
		{Status: models.StatusUp},
		{Status: models.StatusDown},
		{Status: models.StatusPending},
	}
	// sparklineStatus always renders exactly 24 visual chars
	got := lipgloss.Width(m.sparklineStatus(history))
	if got != 24 {
		t.Errorf("sparklineStatus width = %d, want 24", got)
	}
}

func TestSparklineStatusEmpty(t *testing.T) {
	m := newTestModel()
	got := m.sparklineStatus(nil)
	if lipgloss.Width(got) != 24 {
		t.Errorf("empty sparkline width = %d, want 24", lipgloss.Width(got))
	}
}

func TestLatencySparklineLength(t *testing.T) {
	m := newTestModel()
	history := make([]models.Result, 10)
	for i := range history {
		history[i] = models.Result{Status: models.StatusUp, Latency: (i + 1) * 10}
	}
	width := 10
	got := lipgloss.Width(m.latencySparkline(history, width))
	if got != width {
		t.Errorf("latencySparkline width = %d, want %d", got, width)
	}
}

func TestLatencySparklineDownMarked(t *testing.T) {
	m := newTestModel()
	history := []models.Result{
		{Status: models.StatusDown, Latency: 0},
	}
	// The sparkline string should contain ▁ (down marker)
	raw := m.latencySparkline(history, 1)
	if !strings.Contains(raw, "▁") {
		t.Error("down result should render as ▁")
	}
}

func TestLatencyStats(t *testing.T) {
	m := newTestModel()
	history := []models.Result{
		{Status: models.StatusUp, Latency: 10},
		{Status: models.StatusUp, Latency: 30},
		{Status: models.StatusUp, Latency: 20},
	}
	got := m.latencyStats(history)
	if !strings.Contains(got, "10") {
		t.Errorf("latencyStats missing min (10): %q", got)
	}
	if !strings.Contains(got, "30") {
		t.Errorf("latencyStats missing max (30): %q", got)
	}
	if !strings.Contains(got, "20") {
		t.Errorf("latencyStats missing avg (20): %q", got)
	}
}

func TestLatencyStatsEmpty(t *testing.T) {
	m := newTestModel()
	got := m.latencyStats(nil)
	if got != "" {
		t.Errorf("latencyStats(nil) = %q, want empty", got)
	}
}

// ── View smoke tests ──────────────────────────────────────────────────────────

func TestDashboardViewRendersTitle(t *testing.T) {
	m := newTestModel()
	m.loading = false
	got := m.dashboardView()
	if !strings.Contains(got, "uptui") {
		t.Error("dashboard view should contain title 'uptui'")
	}
}

func TestDashboardViewNoMonitors(t *testing.T) {
	m := newTestModel()
	m.loading = false
	got := m.dashboardView()
	if !strings.Contains(got, "No monitors") {
		t.Errorf("empty dashboard should say 'No monitors', got: %q", got)
	}
}

func TestDashboardViewLoading(t *testing.T) {
	m := newTestModel()
	// loading=true by default
	got := m.dashboardView()
	if !strings.Contains(got, "Connecting") {
		t.Errorf("loading dashboard should say 'Connecting', got: %q", got)
	}
}

func TestDashboardViewError(t *testing.T) {
	m := newTestModel()
	m.loading = false
	m.err = "connection refused"
	got := m.dashboardView()
	if !strings.Contains(got, "Error") {
		t.Errorf("error dashboard should say 'Error', got: %q", got)
	}
}

func TestDashboardViewWithMonitors(t *testing.T) {
	m := newTestModel()
	m.loading = false
	m.monitors = monitors3
	got := m.dashboardView()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(got, name) {
			t.Errorf("dashboard should contain monitor name %q", name)
		}
	}
}

func TestDetailViewRendersName(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	m.selected = &models.MonitorStatus{
		Monitor:   models.Monitor{Name: "my-service", Type: models.HTTP, Target: "https://x.com", Interval: 60},
		Status:    models.StatusUp,
		Latency:   55,
		LastCheck: time.Now().Add(-30 * time.Second),
	}
	got := m.detailView()
	if !strings.Contains(got, "my-service") {
		t.Errorf("detail view should contain monitor name, got: %q", got)
	}
}

func TestAddViewRendersFields(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	got := m.addView()

	for _, label := range []string{"Name", "Type", "Target", "Interval", "Accepted"} {
		if !strings.Contains(got, label) {
			t.Errorf("add view missing field label %q", label)
		}
	}
}

func TestDashboardViewConfirmPrompt(t *testing.T) {
	m := newTestModel()
	m.loading = false
	m.monitors = monitors3
	m.pendingDelete = "beta"
	got := m.dashboardView()

	if !strings.Contains(got, "beta") {
		t.Error("confirm prompt should contain monitor name")
	}
	if !strings.Contains(got, "Delete") {
		t.Error("confirm prompt should contain 'Delete'")
	}
}

func TestEditConfirmPromptOnSubmit(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "alpha"
	m.addFocus = 4 // last field → submit on Enter

	m.addInputs[0].SetValue("alpha-renamed")
	m.addInputs[1].SetValue("http")
	m.addInputs[2].SetValue("https://alpha.com")
	m.addInputs[3].SetValue("30")

	m2, cmd := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.pendingEdit == nil {
		t.Fatal("pendingEdit should be set after submitting edit form")
	}
	if got.pendingEdit.Name != "alpha-renamed" {
		t.Errorf("pendingEdit.Name = %q, want alpha-renamed", got.pendingEdit.Name)
	}
	if cmd != nil {
		t.Error("should not fire a cmd until confirmed")
	}
	if got.view != viewAdd {
		t.Error("should stay on add/edit view while confirming")
	}
}

func TestEditConfirmCancelOnOtherKey(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "alpha"
	m.pendingEdit = &models.Monitor{Name: "alpha-renamed"}

	m2, cmd := m.Update(rune_('n'))
	got := mustModel(t, m2)

	if got.pendingEdit != nil {
		t.Error("pendingEdit should be cleared after cancel")
	}
	if cmd != nil {
		t.Error("cancel should return no cmd")
	}
	if got.view != viewAdd {
		t.Error("should stay on add/edit view after cancel")
	}
}

func TestEditConfirmCancelOnEsc(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "alpha"
	m.pendingEdit = &models.Monitor{Name: "alpha-renamed"}

	m2, _ := m.Update(key(tea.KeyEsc))
	got := mustModel(t, m2)

	if got.pendingEdit != nil {
		t.Error("pendingEdit should be cleared on esc")
	}
}

func TestAddViewRendersConfirmPrompt(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "alpha"
	m.pendingEdit = &models.Monitor{Name: "alpha-renamed"}

	got := m.addView()
	if !strings.Contains(got, "Save changes") {
		t.Error("confirm prompt should contain 'Save changes'")
	}
	if !strings.Contains(got, "alpha") {
		t.Error("confirm prompt should contain the monitor name")
	}
}

// ── sort / filter / visibleMonitors ──────────────────────────────────────────

func TestSortCyclesOnS(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3

	m2, _ := m.Update(rune_('s'))
	got := mustModel(t, m2)
	if got.sortKey != sortByStatus {
		t.Errorf("after first s: sortKey = %d, want %d (sortByStatus)", got.sortKey, sortByStatus)
	}

	m3, _ := got.Update(rune_('s'))
	got2 := mustModel(t, m3)
	if got2.sortKey != sortByUptime {
		t.Errorf("after second s: sortKey = %d, want %d (sortByUptime)", got2.sortKey, sortByUptime)
	}

	m4, _ := got2.Update(rune_('s'))
	got3 := mustModel(t, m4)
	if got3.sortKey != sortByName {
		t.Errorf("after third s (wrap): sortKey = %d, want %d (sortByName)", got3.sortKey, sortByName)
	}
}

func TestFilterCyclesOnF(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3

	m2, _ := m.Update(rune_('f'))
	got := mustModel(t, m2)
	if got.filterKey != filterDown {
		t.Errorf("after first f: filterKey = %d, want %d (filterDown)", got.filterKey, filterDown)
	}

	m3, _ := got.Update(rune_('f'))
	got2 := mustModel(t, m3)
	if got2.filterKey != filterProblems {
		t.Errorf("after second f: filterKey = %d, want %d (filterProblems)", got2.filterKey, filterProblems)
	}

	m4, _ := got2.Update(rune_('f'))
	got3 := mustModel(t, m4)
	if got3.filterKey != filterAll {
		t.Errorf("after third f (wrap): filterKey = %d, want %d (filterAll)", got3.filterKey, filterAll)
	}
}

func TestVisibleMonitorsFilterAll(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.filterKey = filterAll

	vis := m.visibleMonitors()
	if len(vis) != 3 {
		t.Errorf("filterAll: len = %d, want 3", len(vis))
	}
}

func TestVisibleMonitorsFilterDown(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3 // alpha=up, beta=down, gamma=pending
	m.filterKey = filterDown

	vis := m.visibleMonitors()
	if len(vis) != 1 {
		t.Errorf("filterDown: len = %d, want 1", len(vis))
	}
	if vis[0].Monitor.Name != "beta" {
		t.Errorf("filterDown: name = %q, want beta", vis[0].Monitor.Name)
	}
}

func TestVisibleMonitorsFilterProblems(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3 // alpha=up, beta=down, gamma=pending
	m.filterKey = filterProblems

	vis := m.visibleMonitors()
	if len(vis) != 2 {
		t.Errorf("filterProblems: len = %d, want 2 (down+pending)", len(vis))
	}
}

func TestVisibleMonitorsSortByStatus(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3 // alpha=up, beta=down, gamma=pending
	m.sortKey = sortByStatus

	vis := m.visibleMonitors()
	// down(beta) first, then pending(gamma), then up(alpha)
	if vis[0].Monitor.Name != "beta" {
		t.Errorf("sort by status [0]: name = %q, want beta (down)", vis[0].Monitor.Name)
	}
	if vis[1].Monitor.Name != "gamma" {
		t.Errorf("sort by status [1]: name = %q, want gamma (pending)", vis[1].Monitor.Name)
	}
	if vis[2].Monitor.Name != "alpha" {
		t.Errorf("sort by status [2]: name = %q, want alpha (up)", vis[2].Monitor.Name)
	}
}

func TestStatusOrder(t *testing.T) {
	if statusOrder(models.StatusDown) >= statusOrder(models.StatusPending) {
		t.Error("down should sort before pending")
	}
	if statusOrder(models.StatusPending) >= statusOrder(models.StatusPaused) {
		t.Error("pending should sort before paused")
	}
	if statusOrder(models.StatusPaused) >= statusOrder(models.StatusUp) {
		t.Error("paused should sort before up")
	}
}

// ── detail scroll ─────────────────────────────────────────────────────────────

func TestDetailScrollDownAddsOlder(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	history := make([]models.Result, 20)
	for i := range history {
		history[i] = models.Result{Status: models.StatusUp, Latency: i + 1}
	}
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}, History: history}

	m2, _ := m.Update(key(tea.KeyDown))
	got := mustModel(t, m2)
	if got.detailScroll != 1 {
		t.Errorf("detailScroll = %d, want 1 after ↓", got.detailScroll)
	}
}

func TestDetailScrollUpDecreases(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	m.detailScroll = 5
	history := make([]models.Result, 20)
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}, History: history}

	m2, _ := m.Update(key(tea.KeyUp))
	got := mustModel(t, m2)
	if got.detailScroll != 4 {
		t.Errorf("detailScroll = %d, want 4 after ↑", got.detailScroll)
	}
}

func TestDetailScrollClampAtMax(t *testing.T) {
	m := newTestModel() // height=24, pageSize = 24-13 = 11
	m.view = viewDetail
	history := make([]models.Result, 20)
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}, History: history}
	m.detailScroll = 8 // maxScroll = 20-11 = 9

	m2, _ := m.Update(key(tea.KeyDown))
	got := mustModel(t, m2)
	// Scroll increases to 9 (maxScroll)
	if got.detailScroll != 9 {
		t.Errorf("detailScroll = %d, want 9 (maxScroll)", got.detailScroll)
	}
	// Another down: stays at 9
	m3, _ := got.Update(key(tea.KeyDown))
	got2 := mustModel(t, m3)
	if got2.detailScroll != 9 {
		t.Errorf("detailScroll = %d, want 9 (clamped at max)", got2.detailScroll)
	}
}

func TestDetailScrollClampAtZero(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	m.detailScroll = 0
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}}

	m2, _ := m.Update(key(tea.KeyUp))
	got := mustModel(t, m2)
	if got.detailScroll != 0 {
		t.Errorf("detailScroll = %d, want 0 (cannot go negative)", got.detailScroll)
	}
}

func TestDetailScrollResetOnBack(t *testing.T) {
	m := newTestModel()
	m.view = viewDetail
	m.detailScroll = 5
	m.selected = &models.MonitorStatus{Monitor: models.Monitor{Name: "test"}}

	m2, _ := m.Update(key(tea.KeyEsc))
	got := mustModel(t, m2)
	if got.detailScroll != 0 {
		t.Errorf("detailScroll = %d, want 0 after esc", got.detailScroll)
	}
}

func TestDetailScrollResetOnEnter(t *testing.T) {
	m := newTestModel()
	m.monitors = monitors3
	m.cursor = 0
	m.detailScroll = 7

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)
	if got.detailScroll != 0 {
		t.Errorf("detailScroll = %d, want 0 after enter (entering detail view)", got.detailScroll)
	}
}

func TestDetailPageSize(t *testing.T) {
	n := detailPageSize(24)
	if n < 5 {
		t.Errorf("detailPageSize(24) = %d, want >= 5", n)
	}
	// Minimum enforced
	small := detailPageSize(5)
	if small != 5 {
		t.Errorf("detailPageSize(5) = %d, want 5 (minimum)", small)
	}
}

func TestDetailViewScrollIndicator(t *testing.T) {
	m := newTestModel() // height=24, pageSize=11
	// 20 history items; maxScroll = 9; with scroll=5 both indicators should appear
	history := make([]models.Result, 20)
	for i := range history {
		history[i] = models.Result{Status: models.StatusUp, Latency: i + 1}
	}
	m.view = viewDetail
	m.selected = &models.MonitorStatus{
		Monitor: models.Monitor{Name: "test"},
		History: history,
	}
	m.detailScroll = 5

	got := m.detailView()
	if !strings.Contains(got, "newer") {
		t.Error("scrolled view should show 'newer' indicator")
	}
	if !strings.Contains(got, "older") {
		t.Error("scrolled view should show 'older' indicator")
	}
}

func TestDetailViewNoScrollIndicatorAtTop(t *testing.T) {
	m := newTestModel() // height=24, pageSize=11
	history := make([]models.Result, 20)
	for i := range history {
		history[i] = models.Result{Status: models.StatusUp, Latency: i + 1}
	}
	m.view = viewDetail
	m.selected = &models.MonitorStatus{
		Monitor: models.Monitor{Name: "test"},
		History: history,
	}
	// scroll=0: at most-recent end, should show "older" but no "newer"
	m.detailScroll = 0
	got := m.detailView()
	if strings.Contains(got, "newer") {
		t.Error("scroll=0 should not show 'newer' indicator")
	}
	if !strings.Contains(got, "older") {
		t.Error("scroll=0 with 20 items should show 'older' indicator")
	}
}

func TestDashboardViewSortFilterInFooter(t *testing.T) {
	m := newTestModel()
	m.loading = false
	m.monitors = monitors3
	got := m.dashboardView()
	// Footer should contain current sort and filter hints
	if !strings.Contains(got, "name") {
		t.Error("footer should contain current sort (name)")
	}
	if !strings.Contains(got, "all") {
		t.Error("footer should contain current filter (all)")
	}
}

// ── dashboard viewport scroll ─────────────────────────────────────────────────

func TestDashboardRows(t *testing.T) {
	if dashboardRows(24) != 18 {
		t.Errorf("dashboardRows(24) = %d, want 18", dashboardRows(24))
	}
	if dashboardRows(6) != 1 { // minimum 1
		t.Errorf("dashboardRows(6) = %d, want 1", dashboardRows(6))
	}
}

func TestClampListOffset(t *testing.T) {
	// cursor above viewport → offset moves up
	if got := clampListOffset(5, 3, 10); got != 3 {
		t.Errorf("clampListOffset(5,3,10) = %d, want 3", got)
	}
	// cursor within viewport → offset unchanged
	if got := clampListOffset(5, 7, 10); got != 5 {
		t.Errorf("clampListOffset(5,7,10) = %d, want 5", got)
	}
	// cursor below viewport → offset scrolls down
	if got := clampListOffset(0, 12, 10); got != 3 {
		t.Errorf("clampListOffset(0,12,10) = %d, want 3", got)
	}
}

func TestDashboardScrollsToFollowCursor(t *testing.T) {
	// Build a list larger than the default viewport (height=24 → rows=18)
	var many []*models.MonitorStatus
	for i := 0; i < 30; i++ {
		many = append(many, &models.MonitorStatus{
			Monitor: models.Monitor{Name: fmt.Sprintf("svc-%02d", i)},
			Status:  models.StatusUp,
		})
	}

	m := newTestModel()
	m.monitors = many
	m.loading = false

	// Move cursor to row 20 (beyond default viewport of 18)
	for i := 0; i < 20; i++ {
		m2, _ := m.Update(key(tea.KeyDown))
		m = mustModel(t, m2)
	}
	if m.cursor != 20 {
		t.Fatalf("cursor = %d, want 20", m.cursor)
	}

	// listOffset must have scrolled so cursor is visible
	rows := dashboardRows(m.height)
	if m.cursor < m.listOffset || m.cursor >= m.listOffset+rows {
		t.Errorf("cursor %d not in viewport [%d, %d)", m.cursor, m.listOffset, m.listOffset+rows)
	}

	// The rendered dashboard must contain the cursor row's service name
	got := m.dashboardView()
	if !strings.Contains(got, "svc-20") {
		t.Error("dashboardView should show the row at the cursor when scrolled")
	}
	// The very first service should be off-screen
	if strings.Contains(got, "svc-00") {
		t.Error("svc-00 should be scrolled off-screen")
	}
}

// ── target format validation ──────────────────────────────────────────────────

func TestSubmitPortTypeNormalized(t *testing.T) {
	// "port" is a legacy alias for "tcp" — the form should accept it and normalize.
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true // avoid real client call
	m.editOldName = "old"
	m.addFocus = 4

	m.addInputs[0].SetValue("ssh")
	m.addInputs[1].SetValue("port") // legacy alias
	m.addInputs[2].SetValue("localhost:22")
	m.addInputs[3].SetValue("30")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr != "" {
		t.Errorf("unexpected error for 'port' type alias: %q", got.addErr)
	}
	if got.pendingEdit == nil {
		t.Fatal("pendingEdit should be set")
	}
	if string(got.pendingEdit.Type) != "tcp" {
		t.Errorf("pendingEdit.Type = %q, want tcp (normalized from port)", got.pendingEdit.Type)
	}
}

func TestSubmitHTTPNoProtocol(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4

	m.addInputs[0].SetValue("my service")
	m.addInputs[1].SetValue("http")
	m.addInputs[2].SetValue("example.com") // missing http://

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for HTTP target without protocol")
	}
	if got.view != viewAdd {
		t.Error("should stay on add view after validation error")
	}
}

func TestSubmitTCPNoPort(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4

	m.addInputs[0].SetValue("my db")
	m.addInputs[1].SetValue("tcp")
	m.addInputs[2].SetValue("localhost") // missing port

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for TCP target without port")
	}
}

func TestSubmitTCPInvalidPort(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.addFocus = 4

	m.addInputs[0].SetValue("my db")
	m.addInputs[1].SetValue("tcp")
	m.addInputs[2].SetValue("localhost:99999") // port out of range

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected validation error for out-of-range port")
	}
}

func TestSubmitTCPValidTarget(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true // avoid real client.Add() call
	m.editOldName = "old"
	m.addFocus = 4

	m.addInputs[0].SetValue("postgres")
	m.addInputs[1].SetValue("tcp")
	m.addInputs[2].SetValue("localhost:5432")
	m.addInputs[3].SetValue("30")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr != "" {
		t.Errorf("unexpected error for valid TCP target: %q", got.addErr)
	}
	if got.pendingEdit == nil {
		t.Error("pendingEdit should be set for valid TCP target in edit mode")
	}
}

// ── accepted statuses form field ──────────────────────────────────────────────

func TestSubmitHTTPAcceptedStatusesValid(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "old"
	m.addFocus = 4

	m.addInputs[0].SetValue("API")
	m.addInputs[1].SetValue("http")
	m.addInputs[2].SetValue("https://api.example.com")
	m.addInputs[3].SetValue("60")
	m.addInputs[4].SetValue("200-299,401")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr != "" {
		t.Errorf("unexpected error for valid accepted statuses: %q", got.addErr)
	}
	if got.pendingEdit == nil {
		t.Fatal("pendingEdit should be set")
	}
	if got.pendingEdit.AcceptedStatuses != "200-299,401" {
		t.Errorf("AcceptedStatuses = %q, want %q", got.pendingEdit.AcceptedStatuses, "200-299,401")
	}
}

func TestSubmitHTTPAcceptedStatusesInvalidFormat(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "old"
	m.addFocus = 4

	m.addInputs[0].SetValue("API")
	m.addInputs[1].SetValue("http")
	m.addInputs[2].SetValue("https://api.example.com")
	m.addInputs[3].SetValue("60")
	m.addInputs[4].SetValue("not-a-code")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr == "" {
		t.Error("expected error for invalid accepted statuses format")
	}
	if got.pendingEdit != nil {
		t.Error("pendingEdit should not be set on validation error")
	}
}

func TestSubmitTCPAcceptedStatusesIgnored(t *testing.T) {
	// AcceptedStatuses is HTTP-only; for TCP it should be silently cleared
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	m.editOldName = "old"
	m.addFocus = 4

	m.addInputs[0].SetValue("postgres")
	m.addInputs[1].SetValue("tcp")
	m.addInputs[2].SetValue("localhost:5432")
	m.addInputs[3].SetValue("30")
	m.addInputs[4].SetValue("200-299")

	m2, _ := m.Update(key(tea.KeyEnter))
	got := mustModel(t, m2)

	if got.addErr != "" {
		t.Errorf("unexpected error: %q", got.addErr)
	}
	if got.pendingEdit == nil {
		t.Fatal("pendingEdit should be set")
	}
	if got.pendingEdit.AcceptedStatuses != "" {
		t.Errorf("AcceptedStatuses should be empty for TCP, got %q", got.pendingEdit.AcceptedStatuses)
	}
}

func TestEditViewRendersEditTitle(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	got := m.addView()

	if !strings.Contains(got, "Edit Monitor") {
		t.Errorf("edit view should say 'Edit Monitor', got: %q", got)
	}
}
