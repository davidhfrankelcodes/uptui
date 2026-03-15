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
	if len(m.addInputs) != 4 {
		t.Errorf("addInputs len = %d, want 4", len(m.addInputs))
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
	m.addFocus = 3 // last field

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
	m.addFocus = 3 // focus on last field so Enter triggers submit

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
	m.addFocus = 3

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
	m.addFocus = 3

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

	for _, label := range []string{"Name", "Type", "Target", "Interval"} {
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
	m.addFocus = 3 // last field → submit on Enter

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

func TestEditViewRendersEditTitle(t *testing.T) {
	m := newTestModel()
	m.view = viewAdd
	m.editMode = true
	got := m.addView()

	if !strings.Contains(got, "Edit Monitor") {
		t.Errorf("edit view should say 'Edit Monitor', got: %q", got)
	}
}
