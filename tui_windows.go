package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenLoading screen = iota
	screenHome
	screenAudit
	screenPlan
	screenConfirm
	screenWorking
	screenResult
	screenHelp
)

type hitbox struct {
	x1, y1, x2, y2 int
	action         string
}

type intent struct {
	kind    string
	profile string
}

type auditMsg struct {
	audit    Audit
	target   screen
	navigate bool
}
type planMsg struct {
	plan Plan
	err  error
}
type operationMsg struct{ result OperationResult }
type spinMsg time.Time

type tuiModel struct {
	width, height int
	screen        screen
	audit         Audit
	plan          Plan
	result        OperationResult
	pending       intent
	hitboxes      []hitbox
	hover         string
	scroll        int
	spin          int
	loadingText   string
}

var (
	colorPurple = lipgloss.Color("#8B5CF6")
	colorViolet = lipgloss.Color("#C4B5FD")
	colorGreen  = lipgloss.Color("#34D399")
	colorRed    = lipgloss.Color("#FB7185")
	colorAmber  = lipgloss.Color("#FBBF24")
	colorMuted  = lipgloss.Color("#94A3B8")
	colorPanel  = lipgloss.Color("#1E293B")
	colorHover  = lipgloss.Color("#4C1D95")
	colorText   = lipgloss.Color("#F8FAFC")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorText).Background(colorPurple).Padding(0, 2)
	cardStyle  = lipgloss.NewStyle().Foreground(colorText).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#475569")).Padding(0, 1)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	goodStyle  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(colorAmber).Bold(true)
	badStyle   = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
)

func runTUI() error {
	model := &tuiModel{screen: screenLoading, width: 100, height: 32, loadingText: "Проверяю железо и состояние Windows"}
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion())
	_, err := program.Run()
	return err
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(loadAuditCmd(screenHome, true), spinCmd())
}

func loadAuditCmd(target screen, navigate bool) tea.Cmd {
	return func() tea.Msg { return auditMsg{audit: collectAudit(), target: target, navigate: navigate} }
}

func loadPlanCmd(profile string) tea.Cmd {
	return func() tea.Msg {
		plan, err := buildPlan(profile)
		return planMsg{plan: plan, err: err}
	}
}

func spinCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return spinMsg(t) })
}

func (m *tuiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case auditMsg:
		m.audit = msg.audit
		if msg.navigate {
			m.screen = msg.target
		}
	case planMsg:
		if msg.err != nil {
			m.result = OperationResult{Title: "Не удалось построить план", Summary: msg.err.Error(), Err: msg.err}
			m.screen = screenResult
		} else {
			m.plan = msg.plan
			m.screen = screenPlan
		}
		m.scroll = 0
	case operationMsg:
		m.result = msg.result
		m.screen = screenResult
		m.scroll = 0
		return m, loadAuditCmd(screenResult, false)
	case spinMsg:
		m.spin = (m.spin + 1) % 8
		if m.screen == screenLoading || m.screen == screenWorking {
			return m, spinCmd()
		}
	case tea.MouseMsg:
		event := tea.MouseEvent(msg)
		m.hover = m.actionAt(event.X, event.Y)
		if event.Button == tea.MouseButtonWheelUp && event.Action == tea.MouseActionPress {
			m.scroll -= 3
			if m.scroll < 0 {
				m.scroll = 0
			}
		}
		if event.Button == tea.MouseButtonWheelDown && event.Action == tea.MouseActionPress {
			m.scroll += 3
		}
		if event.Button == tea.MouseButtonLeft && event.Action == tea.MouseActionPress {
			return m.handleAction(m.actionAt(event.X, event.Y))
		}
	case tea.KeyMsg:
		// Mouse is the primary UI. Esc is only a fallback for terminals without mouse events.
		if msg.String() == "esc" && m.screen != screenHome && m.screen != screenLoading && m.screen != screenWorking {
			m.screen = screenHome
			m.scroll = 0
		}
	}
	return m, nil
}

func (m *tuiModel) handleAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "exit":
		return m, tea.Quit
	case "home":
		m.screen, m.scroll = screenHome, 0
	case "audit":
		m.screen, m.scroll = screenAudit, 0
	case "refresh-audit":
		m.screen, m.loadingText = screenLoading, "Обновляю аудит"
		return m, tea.Batch(loadAuditCmd(screenAudit, true), spinCmd())
	case "help":
		m.screen, m.scroll = screenHelp, 0
	case "plan-recommended":
		m.screen, m.loadingText = screenLoading, "Строю рекомендуемый план"
		return m, tea.Batch(loadPlanCmd(profileRecommended), spinCmd())
	case "plan-maximum":
		m.screen, m.loadingText = screenLoading, "Проверяю возможности CPU, GPU и Ethernet"
		return m, tea.Batch(loadPlanCmd(profileMaximum), spinCmd())
	case "apply-plan":
		m.pending = intent{kind: "apply", profile: m.plan.Profile.ID}
		m.screen = screenConfirm
	case "confirm-clean":
		m.pending = intent{kind: "clean"}
		m.screen = screenConfirm
	case "confirm-restore":
		m.pending = intent{kind: "restore"}
		m.screen = screenConfirm
	case "cancel":
		m.screen = screenHome
	case "run":
		m.screen, m.loadingText = screenWorking, operationTitle(m.pending)
		return m, tea.Batch(executeIntentCmd(m.pending), spinCmd())
	}
	return m, nil
}

func executeIntentCmd(pending intent) tea.Cmd {
	return func() tea.Msg {
		result := OperationResult{Title: operationTitle(pending)}
		switch pending.kind {
		case "apply":
			if isAdministrator() {
				result.BackupPath, result.Err = applyProfile(pending.profile)
			} else {
				args := []string{"apply", "--profile", pending.profile, "--yes", "--quiet"}
				args = append(args, "--parent-pid", fmt.Sprintf("%d", os.Getpid()))
				result.Err = runElevatedAndWait(args)
			}
			result.Summary = "Профиль применён и проверен повторным чтением состояния."
		case "restore":
			if isAdministrator() {
				sid, err := currentUserSID()
				if err != nil {
					result.Err = err
				} else {
					result.BackupPath, result.Err = restoreLatest(sid)
				}
			} else {
				result.Err = runElevatedAndWait([]string{"restore", "--yes", "--quiet", "--parent-pid", fmt.Sprintf("%d", os.Getpid())})
			}
			result.Summary = "Последняя операция отменена и состояние проверено."
		case "clean":
			cleaned, err := cleanTemporaryFiles(2)
			result.Err = err
			result.Summary = fmt.Sprintf("Удалено %d файлов и %d папок (%.1f МБ). Пропущено: %d.", cleaned.Files, cleaned.Dirs, float64(cleaned.Bytes)/(1024*1024), cleaned.Skipped)
		}
		if result.Err != nil {
			result.Title = "Операция не завершена"
			result.Summary = result.Err.Error()
		} else {
			result.Title = "Готово"
		}
		return operationMsg{result: result}
	}
}

func operationTitle(pending intent) string {
	switch pending.kind {
	case "restore":
		return "Восстанавливаю последнюю резервную копию"
	case "clean":
		return "Безопасно очищаю временные файлы"
	case "apply":
		return "Применяю и проверяю профиль"
	default:
		return "Выполняю операцию"
	}
}

func (m *tuiModel) actionAt(x, y int) string {
	for _, box := range m.hitboxes {
		if x >= box.x1 && x <= box.x2 && y >= box.y1 && y <= box.y2 {
			return box.action
		}
	}
	return ""
}

func (m *tuiModel) View() string {
	m.hitboxes = nil
	if m.width < 54 || m.height < 20 {
		width := m.width
		if width < 1 {
			width = 1
		}
		message := fitText("Увеличьте окно терминала минимум до 54×20.", width)
		button := fitText("  Выйти", width)
		m.hitboxes = append(m.hitboxes, hitbox{x1: 0, y1: 2, x2: width - 1, y2: 2, action: "exit"})
		return message + "\n\n" + lipgloss.NewStyle().Foreground(colorText).Background(colorPanel).Render(button)
	}
	contentWidth := m.contentWidth()
	header := fitText("GOFMAN3 OPTIMIZER  "+version, contentWidth-4)
	subtitle := fitText("Windows performance toolkit • mouse-first • полный откат", contentWidth)
	lines := []string{"", "  " + titleStyle.Render(header), "  " + mutedStyle.Render(subtitle)}
	lines = append(lines, "")
	switch m.screen {
	case screenLoading, screenWorking:
		m.renderLoading(&lines)
	case screenHome:
		m.renderHome(&lines)
	case screenAudit:
		m.renderAudit(&lines)
	case screenPlan:
		m.renderPlan(&lines)
	case screenConfirm:
		m.renderConfirm(&lines)
	case screenResult:
		m.renderResult(&lines)
	case screenHelp:
		m.renderHelp(&lines)
	}
	return strings.Join(lines, "\n")
}

func (m *tuiModel) renderLoading(lines *[]string) {
	frames := []string{"◐", "◓", "◑", "◒", "◐", "◓", "◑", "◒"}
	*lines = append(*lines, "", "  "+lipgloss.NewStyle().Foreground(colorViolet).Bold(true).Render(frames[m.spin]+"  "+m.loadingText), "", "  "+mutedStyle.Render("Никакие настройки не меняются без отдельного подтверждения."))
}

func (m *tuiModel) renderHome(lines *[]string) {
	contentWidth := m.contentWidth()
	cpu := "Не определён"
	if len(m.audit.Hardware.CPUs) > 0 {
		item := m.audit.Hardware.CPUs[0]
		cpu = fmt.Sprintf("%s  •  %dC/%dT", item.Name, item.Cores, item.Logical)
	}
	var gpuNames []string
	for _, gpu := range m.audit.Hardware.GPUs {
		gpuNames = append(gpuNames, gpu.Name+" ["+gpu.Vendor+"]")
	}
	if len(gpuNames) == 0 {
		gpuNames = []string{"Не определена"}
	}
	status := goodStyle.Render("рекомендуемый профиль применён")
	if len(m.audit.Findings) > 0 {
		status = warnStyle.Render("есть настройки к применению")
	}
	card := cardStyle.Width(contentWidth - 4).Render("CPU  " + fitText(cpu, contentWidth-10) + "\nGPU  " + fitText(strings.Join(gpuNames, ", "), contentWidth-10) + "\nСТАТУС  " + status)
	*lines = append(*lines, prefixLines(card, "  ")...)
	*lines = append(*lines, "")
	m.addButton(lines, "audit", "Проверить ПК", "read-only аудит железа и статуса игровой оптимизации")
	m.addButton(lines, "plan-recommended", "Рекомендуемый профиль", "Game Mode, capture, мышь и лёгкий интерфейс")
	m.addButton(lines, "plan-maximum", "Максимальная производительность", "отдельная power-схема + поддерживаемый Ethernet low-latency")
	m.addButton(lines, "confirm-clean", "Очистить временные файлы", "только файлы старше 48 часов, без Prefetch и логов")
	m.addButton(lines, "confirm-restore", "Откатить последнее", "точно восстановить сохранённые значения")
	m.addButton(lines, "help", "Что именно делает софт", "поддержка железа, ограничения и проверка результата")
	m.addButton(lines, "exit", "Выйти", "закрыть без изменений")
}

func (m *tuiModel) renderAudit(lines *[]string) {
	var content []string
	content = append(content, goodStyle.Render("АУДИТ СИСТЕМЫ"), fmt.Sprintf("Windows: %s %s (build %s)", m.audit.Hardware.OS.Caption, m.audit.Hardware.OS.Architecture, m.audit.Hardware.OS.BuildNumber), "Power GUID: "+m.audit.ActivePowerGUID, "")
	if len(m.audit.Findings) == 0 {
		content = append(content, goodStyle.Render("Рекомендуемый игровой профиль применён полностью."))
	} else {
		for _, finding := range m.audit.Findings {
			content = append(content, warnStyle.Render(finding.Title), "  "+finding.Evidence, "  Действие: "+finding.Action, "")
		}
	}
	for _, warning := range m.audit.Warnings {
		content = append(content, mutedStyle.Render("Примечание: "+warning))
	}
	m.addViewport(lines, content, m.height-12)
	m.addButton(lines, "refresh-audit", "Обновить аудит", "повторить все read-only проверки")
	if len(m.audit.Findings) > 0 {
		m.addButton(lines, "plan-recommended", "Открыть рекомендуемый план", "посмотреть точные изменения без применения")
	}
	m.addButton(lines, "home", "Назад", "главный экран")
}

func (m *tuiModel) renderPlan(lines *[]string) {
	changed := 0
	for _, item := range m.plan.Items {
		if item.Changed {
			changed++
		}
	}
	content := []string{goodStyle.Render(m.plan.Profile.Name), m.plan.Profile.Description, fmt.Sprintf("Изменений: %d из %d", changed, len(m.plan.Items)), ""}
	for _, warning := range m.plan.Warnings {
		content = append(content, warnStyle.Render("ВНИМАНИЕ • "+warning), "")
	}
	for _, item := range m.plan.Items {
		marker := goodStyle.Render("БЕЗ ИЗМЕНЕНИЙ")
		if item.Changed {
			marker = lipgloss.NewStyle().Foreground(colorViolet).Bold(true).Render("ИЗМЕНИТЬ")
		}
		content = append(content, marker+"  "+item.Category+" • "+item.Name, "  "+item.Current+"  →  "+item.Desired, "")
	}
	m.addViewport(lines, content, m.height-12)
	m.addButton(lines, "apply-plan", "Применить этот план", "backup → apply → verify; ошибка вызывает rollback")
	m.addButton(lines, "home", "Отмена", "вернуться без изменений")
}

func (m *tuiModel) renderConfirm(lines *[]string) {
	title := operationTitle(m.pending)
	text := "Перед изменением будет создана точная резервная копия. При ошибке движок выполнит автоматический откат."
	if m.pending.kind == "clean" {
		text = "Будут удалены только обычные временные файлы старше 48 часов. Junction/symlink, Prefetch, Event Logs, Windows Update и корзина не затрагиваются."
	}
	box := cardStyle.BorderForeground(colorAmber).Width(m.contentWidth() - 4).Render(warnStyle.Render("ПОДТВЕРЖДЕНИЕ") + "\n" + title + "\n\n" + text)
	*lines = append(*lines, prefixLines(box, "  ")...)
	*lines = append(*lines, "")
	m.addButton(lines, "run", "Подтвердить", "запустить выбранную операцию")
	m.addButton(lines, "cancel", "Отмена", "ничего не менять")
}

func (m *tuiModel) renderResult(lines *[]string) {
	style := goodStyle
	if m.result.Err != nil {
		style = badStyle
	}
	content := []string{style.Render(m.result.Title), m.result.Summary}
	if m.result.BackupPath != "" {
		content = append(content, "Backup: "+m.result.BackupPath)
	}
	m.addViewport(lines, content, m.height-11)
	m.addButton(lines, "refresh-audit", "Посмотреть свежий аудит", "дождаться новой проверки итогового состояния")
	m.addButton(lines, "home", "На главный экран", "продолжить работу")
}

func (m *tuiModel) renderHelp(lines *[]string) {
	content := []string{
		goodStyle.Render("КАК УСТРОЕНА ПОДДЕРЖКА ЖЕЛЕЗА"),
		"Софт читает реальные CPU/GPU через CIM и возможности Windows. Он не хранит фейковый список моделей, поэтому новые Intel, AMD, Qualcomm, NVIDIA и другие устройства определяются без обновления каталога.", "",
		goodStyle.Render("РЕКОМЕНДУЕМЫЙ ПРОФИЛЬ"),
		"Только документированные пользовательские настройки: Game Mode, отключение фоновой записи, отключение ускорения мыши и тяжёлых анимаций.", "",
		goodStyle.Render("МАКСИМАЛЬНЫЙ ПРОФИЛЬ"),
		"Применяет поддерживаемые CPU EPP/Boost и создаёт отдельную AC power-схему, не портит текущую. На физическом Ethernet меняет только объявленные драйвером EEE и Interrupt Moderation. Также убирает startup-delay и задержку меню. Wi-Fi, виртуальные адаптеры и неизвестные свойства пропускаются.", "",
		goodStyle.Render("ЧЕГО ЗДЕСЬ НЕТ"),
		"Нет отключения Defender/Firewall/Spectre/CFG/SEHOP, HPET/BCD-магии, фиксированных affinity masks, private GPU registry keys, разгона и загрузки mutable EXE. Такие действия либо опасны, либо не универсальны, либо не имеют честно измеримого выигрыша.", "",
		goodStyle.Render("КАК ПРОВЕРЯТЬ ЭФФЕКТ"),
		"Сравнивайте одинаковую сцену игры, разрешение, лимит FPS, драйвер и фоновые процессы. Запишите минимум три прогона frametime до и после; смотрите median, 1% low и стабильность, а не один пик FPS.",
	}
	m.addViewport(lines, content, m.height-10)
	m.addButton(lines, "home", "Понятно", "вернуться на главный экран")
}

func (m *tuiModel) addViewport(lines *[]string, content []string, maxHeight int) {
	wrapped := make([]string, 0, len(content))
	width := m.contentWidth() - 2
	for _, line := range content {
		rendered := lipgloss.NewStyle().Width(width).Render(line)
		wrapped = append(wrapped, strings.Split(rendered, "\n")...)
	}
	if maxHeight < 4 {
		maxHeight = 4
	}
	maxScroll := len(wrapped) - maxHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	end := m.scroll + maxHeight
	if end > len(wrapped) {
		end = len(wrapped)
	}
	for _, line := range wrapped[m.scroll:end] {
		*lines = append(*lines, "  "+line)
	}
	for len(wrapped[m.scroll:end]) < maxHeight {
		*lines = append(*lines, "")
		maxHeight--
	}
	if maxScroll > 0 {
		*lines = append(*lines, "  "+mutedStyle.Render(fmt.Sprintf("Колесо мыши: %d/%d", m.scroll, maxScroll)))
	}
}

func (m *tuiModel) addButton(lines *[]string, action, title, description string) {
	width := m.contentWidth()
	plain := "  " + title + "  —  " + description
	plain = fitText(plain, width)
	style := lipgloss.NewStyle().Foreground(colorText).Background(colorPanel).Bold(true)
	if m.hover == action {
		style = style.Background(colorHover).Foreground(lipgloss.Color("#FFFFFF"))
	}
	y := len(*lines)
	*lines = append(*lines, "  "+style.Render(plain))
	m.hitboxes = append(m.hitboxes, hitbox{x1: 2, y1: y, x2: 2 + width - 1, y2: y, action: action})
}

func (m *tuiModel) contentWidth() int {
	width := m.width - 4
	if width > 104 {
		width = 104
	}
	if width < 50 {
		width = 50
	}
	return width
}

func fitText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	for lipgloss.Width(text) > width {
		runes := []rune(text)
		if len(runes) <= 1 {
			return ""
		}
		text = string(runes[:len(runes)-1])
	}
	padding := width - lipgloss.Width(text)
	return text + strings.Repeat(" ", padding)
}

func prefixLines(value, prefix string) []string {
	parts := strings.Split(value, "\n")
	for i := range parts {
		parts[i] = prefix + parts[i]
	}
	return parts
}
