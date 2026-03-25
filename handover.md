当前目标已从“调查 Feishu 发送者昵称来源”收敛为“接受通过 `im/v1/chats/:chat_id/members` 获取 sender `name`，并承认它在当前群里仍可能不是用户实际看到的昵称，因为连飞书官方机器人也复现了这个现象”。
已完成的关键实现是：Feishu 入站归档的 `SenderName` 不再使用 `contact/v3/users/:user_id`，而是改成调用 `im/v1/chats/:chat_id/members` 分页查找当前 `chat_id + sender_open_id` 对应成员名，并把缓存键切成 `chat_id + open_id`。 
日志与缓存也已同步落地：新增 `logs/feishu_chat_members_raw.jsonl` 记录每次群成员接口原始响应，sqlite 中新增 `feishu_chat_user_cache(chat_id, open_id, display_name, refreshed_at_unix)`，TTL 为 24 小时，失败时允许回退到 7 天 stale 缓存。 
当前代码状态是：`internal/gateway/feishu.go` 与 `internal/gateway/state.go` 已更新，期间修复过一次 `CREATE TABLE feishu_chat_user_cache` 少逗号导致的 `init db: SQL logic error: near \"(\": syntax error (1)`，并且 `gofmt` 与 `go build ./...` 已再次通过。 
下一步最合理的动作是继续用真实群消息观察 `feishu_chat_members_raw.jsonl` 与归档输出，如果确认该接口长期只能返回错误名字，就把这件事正式记录为飞书侧限制/疑似 bug，并避免再为“真实群昵称”追加高风险推断逻辑。 
