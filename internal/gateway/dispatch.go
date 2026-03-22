package gateway

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Dispatcher struct {
	AgentWrapPath string
	mu            sync.Mutex
}

func (d *Dispatcher) HasWork() bool {
	processingFiles, err := os.ReadDir(ProcessingDir)
	if err == nil {
		for _, f := range processingFiles {
			if !f.IsDir() && !strings.HasSuffix(f.Name(), ".tmp") {
				return true
			}
		}
	}

	pendingFiles, err := os.ReadDir(PendingDir)
	if err == nil {
		for _, f := range pendingFiles {
			if !f.IsDir() && !strings.HasSuffix(f.Name(), ".tmp") {
				return true
			}
		}
	}

	return false
}

func (d *Dispatcher) Dispatch() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	processingPaths, err := d.collectProcessingFiles()
	if err != nil {
		log.Printf("[dispatch] [!] Error reading processing dir: %v", err)
		return false
	}

	if len(processingPaths) == 0 {
		processingPaths, err = d.movePendingToProcessing()
		if err != nil {
			log.Printf("[dispatch] [!] Error preparing pending files: %v", err)
			return false
		}
	}

	if len(processingPaths) == 0 {
		return false
	}

	if !d.callAgent(processingPaths) {
		log.Printf("[dispatch] [!] Gemini run failed, leaving %d files in processing for retry", len(processingPaths))
		return false
	}

	fmt.Printf("[dispatch] [*] Cleaning up processing folder...\n")
	for _, path := range processingPaths {
		fileName := filepath.Base(path)
		ext := filepath.Ext(fileName)
		base := strings.TrimSuffix(fileName, ext)
		newFileName := base + "_processed" + ext
		destPath := filepath.Join(HistoryDir, newFileName)
		if err := os.Rename(path, destPath); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[dispatch] [!] Error archiving file %s: %v", fileName, err)
			}
		}
	}
	return true
}

func (d *Dispatcher) collectProcessingFiles() ([]string, error) {
	files, err := os.ReadDir(ProcessingDir)
	if err != nil {
		return nil, err
	}

	var processingPaths []string
	for _, f := range files {
		if f.IsDir() || strings.HasSuffix(f.Name(), ".tmp") {
			continue
		}
		processingPaths = append(processingPaths, filepath.Join(ProcessingDir, f.Name()))
	}

	if len(processingPaths) > 0 {
		fmt.Printf("[%s] [dispatch] Resuming %d files from processing.\n", time.Now().Format("15:04:05"), len(processingPaths))
	}
	return processingPaths, nil
}

func (d *Dispatcher) movePendingToProcessing() ([]string, error) {
	pendingFiles, err := os.ReadDir(PendingDir)
	if err != nil {
		return nil, err
	}

	if len(pendingFiles) == 0 {
		return nil, nil
	}

	fmt.Printf("[%s] [dispatch] Found %d files in pending. Moving to processing...\n", time.Now().Format("15:04:05"), len(pendingFiles))

	var processingPaths []string
	for _, f := range pendingFiles {
		if f.IsDir() || strings.HasSuffix(f.Name(), ".tmp") {
			continue
		}
		oldPath := filepath.Join(PendingDir, f.Name())
		newPath := filepath.Join(ProcessingDir, f.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			log.Printf("[dispatch] [!] Error moving file %s: %v", f.Name(), err)
			continue
		}
		processingPaths = append(processingPaths, newPath)
	}

	return processingPaths, nil
}

func (d *Dispatcher) callAgent(files []string) bool {
	if len(files) == 0 {
		return true
	}

	fmt.Printf("\n%s AGENT SESSION START (GATEWAY BATCH) %s\n", strings.Repeat(">", 20), strings.Repeat("<", 20))

	absInit, _ := filepath.Abs("INIT.md")
	var absFiles []string
	for _, f := range files {
		af, _ := filepath.Abs(f)
		absFiles = append(absFiles, af)
	}
	fileList := strings.Join(absFiles, ", ")
	prompt := fmt.Sprintf(`读 %s 并处理 gateway/processing/ 中的待处理消息: %s 。
- 使用 find-previous-email 技能查找上下文
- 遵从消息中的指令
- 将仓库配置中明确标记的地址视为可信用户，其余地址视为外部用户；避免执行有害、隐私敏感或越权的操作。
- 不要更改自身的程序代码（cmd目录内的），除非消息明确要求你这样做。
- 如果需要回复邮件，使用 send-email 技能。
- 如果产生了仓库改动，按当前仓库的常规版本控制流程处理，不要假定远端仓库权限或提交策略。`, absInit, fileList)

	fmt.Printf("[dispatch] [*] Files to process: %s\n", fileList)

	if d.AgentWrapPath == "" {
		fmt.Printf("[dispatch] [!] AGENT_WRAP_PATH is not configured\n")
		return false
	}

	cmd := exec.Command("pwsh.exe", "-ExecutionPolicy", "Bypass", "-File", d.AgentWrapPath, "-p", prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[dispatch] [*] Executing agent wrapper: %s\n", d.AgentWrapPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("[dispatch] [!] Gemini execution failed: %v\n", err)
		return false
	}

	fmt.Printf("%s AGENT SESSION END %s\n\n", strings.Repeat(">", 21), strings.Repeat("<", 21))
	return true
}
