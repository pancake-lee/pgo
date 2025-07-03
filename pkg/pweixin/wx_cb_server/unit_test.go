package main

import (
	"encoding/xml"
	"path/filepath"
	"testing"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"

	// "github.com/pancake-lee/pgo/pkg/pweixin/wx_cb_server/json_callback/wxbizjsonmsgcrypt"

	"github.com/pancake-lee/pgo/pkg/pweixin/wx_cb_server/xml_callback"
)

var msg_signature = "d258cfa79a31d24718e793b47e58e9faa32f9d33"
var timestamp = "1751532791"
var nonce = "1751337789"

var body = []byte(`<xml><ToUserName><![CDATA[ww671cddc7b394874a]]></ToUserName><Encrypt><![CDATA[mGbXVQ0UjbkUFDogAEhZqQWd4SZuAkWucksAdZNw+PAv3d7Wj6OS70+tC4IHabMsQ1IzrX5tJonLsOYKlwYfaF0eTyZ+NCaxn8Rg3sTGN+3fUFnjSNiY99QfE5WmM86RfPe+rT7lqCpowF1FyBLsRZMwb170Pl3J/aFeXq70syGn0WDWSXJfvI9rEMpbNynRFVwTTyxk7Fu8bcFpWpzkz4BBpmY8oZelbVz1cKS9hSY+eW1o+GebCwvWcxWYvDBSHaWKtBrKYbun7/Iwc7E0Qj4PMk8o9KVV5+Sc557OBtB0CBIav7/IL7cqk1/zCy1ZguhYS6K43rNt0OwegunppavxpRkljj4yrePQ7U1PGhJfs3Ta1NvUwIvUyPzNBnUlv3VqPJAbKuWPM6uVkA9mt56bwSv1m5zr/LCLfUYxBBMUAIrB/ttSJD9cPk0/3HK6hd6R3G0z9z86EXKlBUESpFbUz9LMr2GrVh+tpovPRoIZahAoBgbcp3zR23GPtF0EOE/zzHpUQmnKKanZ1pavXKx0j/JcLkd/XDyx/uukmu2tmYc68azjoA4rKZ4zF1q4Mm6WB2SEOeKuLPci8pOsR4Jhcnm66QnUInQW7Y6aI+N2DkI+d1kJDSH/jx9vsKNuJjUiMEyKqrbxKdQEHZyqxG80aABdXCSACSuE3M+u+LI=]]></Encrypt><AgentID><![CDATA[1000002]]></AgentID></xml>`)

func TestDecode(t *testing.T) {
	var err error
	plogger.InitConsoleLogger()

	pconfig.MustInitConfig(filepath.Join(putil.GetCurDir(), "../../../configs/pancake.yaml"))
	token = pconfig.GetStringM("WX.cbToken")
	encodingAeskey = pconfig.GetStringM("WX.cbEncodingAESKey")
	receiverId = pconfig.GetStringM("WX.cbReceiverId")

	wxcpt := xml_callback.NewWXBizMsgCrypt(token, encodingAeskey, receiverId, xml_callback.XmlType)

	msg, cryptErr := wxcpt.DecryptMsg(msg_signature, timestamp, nonce, body)
	if cryptErr != nil {
		plogger.Error("DecryptMsg fail", cryptErr)
		return
	}

	plogger.Debugf("decrypt msg: %v", string(msg))

	var req SheetChangeRecord
	err = xml.Unmarshal(msg, &req)
	if nil != err {
		plogger.Error("Unmarshal fail", err)
		return
	}

	if req.MsgType == "event" &&
		req.Event == "smart_sheet_change" &&
		req.ChangeType == "update_record" {

		plogger.Debugf("Received smart_sheet_change event: %v", req)
	} else {
		plogger.Debugf("Received unknown event: %v", req)
	}
}
