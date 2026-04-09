package robot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/gateway"
	"github.com/LittleSongxx/TinyClaw/i18n"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/metrics"
	"github.com/LittleSongxx/TinyClaw/node"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkapplication "github.com/larksuite/oapi-sdk-go/v3/service/application/v6"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

type MessageText struct {
	Text string `json:"text"`
}

var (
	cli           *larkws.Client
	BotName       string
	LarkBotClient *lark.Client
)

type LarkRobot struct {
	Message *larkim.P2MessageReceiveV1
	Robot   *RobotInfo
	Client  *lark.Client

	Command      string
	Prompt       string
	BotName      string
	ImageContent []byte
	AudioContent []byte
	UserName     string
}

func StartLarkRobot(ctx context.Context) {
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(LarkMessageHandler).
		OnP2CardActionTrigger(LarkApprovalCardHandler)

	cli = larkws.NewClient(conf.BaseConfInfo.LarkAPPID, conf.BaseConfInfo.LarkAppSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
		larkws.WithLogger(logger.Logger),
	)

	LarkBotClient = lark.NewClient(conf.BaseConfInfo.LarkAPPID, conf.BaseConfInfo.LarkAppSecret,
		lark.WithHttpClient(utils.GetRobotProxyClient()))

	// get bot name
	resp, err := LarkBotClient.Application.Application.Get(ctx, larkapplication.NewGetApplicationReqBuilder().
		AppId(conf.BaseConfInfo.LarkAPPID).Lang("zh_cn").Build())
	if err != nil || !resp.Success() {
		logger.ErrorCtx(ctx, "get robot name error", "error", err, "resp", resp)
		return
	}
	BotName = larkcore.StringValue(resp.Data.App.AppName)
	logger.Info("LarkBot Info", "username", BotName)

	err = cli.Start(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "start larkbot fail", "err", err)
	}
}

func NewLarkRobot(message *larkim.P2MessageReceiveV1) *LarkRobot {
	metrics.AppRequestCount.WithLabelValues("lark").Inc()
	return &LarkRobot{
		Message: message,
		Client:  LarkBotClient,
		BotName: BotName,
	}
}

func LarkMessageHandler(ctx context.Context, message *larkim.P2MessageReceiveV1) error {
	l := NewLarkRobot(message)
	l.Robot = NewRobot(WithRobot(l), WithContext(ctx))
	groupID := ""
	if larkcore.StringValue(message.Event.Message.ChatType) == "group" {
		groupID = larkcore.StringValue(message.Event.Message.ChatId)
	}
	_, state, err := gateway.DefaultService().BeginInbound(ctx, gateway.InboundMessage{
		Channel:   "lark",
		AccountID: conf.BaseConfInfo.LarkAPPID,
		PeerID:    larkcore.StringValue(message.Event.Sender.SenderId.UserId),
		GroupID:   groupID,
		MessageID: larkcore.StringValue(message.Event.Message.MessageId),
		Kind:      "",
		Metadata: map[string]string{
			"source": "lark",
		},
	})
	if err == nil {
		l.Robot.ApplyContextState(state)
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.ErrorCtx(ctx, "exec panic", "err", err, "stack", string(debug.Stack()))
			}
		}()
		userInfo, err := LarkBotClient.Contact.V3.User.Get(l.Robot.Ctx, larkcontact.NewGetUserReqBuilder().
			UserId(*message.Event.Sender.SenderId.UserId).UserIdType("user_id").Build())
		if err != nil || userInfo.Code != 0 {
			logger.ErrorCtx(ctx, "get user info error", "err", err, "user_info", userInfo)
		} else {
			l.UserName = *userInfo.Data.User.Name
		}

		l.Robot.Exec()
	}()

	return nil
}

func (l *LarkRobot) checkValid() bool {
	chatId, msgId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()

	// group need to at bot
	atBot, err := l.GetMessageContent()
	if err != nil {
		logger.ErrorCtx(l.Robot.Ctx, "get message content error", "err", err)
		l.Robot.SendMsg(chatId, err.Error(), msgId, "", nil)
		return false
	}
	if larkcore.StringValue(l.Message.Event.Message.ChatType) == "group" {
		if !atBot {
			logger.Warn("no at bot")
			return false
		}
	}

	return true
}

func (l *LarkRobot) getMsgContent() string {
	return l.Command
}

func (l *LarkRobot) requestLLM(content string) {
	if !strings.Contains(content, "/") && !strings.Contains(content, "$") && l.Prompt == "" {
		l.Prompt = content
	}
	l.Robot.ExecCmd(content, l.sendChatMessage, nil, nil)
}

func (l *LarkRobot) sendImg() {
	l.Robot.TalkingPreCheck(func() {
		chatId, msgId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()

		prompt := strings.TrimSpace(l.Prompt)
		if prompt == "" {
			logger.Warn("prompt is empty")
			l.Robot.SendMsg(chatId, i18n.GetMessage("photo_empty_content", nil), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		lastImageContent := l.ImageContent
		var err error
		if len(lastImageContent) == 0 && strings.Contains(l.Command, "edit_photo") {
			lastImageContent, err = l.Robot.GetLastImageContent()
			if err != nil {
				logger.Warn("get last image record fail", "err", err)
			}
		}

		imageContent, totalToken, err := l.Robot.CreatePhoto(prompt, lastImageContent)
		if err != nil {
			logger.Warn("generate image fail", "err", err)
			l.Robot.SendMsg(chatId, err.Error(), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		err = l.sendMedia(imageContent, utils.DetectImageFormat(imageContent), "image")
		if err != nil {
			logger.Warn("send image fail", "err", err)
			l.Robot.SendMsg(chatId, err.Error(), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		l.Robot.saveRecord(imageContent, lastImageContent, param.ImageRecordType, totalToken)
	})
}

func (l *LarkRobot) sendMedia(media []byte, contentType, sType string) error {
	postContent := make([]larkim.MessagePostElement, 0)
	chatId, _, _ := l.Robot.GetChatIdAndMsgIdAndUserID()
	if sType == "image" {
		imageKey, err := l.getImageInfo(media)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "create image fail", "err", err)
			return err
		}

		postContent = append(postContent, &larkim.MessagePostImage{
			ImageKey: imageKey,
		})

	} else {
		fileKey, err := l.getVideoInfo(media)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "get image info fail", "err", err)
			return err
		}

		postContent = append(postContent, &larkim.MessagePostMedia{
			FileKey: fileKey,
		})
	}

	msgContent, _ := larkim.NewMessagePost().ZhCn(larkim.NewMessagePostContent().AppendContent(postContent).Build()).Build()
	res, err := l.Client.Im.Message.Create(l.Robot.Ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(larkim.MsgTypePost).
			ReceiveId(chatId).
			Content(msgContent).
			Build()).
		Build())
	if err != nil || !res.Success() {
		logger.Warn("send message fail", "err", err, "resp", res)
		return err
	}

	return nil
}

func (l *LarkRobot) sendVideo() {
	// 检查 prompt
	l.Robot.TalkingPreCheck(func() {
		chatId, msgId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()

		prompt := strings.TrimSpace(l.Prompt)
		if prompt == "" {
			logger.Warn("prompt is empty")
			l.Robot.SendMsg(chatId, i18n.GetMessage("video_empty_content", nil), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		videoContent, totalToken, err := l.Robot.CreateVideo(prompt, l.ImageContent)
		if err != nil {
			logger.Warn("generate video fail", "err", err)
			l.Robot.SendMsg(chatId, err.Error(), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		err = l.sendMedia(videoContent, utils.DetectVideoMimeType(videoContent), "video")
		if err != nil {
			logger.Warn("send video fail", "err", err)
			l.Robot.SendMsg(chatId, err.Error(), msgId, tgbotapi.ModeMarkdown, nil)
			return
		}

		l.Robot.saveRecord(videoContent, l.ImageContent, param.VideoRecordType, totalToken)
	})

}

func (l *LarkRobot) sendChatMessage() {
	l.Robot.TalkingPreCheck(func() {
		chatId, msgId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()
		l.Robot.SendMsg(chatId, i18n.GetMessage("thinking", nil), msgId, tgbotapi.ModeMarkdown, nil)
		if conf.RagConfInfo.Store != nil {
			l.executeChain()
		} else {
			l.executeLLM()
		}
	})

}

func (l *LarkRobot) executeChain() {
	messageChan := &MsgChan{
		NormalMessageChan: make(chan *param.MsgInfo),
	}
	go l.Robot.ExecChain(l.Prompt, messageChan)

	go l.Robot.HandleUpdate(messageChan, "opus")
}

func (l *LarkRobot) sendTextStream(messageChan *MsgChan) {
	var msg *param.MsgInfo
	chatId, messageId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()
	for msg = range messageChan.NormalMessageChan {
		if msg.Kind == "approval_request" {
			if err := l.sendApprovalPrompt(msg, messageId); err != nil {
				logger.Warn("send approval prompt fail", "err", err)
				l.Robot.SendMsg(chatId, msg.Content, messageId, tgbotapi.ModeMarkdown, nil)
			}
			continue
		}
		if len(msg.Content) == 0 {
			msg.Content = "get nothing from llm!"
		}

		if msg.MsgId == "" {
			msgId := l.Robot.SendMsg(chatId, msg.Content, messageId, tgbotapi.ModeMarkdown, nil)
			msg.MsgId = msgId
		} else {

			resp, err := l.Client.Im.Message.Update(l.Robot.Ctx, larkim.NewUpdateMessageReqBuilder().
				MessageId(msg.MsgId).
				Body(larkim.NewUpdateMessageReqBodyBuilder().
					MsgType(larkim.MsgTypePost).
					Content(GetMarkdownContent(msg.Content)).
					Build()).
				Build())
			if err != nil || !resp.Success() {
				logger.Warn("send message fail", "err", err, "resp", resp)
				continue
			}
		}
	}
}

func (l *LarkRobot) sendApprovalPrompt(msg *param.MsgInfo, replyToMessageID string) error {
	if msg == nil {
		return errors.New("approval message is empty")
	}
	approvalID := strings.TrimSpace(stringValueFromAny(msg.Payload["approval_id"]))
	summary := strings.TrimSpace(stringValueFromAny(msg.Payload["summary"]))
	sessionID := strings.TrimSpace(stringValueFromAny(msg.Payload["session_id"]))
	if approvalID == "" || summary == "" {
		return errors.New("approval payload is incomplete")
	}

	card := buildLarkApprovalCard(approvalID, sessionID, summary, approvalModesFromAny(msg.Payload["approval_modes"]), true, "", "")
	content, err := card.JSON()
	if err != nil {
		return err
	}
	resp, err := l.Client.Im.Message.Reply(l.Robot.Ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(replyToMessageID).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeInteractive).
			Content(content).
			Build()).
		Build())
	if err != nil || !resp.Success() {
		logger.Warn("send lark approval card fail", "err", err, "resp", resp)
		if err != nil {
			return err
		}
		return errors.New("send approval card failed")
	}
	return nil
}

func buildLarkApprovalCard(approvalID, sessionID, summary string, modes []string, pending bool, status string, detail string) *larkcard.MessageCard {
	template := larkcard.TemplateOrange
	title := "等待确认的设备操作"
	body := "**设备操作审批**\n"
	body += "- 审批编号: `" + approvalID + "`\n"
	body += "- 操作: " + summary + "\n"
	if pending {
		body += "- 状态: 待处理\n"
		body += "- 你也可以继续回复“确认”或“取消”作为文字兜底。"
	} else {
		if status == "" {
			status = "已处理"
		}
		body += "- 状态: " + status + "\n"
		if detail != "" {
			body += "- 结果: " + detail
		}
		switch status {
		case "已批准并执行":
			template = larkcard.TemplateGreen
			title = "设备操作已执行"
		case "已拒绝":
			template = larkcard.TemplateRed
			title = "设备操作已拒绝"
		default:
			template = larkcard.TemplateWathet
			title = "设备操作已更新"
		}
	}

	elements := []larkcard.MessageCardElement{
		larkcard.NewMessageCardMarkdown().Content(body).Build(),
	}
	if pending {
		actions := []larkcard.MessageCardActionElement{
			larkcard.NewMessageCardEmbedButton().
				Text(larkcard.NewMessageCardPlainText().Content("本次允许").Build()).
				Type(larkcard.MessageCardButtonTypePrimary).
				Value(map[string]interface{}{
					"tinyclaw_action": "approval",
					"approval_id":     approvalID,
					"mode":            string(node.ApprovalModeAllowOnce),
					"session_id":      sessionID,
				}).
				Build(),
		}
		if containsApprovalMode(modes, string(node.ApprovalModeAllowSession)) {
			actions = append(actions, larkcard.NewMessageCardEmbedButton().
				Text(larkcard.NewMessageCardPlainText().Content("本会话允许").Build()).
				Type(larkcard.MessageCardButtonTypeDefault).
				Value(map[string]interface{}{
					"tinyclaw_action": "approval",
					"approval_id":     approvalID,
					"mode":            string(node.ApprovalModeAllowSession),
					"session_id":      sessionID,
				}).
				Build())
		}
		actions = append(actions, larkcard.NewMessageCardEmbedButton().
			Text(larkcard.NewMessageCardPlainText().Content("拒绝").Build()).
			Type(larkcard.MessageCardButtonTypeDanger).
			Value(map[string]interface{}{
				"tinyclaw_action": "approval",
				"approval_id":     approvalID,
				"mode":            string(node.ApprovalModeReject),
				"session_id":      sessionID,
			}).
			Build())
		elements = append(elements, larkcard.NewMessageCardAction().
			Actions(actions).
			Layout(larkcard.MessageCardActionLayoutFlow.Ptr()).
			Build())
	}

	return larkcard.NewMessageCard().
		Header(larkcard.NewMessageCardHeader().
			Template(template).
			Title(larkcard.NewMessageCardPlainText().Content(title).Build()).
			Build()).
		Elements(elements).
		Build()
}

func LarkApprovalCardHandler(ctx context.Context, event *larkcallback.CardActionTriggerEvent) (*larkcallback.CardActionTriggerResponse, error) {
	if event == nil || event.Event == nil || event.Event.Action == nil {
		return &larkcallback.CardActionTriggerResponse{
			Toast: &larkcallback.Toast{
				Type:    "error",
				Content: "审批卡片数据无效",
			},
		}, nil
	}
	if stringValueFromAny(event.Event.Action.Value["tinyclaw_action"]) != "approval" {
		return &larkcallback.CardActionTriggerResponse{
			Toast: &larkcallback.Toast{
				Type:    "error",
				Content: "不支持的卡片动作",
			},
		}, nil
	}

	approvalID := strings.TrimSpace(stringValueFromAny(event.Event.Action.Value["approval_id"]))
	mode := node.ApprovalMode(strings.TrimSpace(stringValueFromAny(event.Event.Action.Value["mode"])))
	sessionID := strings.TrimSpace(stringValueFromAny(event.Event.Action.Value["session_id"]))
	userID := ""
	if event.Event.Operator != nil && event.Event.Operator.UserID != nil {
		userID = strings.TrimSpace(*event.Event.Operator.UserID)
	}
	if approvalID == "" || mode == "" {
		return &larkcallback.CardActionTriggerResponse{
			Toast: &larkcallback.Toast{
				Type:    "error",
				Content: "审批卡片缺少必要参数",
			},
		}, nil
	}

	result, err := gateway.DefaultService().DecideApproval(ctx, node.ApprovalDecision{
		CommandID: approvalID,
		SessionID: sessionID,
		UserID:    userID,
		Approved:  mode != node.ApprovalModeReject,
		Mode:      mode,
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		return &larkcallback.CardActionTriggerResponse{
			Toast: &larkcallback.Toast{
				Type:    "error",
				Content: "审批处理失败: " + err.Error(),
			},
		}, nil
	}

	approval, _ := result["approval"].(*node.ApprovalRequest)
	commandResult, _ := result["result"].(*node.NodeCommandResult)
	summary := ""
	sessionID = ""
	modes := []string(nil)
	if approval != nil {
		summary = approval.Summary
		sessionID = approval.SessionID
		for _, item := range approval.ApprovalModes {
			modes = append(modes, string(item))
		}
	}
	if summary == "" {
		summary = "设备操作"
	}

	status := "已更新"
	detail := ""
	toastContent := "审批已处理"
	switch mode {
	case node.ApprovalModeReject:
		status = "已拒绝"
		detail = "该操作已被拒绝，不会继续执行。"
		toastContent = "已拒绝设备操作"
	default:
		status = "已批准并执行"
		if commandResult != nil && commandResult.Error != "" {
			status = "执行失败"
			detail = friendlyNodeError(commandResult.Error)
			toastContent = "设备操作执行失败"
		} else if detail = approvalSuccessDetail(commandResult); detail == "" {
			detail = "设备操作已执行。"
			toastContent = "已批准并执行设备操作"
		} else {
			toastContent = "已批准并执行设备操作"
		}
	}

	card := buildLarkApprovalCard(approvalID, sessionID, summary, modes, false, status, detail)
	return &larkcallback.CardActionTriggerResponse{
		Toast: &larkcallback.Toast{
			Type:    "info",
			Content: toastContent,
		},
		Card: &larkcallback.Card{
			Type: "card_json",
			Data: card,
		},
	}, nil
}

func approvalModesFromAny(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		return values
	case []interface{}:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func containsApprovalMode(modes []string, target string) bool {
	for _, mode := range modes {
		if mode == target {
			return true
		}
	}
	return false
}

func stringValueFromAny(raw interface{}) string {
	value, _ := raw.(string)
	return value
}

func (l *LarkRobot) executeLLM() {
	messageChan := &MsgChan{
		NormalMessageChan: make(chan *param.MsgInfo),
	}
	go l.Robot.HandleUpdate(messageChan, "opus")

	go l.Robot.ExecLLM(l.Prompt, messageChan)

}

func GetMarkdownContent(content string) string {
	content = sanitizeLarkMarkdownContent(content)
	markdownMsg, _ := larkim.NewMessagePost().ZhCn(larkim.NewMessagePostContent().AppendContent(
		[]larkim.MessagePostElement{
			&MessagePostMarkdown{
				Text: content,
			},
		}).Build()).Build()

	return markdownMsg
}

func sanitizeLarkMarkdownContent(content string) string {
	cleaned := strings.TrimSpace(content)
	if cleaned == "" {
		return cleaned
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?s)\[MCP 返回的 Base64 图像已直接发送给用户\]`),
		regexp.MustCompile(`(?s)\{这里只是一个示例 Base64 图像字符串.*?\}`),
		regexp.MustCompile(`(?m)^\s*\].*$`),
	}
	for _, pattern := range patterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	lines := strings.Split(cleaned, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "Base64 图像") || strings.Contains(trimmed, "MCP 返回的图像数据") {
			continue
		}
		filtered = append(filtered, strings.TrimLeft(trimmed, "] "))
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

type MessagePostMarkdown struct {
	Text string `json:"text,omitempty"`
}

func (m *MessagePostMarkdown) Tag() string {
	return "md"
}

func (m *MessagePostMarkdown) IsPost() {
}

func (m *MessagePostMarkdown) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"tag":  "md",
		"text": m.Text,
	}
	return json.Marshal(data)
}

type MessagePostContent struct {
	Title   string                 `json:"title"`
	Content [][]MessagePostElement `json:"content"`
}

type MessagePostElement struct {
	Tag      string `json:"tag"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
	UserName string `json:"user_name"`
}

func (l *LarkRobot) GetMessageContent() (bool, error) {
	_, msgId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()
	msgType := larkcore.StringValue(l.Message.Event.Message.MessageType)
	botShowName := ""
	if msgType == larkim.MsgTypeText {
		textMsg := new(MessageText)
		err := json.Unmarshal([]byte(larkcore.StringValue(l.Message.Event.Message.Content)), textMsg)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "unmarshal text message error", "error", err)
			return false, err
		}
		l.Command, l.Prompt = ParseCommand(textMsg.Text)
		for _, at := range l.Message.Event.Message.Mentions {
			if larkcore.StringValue(at.Name) == l.BotName {
				botShowName = larkcore.StringValue(at.Key)
				break
			}
		}

		l.Prompt = strings.ReplaceAll(l.Prompt, "@"+botShowName, "")
		for _, at := range l.Message.Event.Message.Mentions {
			if larkcore.StringValue(at.Name) == l.BotName {
				botShowName = larkcore.StringValue(at.Name)
				break
			}
		}
	} else if msgType == larkim.MsgTypePost {
		postMsg := new(MessagePostContent)
		err := json.Unmarshal([]byte(larkcore.StringValue(l.Message.Event.Message.Content)), postMsg)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "unmarshal text message error", "error", err)
			return false, err
		}

		for _, msgPostContents := range postMsg.Content {
			for _, msgPostContent := range msgPostContents {
				switch msgPostContent.Tag {
				case "text":
					command, prompt := ParseCommand(msgPostContent.Text)
					if command != "" {
						l.Command = command
					}
					if prompt != "" {
						l.Prompt = prompt
					}
				case "img":
					resp, err := l.Client.Im.V1.MessageResource.Get(l.Robot.Ctx,
						larkim.NewGetMessageResourceReqBuilder().
							MessageId(msgId).
							FileKey(msgPostContent.ImageKey).
							Type("image").
							Build())
					if err != nil || !resp.Success() {
						logger.ErrorCtx(l.Robot.Ctx, "get image failed", "err", err, "resp", resp)
						return false, err
					}

					bs, err := io.ReadAll(resp.File)
					if err != nil {
						logger.ErrorCtx(l.Robot.Ctx, "read image failed", "err", err)
						return false, err
					}
					l.ImageContent = bs
				case "at":
					if l.BotName == msgPostContent.UserName {
						botShowName = msgPostContent.UserName
					}

				}
			}
		}
	} else if msgType == larkim.MsgTypeAudio {
		msgAudio := new(larkim.MessageAudio)
		err := json.Unmarshal([]byte(larkcore.StringValue(l.Message.Event.Message.Content)), msgAudio)
		if err != nil {
			logger.Warn("unmarshal message audio failed", "err", err)
			return false, err
		}
		resp, err := l.Client.Im.V1.MessageResource.Get(l.Robot.Ctx,
			larkim.NewGetMessageResourceReqBuilder().
				MessageId(msgId).
				FileKey(msgAudio.FileKey).
				Type("file").
				Build())
		if err != nil || !resp.Success() {
			logger.ErrorCtx(l.Robot.Ctx, "get image failed", "err", err, "resp", resp)
			return false, err
		}

		bs, err := io.ReadAll(resp.File)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "read image failed", "err", err)
			return false, err
		}
		l.AudioContent = bs

		l.Prompt, err = l.Robot.GetAudioContent(bs)
		if err != nil {
			logger.Warn("generate text from audio failed", "err", err)
			return false, err
		}
	} else if msgType == larkim.MsgTypeImage {
		msgImage := new(larkim.MessageImage)
		err := json.Unmarshal([]byte(larkcore.StringValue(l.Message.Event.Message.Content)), msgImage)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "unmarshal message image failed", "err", err)
			return false, err
		}

		resp, err := l.Client.Im.V1.MessageResource.Get(l.Robot.Ctx,
			larkim.NewGetMessageResourceReqBuilder().
				MessageId(msgId).
				FileKey(msgImage.ImageKey).
				Type("image").
				Build())
		if err != nil || !resp.Success() {
			logger.ErrorCtx(l.Robot.Ctx, "get image failed", "err", err, "resp", resp)
			return false, err
		}

		l.ImageContent, err = io.ReadAll(resp.File)
		if err != nil {
			logger.ErrorCtx(l.Robot.Ctx, "read image failed", "err", err)
			return false, err
		}
	}

	l.Prompt = strings.ReplaceAll(l.Prompt, "@"+l.BotName, "")
	return botShowName == l.BotName, nil
}

func (l *LarkRobot) getPrompt() string {
	return l.Prompt
}

func (l *LarkRobot) getPerMsgLen() int {
	return 4500
}

func (l *LarkRobot) sendVoiceContent(voiceContent []byte, duration int) error {
	_, messageId, _ := l.Robot.GetChatIdAndMsgIdAndUserID()

	resp, err := l.Client.Im.V1.File.Create(l.Robot.Ctx, larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType("opus").
			FileName(utils.RandomFilename(".ogg")).
			Duration(duration).
			File(bytes.NewReader(voiceContent)).
			Build()).
		Build())
	if err != nil || !resp.Success() {
		logger.Warn("create voice fail", "err", err, "resp", resp)
		return errors.New("request upload file fail")
	}

	audio := larkim.MessageAudio{
		FileKey: *resp.Data.FileKey,
	}
	msgContent, _ := audio.String()

	updateRes, err := l.Client.Im.Message.Reply(l.Robot.Ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(messageId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeAudio).
			Content(msgContent).
			Build()).
		Build())
	if err != nil || !updateRes.Success() {
		logger.Warn("send message fail", "err", err, "resp", updateRes)
		return errors.New("send voice fail")
	}

	return err
}

func (l *LarkRobot) setCommand(command string) {
	l.Command = command
}

func (l *LarkRobot) getCommand() string {
	return l.Command
}

func (l *LarkRobot) getUserName() string {
	return l.UserName
}

func (l *LarkRobot) getVideoInfo(videoContent []byte) (string, error) {
	format := utils.DetectVideoMimeType(videoContent)
	resp, err := l.Client.Im.V1.File.Create(l.Robot.Ctx, larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType(format).
			FileName(utils.RandomFilename(format)).
			Duration(conf.VideoConfInfo.Duration).
			File(bytes.NewReader(videoContent)).
			Build()).
		Build())
	if err != nil || !resp.Success() {
		logger.ErrorCtx(l.Robot.Ctx, "create image fail", "err", err, "resp", resp)
		return "", err
	}

	return larkcore.StringValue(resp.Data.FileKey), nil
}

func (l *LarkRobot) getImageInfo(imageContent []byte) (string, error) {
	resp, err := l.Client.Im.V1.Image.Create(l.Robot.Ctx, larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(bytes.NewReader(imageContent)).
			Build()).
		Build())
	if err != nil || !resp.Success() {
		logger.Warn("create image fail", "err", err, "resp", resp)
		return "", err
	}

	return larkcore.StringValue(resp.Data.ImageKey), nil
}

func (l *LarkRobot) setPrompt(prompt string) {
	l.Prompt = prompt
}

func (l *LarkRobot) getAudio() []byte {
	return l.AudioContent
}

func (l *LarkRobot) getImage() []byte {
	return l.ImageContent
}

func (l *LarkRobot) setImage(image []byte) {
	l.ImageContent = image
}
