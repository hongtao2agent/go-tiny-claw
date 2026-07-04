package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/yourname/go-tiny-claw/internal/engine"
)

const receiveIDTypeChatID = "chat_id"

type Config struct {
	AppID       string
	AppSecret   string
	VerifyToken string
	EncryptKey  string
}

func ConfigFromEnv() Config {
	return Config{
		AppID:       os.Getenv("FEISHU_APP_ID"),
		AppSecret:   os.Getenv("FEISHU_APP_SECRET"),
		VerifyToken: os.Getenv("FEISHU_VERIFY_TOKEN"),
		EncryptKey:  os.Getenv("FEISHU_ENCRYPT_KEY"),
	}
}

type FeishuBot struct {
	client *lark.Client
	engine *engine.AgentEngine
	config Config
}

func NewFeishuBotFromEnv(eng *engine.AgentEngine) (*FeishuBot, error) {
	return NewFeishuBot(eng, ConfigFromEnv())
}

func NewFeishuBot(eng *engine.AgentEngine, cfg Config) (*FeishuBot, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("请设置 FEISHU_APP_ID 和 FEISHU_APP_SECRET")
	}
	if cfg.VerifyToken == "" {
		return nil, fmt.Errorf("请设置 FEISHU_VERIFY_TOKEN")
	}

	return &FeishuBot{
		client: lark.NewClient(cfg.AppID, cfg.AppSecret),
		engine: eng,
		config: cfg,
	}, nil
}

func (b *FeishuBot) GetEventDispatcher() *dispatcher.EventDispatcher {
	return dispatcher.NewEventDispatcher(b.config.VerifyToken, b.config.EncryptKey).
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return b.handleMessageReceive(ctx, event)
		})
}

func (b *FeishuBot) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	if message.ChatId == nil || *message.ChatId == "" {
		return nil
	}

	chatID := *message.ChatId
	messageType := valueOf(message.MessageType)
	if messageType != "" && messageType != larkim.MsgTypeText {
		reporter := NewFeishuReporter(b.client, chatID)
		reporter.OnMessage(ctx, "当前仅支持处理文本消息。")
		return nil
	}

	prompt := parseTextContent(valueOf(message.Content))
	if prompt == "" {
		return nil
	}

	log.Printf("[Feishu] 收到会话 %s 消息: %s\n", chatID, prompt)

	go b.handleAgentRun(chatID, prompt)
	return nil
}

func (b *FeishuBot) handleAgentRun(chatID string, prompt string) {
	reporter := NewFeishuReporter(b.client, chatID)
	reporter.OnMessage(context.Background(), "收到任务，go-tiny-claw 开始执行。")

	err := b.engine.RunWithReporter(context.Background(), prompt, reporter)
	if err != nil {
		reporter.OnMessage(context.Background(), fmt.Sprintf("❌ Agent 运行崩溃: %v", err))
	}
}

type FeishuReporter struct {
	client *lark.Client
	chatID string
}

func NewFeishuReporter(client *lark.Client, chatID string) *FeishuReporter {
	return &FeishuReporter{
		client: client,
		chatID: chatID,
	}
}

func (r *FeishuReporter) OnThinking(ctx context.Context) {
	r.sendMsg(ctx, "🤔 模型正在慢思考 (Thinking)...")
}

func (r *FeishuReporter) OnToolCall(ctx context.Context, toolName string, args string) {
	r.sendMsg(ctx, fmt.Sprintf("🛠️ 正在执行工具：`%s`\n参数：`%s`", toolName, args))
}

func (r *FeishuReporter) OnToolResult(ctx context.Context, toolName string, result string, isError bool) {
	if isError {
		r.sendMsg(ctx, fmt.Sprintf("⚠️ 执行报错 (%s)：\n%s", toolName, result))
		return
	}

	r.sendMsg(ctx, fmt.Sprintf("✅ 执行成功 (%s)", toolName))
}

func (r *FeishuReporter) OnMessage(ctx context.Context, content string) {
	r.sendMsg(ctx, content)
}

func (r *FeishuReporter) sendMsg(ctx context.Context, text string) {
	if r.client == nil || r.chatID == "" {
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	textContent := map[string]string{"text": text}
	contentBytes, err := json.Marshal(textContent)
	if err != nil {
		log.Printf("[Feishu] 编码消息失败: %v\n", err)
		return
	}

	msgReq := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDTypeChatID).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(r.chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(contentBytes)).
			Build()).
		Build()

	resp, err := r.client.Im.Message.Create(reqCtx, msgReq)
	if err != nil {
		log.Printf("[Feishu] 发送消息失败: %v\n", err)
		return
	}
	if !resp.Success() {
		log.Printf("[Feishu] 发送消息失败: code=%d msg=%s\n", resp.Code, resp.Msg)
	}
}

func parseTextContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err == nil && payload.Text != "" {
		return strings.TrimSpace(payload.Text)
	}

	return content
}

func valueOf(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var _ engine.Reporter = (*FeishuReporter)(nil)
