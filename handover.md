当前目标已经从“只收敛 prompt”升级为“统一成 `glaw.exe` 主程序入口”，包括 `serve` 和 `feishu list-messages` 两类子命令。 
`cmd/gateway/main.go` 与 `cmd/feishu-list-messages/main.go` 已合并进 `cmd/glaw/main.go`，模块名也已从 `g-claw` 改成 `glaw`。 
`internal/gateway/dispatch.go` 中飞书 prompt 已不再要求运行 ad hoc 的 `pwsh + go run`，而是改为运行 `~/bin/glaw.exe feishu list-messages ...`，并保留尾部跟随时的二次检查逻辑。 
`cmd/debug-agent-cmd` 已删除，README 与 `dev.ps1` 也已更新为 `go run ./cmd/glaw serve`、`go build -o ~/bin/glaw.exe ./cmd/glaw` 和 `~/bin/glaw.exe serve` 这一套新用法。 
这一轮还没有重新跑 `go build ./...`，下一步应先编译验证并清理仓库里残留的旧路径或旧命令引用。 
