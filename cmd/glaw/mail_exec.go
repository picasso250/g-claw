package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	multipart "mime/multipart"
	stdmail "net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gatewaypkg "glaw/internal/gateway"
)

const mailExecTimeout = 5 * time.Minute

func normalizeReplySubject(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "Execution Result"
	}
	if strings.HasPrefix(strings.ToLower(subject), "re:") {
		return subject
	}
	return "Re: " + subject
}

func subjectMatchesKeyword(subject, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	return strings.Contains(strings.ToLower(subject), strings.ToLower(keyword))
}

func sanitizeFilenameToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(":", "-", "/", "-", "\\", "-", " ", "_").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func deriveSMTPServer(imapServer string) string {
	imapServer = strings.TrimSpace(imapServer)
	if imapServer == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(imapServer), "imap.") {
		return "smtp." + imapServer[5:]
	}
	return imapServer
}

func resolveSMTPConfig(config Config) (string, int, error) {
	server := strings.TrimSpace(config.MailSmtpServer)
	if server == "" {
		server = deriveSMTPServer(config.MailImapServer)
	}
	if server == "" {
		return "", 0, fmt.Errorf("MAIL_SMTP_SERVER is empty")
	}
	port := config.MailSmtpPort
	if port <= 0 {
		port = 465
	}
	return server, port, nil
}

func textprotoMIMEHeader(values map[string]string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader, len(values))
	for key, value := range values {
		header.Set(key, value)
	}
	return header
}

func writeBase64Lines(w io.Writer, src []byte) error {
	encoded := base64.StdEncoding.EncodeToString(src)
	for len(encoded) > 76 {
		if _, err := io.WriteString(w, encoded[:76]+"\r\n"); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	_, err := io.WriteString(w, encoded+"\r\n")
	return err
}

func buildMailWithAttachments(fromAddr, toAddr, subject, body string, attachmentPaths []string) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if _, err := fmt.Fprintf(&buf, "From: %s\r\n", (&stdmail.Address{Address: fromAddr}).String()); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(&buf, "To: %s\r\n", (&stdmail.Address{Address: toAddr}).String()); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject)); err != nil {
		return nil, err
	}
	if _, err := io.WriteString(&buf, "MIME-Version: 1.0\r\n"); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", writer.Boundary()); err != nil {
		return nil, err
	}

	bodyPart, err := writer.CreatePart(textprotoMIMEHeader(map[string]string{
		"Content-Type":              `text/plain; charset="utf-8"`,
		"Content-Transfer-Encoding": "8bit",
	}))
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(bodyPart, body); err != nil {
		return nil, err
	}

	for _, attachmentPath := range attachmentPaths {
		data, err := os.ReadFile(attachmentPath)
		if err != nil {
			return nil, err
		}
		fileName := filepath.Base(attachmentPath)
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		part, err := writer.CreatePart(textprotoMIMEHeader(map[string]string{
			"Content-Type":              fmt.Sprintf(`%s; name="%s"`, contentType, fileName),
			"Content-Disposition":       fmt.Sprintf(`attachment; filename="%s"`, fileName),
			"Content-Transfer-Encoding": "base64",
		}))
		if err != nil {
			return nil, err
		}
		if err := writeBase64Lines(part, data); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sendMailWithAttachments(config Config, toAddr, subject, body string, attachmentPaths []string) error {
	server, port, err := resolveSMTPConfig(config)
	if err != nil {
		return err
	}
	log.Printf("[mail_exec] [*] Sending reply mail to=%s smtp=%s:%d attachments=%d", toAddr, server, port, len(attachmentPaths))
	payload, err := buildMailWithAttachments(config.MailUser, toAddr, subject, body, attachmentPaths)
	if err != nil {
		return err
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", server, port), &tls.Config{ServerName: server})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, server)
	if err != nil {
		return err
	}
	defer client.Quit()

	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(smtp.PlainAuth("", config.MailUser, config.MailPass, server)); err != nil {
			return err
		}
	}
	if err := client.Mail(config.MailUser); err != nil {
		return err
	}
	if err := client.Rcpt(toAddr); err != nil {
		return err
	}
	wc, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(payload); err != nil {
		wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	log.Printf("[mail_exec] [*] Reply mail sent to=%s", toAddr)
	return nil
}

func buildExecutionResultPaths(baseDir string, uid uint32, sender string, now time.Time) (string, string) {
	base := fmt.Sprintf(
		"mail_exec_%s_%s_%d",
		sanitizeFilenameToken(sender),
		now.UTC().Format("2006-01-02T15-04-05Z"),
		uid,
	)
	return filepath.Join(baseDir, base+".stdout.txt"), filepath.Join(baseDir, base+".stderr.txt")
}

func buildExecutionZipPath(baseDir string, uid uint32, sender string, now time.Time) string {
	base := fmt.Sprintf(
		"mail_exec_%s_%s_%d",
		sanitizeFilenameToken(sender),
		now.UTC().Format("2006-01-02T15-04-05Z"),
		uid,
	)
	return filepath.Join(baseDir, base+".zip")
}

func absPath(path string) (string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func selectExecutableAttachment(savedNames []string) (string, []string, error) {
	if len(savedNames) == 0 {
		return "", nil, fmt.Errorf("expected at least 1 attachment, got 0")
	}

	var scriptPath string
	var resourceAttachmentPaths []string
	for _, savedName := range savedNames {
		fullPath, err := absPath(filepath.Join(gatewaypkg.MediaDir, savedName))
		if err != nil {
			return "", nil, fmt.Errorf("resolve attachment path %q: %w", savedName, err)
		}

		ext := strings.ToLower(filepath.Ext(fullPath))
		if ext != ".ps1" && ext != ".py" {
			resourceAttachmentPaths = append(resourceAttachmentPaths, fullPath)
			continue
		}
		if scriptPath != "" {
			return "", nil, fmt.Errorf("expected exactly 1 executable attachment, got multiple")
		}
		scriptPath = fullPath
	}
	if scriptPath == "" {
		return "", nil, fmt.Errorf("expected exactly 1 executable attachment, got 0")
	}
	return scriptPath, resourceAttachmentPaths, nil
}

func copyFile(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, 0644)
}

func executeMailAttachment(scriptPath string, resourceAttachmentPaths []string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mailExecTimeout)
	defer cancel()

	var cmd *exec.Cmd
	runner := ""
	switch strings.ToLower(filepath.Ext(scriptPath)) {
	case ".ps1":
		args := append([]string{"-File", scriptPath}, resourceAttachmentPaths...)
		cmd = exec.CommandContext(ctx, "pwsh", args...)
		runner = "pwsh"
	case ".py":
		args := append([]string{scriptPath}, resourceAttachmentPaths...)
		cmd = exec.CommandContext(ctx, "python", args...)
		runner = "python"
	default:
		return "", "", fmt.Errorf("unsupported attachment type %q", filepath.Ext(scriptPath))
	}

	cmd.Dir = filepath.Dir(scriptPath)
	log.Printf("[mail_exec] [*] Executing attachment via %s: %s resource_attachments=%v", runner, scriptPath, resourceAttachmentPaths)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		if stderrBuf.Len() > 0 {
			stderrBuf.WriteString("\n")
		}
		stderrBuf.WriteString("TIMEOUT")
		log.Printf("[mail_exec] [!] Attachment timed out after %s: %s", mailExecTimeout, scriptPath)
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("attachment execution timed out after %s", mailExecTimeout)
	}
	if err != nil {
		log.Printf("[mail_exec] [!] Attachment execution failed: %s err=%v", scriptPath, err)
	} else {
		log.Printf("[mail_exec] [*] Attachment execution finished: %s", scriptPath)
	}
	return stdoutBuf.String(), stderrBuf.String(), err
}

func collectReplyAttachmentPaths(body, baseDir string) []string {
	var paths []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(body, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, "#") {
			continue
		}

		resolved := candidate
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(baseDir, resolved)
		}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			continue
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			continue
		}
		key := strings.ToLower(resolved)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, resolved)
	}
	return paths
}

func addFileToZip(zw *zip.Writer, diskPath, archiveName string) error {
	data, err := os.ReadFile(diskPath)
	if err != nil {
		return err
	}
	w, err := zw.Create(archiveName)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func buildExecutionResultZip(zipPath string, stdoutPath string, stderrPath string, extraPaths []string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	if err := addFileToZip(zw, stdoutPath, filepath.Base(stdoutPath)); err != nil {
		return err
	}
	if err := addFileToZip(zw, stderrPath, filepath.Base(stderrPath)); err != nil {
		return err
	}
	for _, extraPath := range extraPaths {
		if err := addFileToZip(zw, extraPath, filepath.Base(extraPath)); err != nil {
			return err
		}
	}
	return nil
}

func processExecutionMail(config *Config, uid uint32, sender, subject string, archivedEmail *gatewaypkg.ArchivedEmail) error {
	tempDir, err := os.MkdirTemp("", "glaw-mail-exec-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	log.Printf("[mail_exec] [*] Created temp dir for uid=%d: %s", uid, tempDir)

	stdoutPath, stderrPath := buildExecutionResultPaths(tempDir, uid, sender, time.Now())
	zipPath := buildExecutionZipPath(tempDir, uid, sender, time.Now())
	stdout := ""
	stderr := ""

	scriptPath, resourceAttachmentPaths, err := selectExecutableAttachment(archivedEmail.Attachments)
	if err != nil {
		log.Printf("[mail_exec] [!] Attachment selection failed for uid=%d: %v", uid, err)
		stderr = err.Error() + "\n"
	} else {
		log.Printf("[mail_exec] [*] Selected executable attachment for uid=%d: %s", uid, scriptPath)
		tempScriptPath := filepath.Join(tempDir, filepath.Base(scriptPath))
		if err := copyFile(scriptPath, tempScriptPath); err != nil {
			log.Printf("[mail_exec] [!] Copy attachment failed for uid=%d: %v", uid, err)
			stderr = err.Error() + "\n"
		} else {
			log.Printf("[mail_exec] [*] Copied attachment for uid=%d to: %s", uid, tempScriptPath)
			stdout, stderr, err = executeMailAttachment(tempScriptPath, resourceAttachmentPaths)
		}
		if err != nil {
			if strings.TrimSpace(stderr) == "" {
				stderr = err.Error() + "\n"
			} else {
				stderr = strings.TrimRight(stderr, "\n") + "\n" + err.Error() + "\n"
			}
		}
	}

	if err := os.WriteFile(stdoutPath, []byte(stdout), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(stderrPath, []byte(stderr), 0644); err != nil {
		return err
	}
	log.Printf("[mail_exec] [*] Wrote execution result files for uid=%d stdout=%s stderr=%s", uid, stdoutPath, stderrPath)
	replyAttachmentPaths := collectReplyAttachmentPaths(archivedEmail.Body, tempDir)
	if len(replyAttachmentPaths) > 0 {
		log.Printf("[mail_exec] [*] Collected %d reply attachment path(s) from body for uid=%d", len(replyAttachmentPaths), uid)
	}
	if err := buildExecutionResultZip(zipPath, stdoutPath, stderrPath, replyAttachmentPaths); err != nil {
		return err
	}
	log.Printf("[mail_exec] [*] Built execution result zip for uid=%d: %s", uid, zipPath)

	return sendMailWithAttachments(
		*config,
		sender,
		normalizeReplySubject(subject),
		"Attached is a zip containing stdout.txt, stderr.txt, and any existing file paths listed line-by-line in the mail body.\n",
		[]string{zipPath},
	)
}
