package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

var DefaultCronConfigPath = defaultCronConfigPath()

func defaultCronConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".glaw", "cron.json")
	}
	return filepath.Join(home, ".glaw", "cron.json")
}

type ScheduledTask struct {
	Name     string   `json:"name"`
	Schedule string   `json:"schedule"`
	Type     string   `json:"type"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Prompt   string   `json:"prompt"`
	Hours    []int    `json:"hours"`
	Enabled  *bool    `json:"enabled"`
	WorkDir  string   `json:"workdir"`
}

type Scheduler struct {
	configPath string
	dispatchCh chan<- DispatchRequest
	lastRun    map[string]string
}

func NewScheduler(configPath string, dispatchCh chan<- DispatchRequest) *Scheduler {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = DefaultCronConfigPath
	}
	return &Scheduler{
		configPath: configPath,
		dispatchCh: dispatchCh,
		lastRun:    make(map[string]string),
	}
}

func (s *Scheduler) Run(stopChan <-chan bool) {
	log.Printf("[scheduler] [*] Watching %s", s.configPath)
	s.runDueTasks(time.Now())

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			log.Printf("[scheduler] [*] Stopping...")
			return
		case now := <-ticker.C:
			s.runDueTasks(now)
		}
	}
}

func (s *Scheduler) runDueTasks(now time.Time) {
	tasks, err := loadScheduledTasks(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("[scheduler] [!] Load %s failed: %v", s.configPath, err)
		return
	}

	for i, task := range tasks {
		if !task.isEnabled() {
			continue
		}

		slot, shouldRun, err := task.runSlot(now)
		if err != nil {
			log.Printf("[scheduler] [!] Skip task #%d (%s): %v", i, task.displayName(i), err)
			continue
		}
		if !shouldRun {
			continue
		}

		taskKey := task.key(i)
		if s.lastRun[taskKey] == slot {
			continue
		}

		if err := s.executeTask(task, i); err != nil {
			log.Printf("[scheduler] [!] Task %s failed: %v", task.displayName(i), err)
			continue
		}
		s.lastRun[taskKey] = slot
	}
}

func loadScheduledTasks(configPath string) ([]ScheduledTask, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var tasks []ScheduledTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func LoadScheduledTasks(configPath string) ([]ScheduledTask, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = DefaultCronConfigPath
	}
	return loadScheduledTasks(configPath)
}

func (t ScheduledTask) isEnabled() bool {
	return t.Enabled == nil || *t.Enabled
}

func (t ScheduledTask) normalizedType() string {
	switch strings.ToLower(strings.TrimSpace(t.Type)) {
	case "", "program":
		return "program"
	case "ai":
		return "ai"
	default:
		return strings.ToLower(strings.TrimSpace(t.Type))
	}
}

func (t ScheduledTask) normalizedSchedule() string {
	switch strings.ToLower(strings.TrimSpace(t.Schedule)) {
	case "":
		return "hourly"
	default:
		return strings.ToLower(strings.TrimSpace(t.Schedule))
	}
}

func (t ScheduledTask) displayName(index int) string {
	name := strings.TrimSpace(t.Name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("#%d", index)
}

func (t ScheduledTask) key(index int) string {
	if name := strings.TrimSpace(t.Name); name != "" {
		return name
	}
	return fmt.Sprintf("task-%d", index)
}

func (t ScheduledTask) DisplayName(index int) string {
	return t.displayName(index)
}

func (t ScheduledTask) IsEnabled() bool {
	return t.isEnabled()
}

func (t ScheduledTask) NormalizedType() string {
	return t.normalizedType()
}

func (t ScheduledTask) NormalizedSchedule() string {
	return t.normalizedSchedule()
}

func (t ScheduledTask) runSlot(now time.Time) (string, bool, error) {
	switch t.normalizedSchedule() {
	case "hourly":
		if now.Minute() != 0 {
			return "", false, nil
		}
		return now.Format("2006-01-02T15"), true, nil
	case "daily":
		if now.Minute() != 0 {
			return "", false, nil
		}
		hours, err := t.validatedHours()
		if err != nil {
			return "", false, err
		}
		if !slices.Contains(hours, now.Hour()) {
			return "", false, nil
		}
		return now.Format("2006-01-02T15"), true, nil
	default:
		return "", false, fmt.Errorf("unsupported schedule %q", t.Schedule)
	}
}

func (t ScheduledTask) RunSlot(now time.Time) (string, bool, error) {
	return t.runSlot(now)
}

func (t ScheduledTask) validatedHours() ([]int, error) {
	if len(t.Hours) == 0 {
		return nil, fmt.Errorf("daily task requires non-empty hours")
	}

	seen := make(map[int]struct{}, len(t.Hours))
	var hours []int
	for _, hour := range t.Hours {
		if hour < 0 || hour > 23 {
			return nil, fmt.Errorf("invalid hour %d", hour)
		}
		if _, ok := seen[hour]; ok {
			continue
		}
		seen[hour] = struct{}{}
		hours = append(hours, hour)
	}
	slices.Sort(hours)
	return hours, nil
}

func (s *Scheduler) executeTask(task ScheduledTask, index int) error {
	switch task.normalizedType() {
	case "program":
		return s.executeProgramTask(task, index)
	case "ai":
		return s.executeAITask(task, index)
	default:
		return fmt.Errorf("unsupported type %q", task.Type)
	}
}

func (s *Scheduler) executeProgramTask(task ScheduledTask, index int) error {
	command := strings.TrimSpace(task.Command)
	if command == "" {
		return fmt.Errorf("program task requires command")
	}

	resolvedCommand := resolveScheduledCommand(command)
	cmd := exec.Command(resolvedCommand, task.Args...)
	if workDir := strings.TrimSpace(task.WorkDir); workDir != "" {
		cmd.Dir = workDir
	}

	log.Printf("[scheduler] [EXEC] %s (%s %v)", task.displayName(index), resolvedCommand, task.Args)
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		log.Printf("[scheduler] [OUT] %s\n%s", task.displayName(index), strings.TrimRight(string(output), "\n"))
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) executeAITask(task ScheduledTask, index int) error {
	prompt := strings.TrimSpace(task.Prompt)
	if prompt == "" {
		return fmt.Errorf("ai task requires prompt")
	}

	log.Printf("[scheduler] [QUEUE] %s -> dispatch", task.displayName(index))
	s.dispatchCh <- DispatchRequest{
		Type:    "ai",
		Message: prompt,
	}
	return nil
}

func resolveScheduledCommand(command string) string {
	if runtime.GOOS != "windows" {
		return command
	}
	if command != "python" && command != "python3" {
		return command
	}
	if _, err := exec.LookPath(command); err == nil {
		return command
	}
	if _, err := exec.LookPath("py"); err == nil {
		return "py"
	}
	return command
}

func ResolveScheduledCommand(command string) string {
	return resolveScheduledCommand(command)
}
