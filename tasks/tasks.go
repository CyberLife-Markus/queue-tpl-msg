package wechat

import (
	"gopkg.in/chanxuehong/wechat.v2/mp/core"
	"gopkg.in/chanxuehong/wechat.v2/mp/message/template"
	"os"
)

// TplMessage 消息结构
type TemplateMsg struct {
	First    template.DataItem `json:"first"`
	Keyword1 template.DataItem `json:"keyword1"`
	Keyword2 template.DataItem `json:"keyword2"`
	Remark   template.DataItem `json:"remark"`
}

func PushTplMsg(data []string) (int64, error) {
	wxAppID := os.Getenv("WECHAT_APPID")
	wxAppSecret := os.Getenv("WECHAT_APPSECRET")
	ats := core.NewDefaultAccessTokenServer(wxAppID, wxAppSecret, nil)

	client := core.NewClient(ats, nil)

	tpl := TemplateMsg{
		First:    template.DataItem{Value: "您有新的签约试音任务，请及时查看并提交试音文件", Color: "#ff0000"},
		Keyword1: template.DataItem{Value: data[1], Color: "#19439c"},
		Keyword2: template.DataItem{Value: data[2], Color: "#19439c"},
		Remark:   template.DataItem{Value: os.Getenv("WECHAT_REMARK"), Color: ""},
	}

	msg := template.TemplateMessage2{
		ToUser:     data[0],
		TemplateId: os.Getenv("WECHAT_TEMPLATE"),
		URL:        data[3],
		Data:       tpl,
	}

	result, err := template.Send(client, msg)

	if err != nil {
		return 500, err
	}

	return result, nil
}
