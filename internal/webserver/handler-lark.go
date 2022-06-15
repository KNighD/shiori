package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/julienschmidt/httprouter"
	"github.com/larksuite/oapi-sdk-go/api/core/response"
	"github.com/larksuite/oapi-sdk-go/core"
	"github.com/larksuite/oapi-sdk-go/core/config"
	"github.com/larksuite/oapi-sdk-go/core/tools"
	im "github.com/larksuite/oapi-sdk-go/service/im/v1"
)

type EventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	CreateTime string `json:"create_time"`
	Token      string `json:"token"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
}

type UserID struct {
	UnionID string `json:"union_id"`
	UserID  string `json:"user_id"`
	OpenID  string `json:"open_id"`
}

type MentionEvent struct {
	Key       string `json:"key"`
	ID        UserID `json:"id"`
	Name      string `json:"name"`
	TenantKey string `json:"tenant_key"`
}

type Message struct {
	MessageID   string         `json:"message_id"`
	RootID      string         `json:"root_id"`
	ParentID    string         `json:"parent_id"`
	CreatedTime string         `json:"create_time"`
	ChatID      string         `json:"chat_id"`
	ChatType    string         `json:"chat_type"`
	MessageType string         `json:"message_type"`
	Content     string         `json:"content"`
	Mentions    []MentionEvent `json:"mentions"`
}

type EventSender struct {
	SenderID   UserID `json:"sender_id"`
	SenderType string `json:"sender_type"`
	TenantKey  string `json:"tenant_key"`
}

type Event struct {
	Sender  EventSender `json:"sender"`
	Message Message     `json:"message"`
}

type LarkMessageEvent struct {
	Schema string      `json:"schema"`
	Header EventHeader `json:"header"`
	Event  Event       `json:"event"`
}

// robot verification
type UrlVerificationEvent struct {
	Type      string `json:"url_verification"`
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
}

type LarkConfig struct {
	AppID             string
	AppSecret         string
	VerificationToken string
	EncryptKey        string
}

var conf *config.Config

func init() {
	file, err := filepath.Abs("./lark-config.json")
	checkError(err)
	data, err := ioutil.ReadFile(file)
	checkError(err)
	var config LarkConfig
	err = json.Unmarshal(data, &config)
	checkError(err)
	// 企业自建应用的配置
	// @see https://github.com/larksuite/oapi-sdk-go/blob/main/README.zh.md
	// AppID、AppSecret: "开发者后台" -> "凭证与基础信息" -> 应用凭证（App ID、App Secret）
	// EncryptKey、VerificationToken："开发者后台" -> "事件订阅" -> 事件订阅（Encrypt Key、Verification Token）
	// HelpDeskID、HelpDeskToken, 服务台 token：https://open.feishu.cn/document/ukTMukTMukTM/ugDOyYjL4gjM24CO4IjN
	// 更多介绍请看：Github->README.zh.md->如何构建应用配置（AppSettings）
	appSettings := core.NewInternalAppSettings(
		core.SetAppCredentials(config.AppID, config.AppSecret),           // 必需
		core.SetAppEventKey(config.VerificationToken, config.EncryptKey), // 非必需，订阅事件、消息卡片时必需
	// core.SetHelpDeskCredentials("HelpDeskID", "HelpDeskToken")
	) // 非必需，使用服务台API时必需

	// 当前访问的是飞书，使用默认的内存存储（app/tenant access token）、默认日志（Error级别）
	// 更多介绍请看：Github->README.zh.md->如何构建整体配置（Config）
	conf = core.NewConfig(core.DomainFeiShu, appSettings, core.SetLoggerLevel(core.LoggerLevelError))
}

func notify(receiveId string) {
	imService := im.NewService(conf)
	coreCtx := core.WrapContext(context.Background())
	reqCall := imService.Messages.Create(coreCtx, &im.MessageCreateReqBody{
		ReceiveId: receiveId,
		Content:   `{"text":"test content"}`,
		MsgType:   "text",
	})
	reqCall.SetReceiveIdType("open_id")
	message, err := reqCall.Do()
	// 打印 request_id 方便 oncall 时排查问题
	fmt.Println(coreCtx.GetRequestID())
	fmt.Println(coreCtx.GetHTTPStatusCode())
	if err != nil {
		fmt.Println(tools.Prettify(err))
		e := err.(*response.Error)
		fmt.Println(e.Code)
		fmt.Println(e.Msg)
		return
	}
	fmt.Println(tools.Prettify(message))
}

// 飞书机器人 API
func (h *handler) HandleLarkMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data, err := ioutil.ReadAll(r.Body)
	fmt.Printf("from feishu bot：%s\n", string(data))
	checkError(err)
	var event LarkMessageEvent
	json.Unmarshal(data, &event)
	w.Header().Set("Content-Type", "application/json")
	if event.Schema == "" {
		var verifyEvent UrlVerificationEvent
		err := json.Unmarshal(data, &verifyEvent)
		checkError(err)
		resp := map[string]interface{}{
			"challenge": verifyEvent.Challenge,
		}
		err = json.NewEncoder(w).Encode(&resp)
		checkError(err)
	} else {
		notify(event.Event.Sender.SenderID.OpenID)
	}
}
