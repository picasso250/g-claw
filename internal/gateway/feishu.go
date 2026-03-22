package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

type FeishuConfig struct {
	Enable         bool
	AppID          string
	AppSecret      string
	AllowedOpenIDs []string
	AllowedChatIDs []string
}

type feishuTextContent struct {
	Text string `json:"text"`
}

func StartFeishuLongConn(config FeishuConfig, db *sql.DB, dispatchCh chan struct{}) error {
	if !config.Enable {
		return nil
	}
	if strings.TrimSpace(config.AppID) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return fmt.Errorf("FEISHU_APP_ID or FEISHU_APP_SECRET is empty")
	}

	handler := larkdispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return handleFeishuEvent(config, db, dispatchCh, event)
		})

	client := larkws.NewClient(
		config.AppID,
		config.AppSecret,
		larkws.WithEventHandler(handler),
	)

	log.Printf("[feishu] [*] Starting long connection client")
	return client.Start(context.Background())
}

func handleFeishuEvent(config FeishuConfig, db *sql.DB, dispatchCh chan struct{}, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	messageID := derefString(message.MessageId)
	messageType := derefString(message.MessageType)
	chatID := derefString(message.ChatId)
	chatType := derefString(message.ChatType)
	senderOpenID := ""
	senderType := ""
	if sender != nil {
		senderType = derefString(sender.SenderType)
		if sender.SenderId != nil {
			senderOpenID = derefString(sender.SenderId.OpenId)
		}
	}

	if messageType != "text" {
		log.Printf("[feishu] [*] Ignoring non-text message %s of type %s", messageID, messageType)
		return nil
	}

	if !isFeishuMessageAllowed(config, senderOpenID, chatID) {
		log.Printf("[feishu] [*] Ignoring message %s from sender %s in chat %s", messageID, senderOpenID, chatID)
		return nil
	}

	if _, err := LookupMessageState(db, "feishu", messageID); err == nil {
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("lookup state for %s: %w", messageID, err)
	}

	body, err := extractFeishuText(derefString(message.Content))
	if err != nil {
		return fmt.Errorf("parse message content for %s: %w", messageID, err)
	}

	msgTime := parseFeishuTime(derefString(message.CreateTime))
	archiveContent := BuildMessageArchiveContent(ArchivedMessage{
		Source:         "feishu",
		SenderName:     senderType,
		SenderID:       senderOpenID,
		ConversationID: chatID,
		Subject:        chatType,
		MessageID:      messageID,
		Date:           msgTime,
		Body:           body,
	})

	archiveFile, err := SavePendingMessage("feishu", messageID, senderOpenID, archiveContent, time.Now())
	if err != nil {
		return fmt.Errorf("save pending for %s: %w", messageID, err)
	}

	if err := SaveMessageState(db, "feishu", messageID, senderOpenID, chatType, StateProcessed); err != nil {
		return fmt.Errorf("save state for %s: %w", messageID, err)
	}

	log.Printf("[feishu] [*] Saved message %s to %s", messageID, archiveFile)
	select {
	case dispatchCh <- struct{}{}:
	default:
	}
	return nil
}

func extractFeishuText(raw string) (string, error) {
	var content feishuTextContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		return "", err
	}
	return strings.TrimSpace(content.Text), nil
}

func parseFeishuTime(values ...string) time.Time {
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if ts, err := time.Parse(time.RFC3339, raw); err == nil {
			return ts
		}

		if unixMillis, err := parseUnixMillis(raw); err == nil {
			return unixMillis
		}
	}
	return time.Now()
}

func parseUnixMillis(raw string) (time.Time, error) {
	var millis int64
	if _, err := fmt.Sscanf(raw, "%d", &millis); err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(millis), nil
}

func isAllowed(allowlist []string, value string) bool {
	if len(allowlist) == 0 {
		return false
	}
	value = strings.TrimSpace(value)
	for _, allowed := range allowlist {
		if value == allowed {
			return true
		}
	}
	return false
}

func isFeishuMessageAllowed(config FeishuConfig, senderOpenID, chatID string) bool {
	if len(config.AllowedOpenIDs) == 0 && len(config.AllowedChatIDs) == 0 {
		return true
	}
	if isAllowed(config.AllowedOpenIDs, senderOpenID) {
		return true
	}
	if isAllowed(config.AllowedChatIDs, chatID) {
		return true
	}
	return false
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
