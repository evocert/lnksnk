package mqttsessioning

import (
	"github.com/evocert/lnksnk/mqtt"
	"github.com/evocert/lnksnk/serve"
)

var MQTTMNGR = mqtt.GLOBALMQTTMANAGER()

func init() {
	MQTTMNGR.MqttEventing = func(event mqtt.MqttEvent) {
		var mqttevent = map[string]interface{}{"mqttevent": event}
		defer func() { mqttevent = nil }()
		serve.ProcessRequestPath(event.EventPath(), mqttevent)
	}

	MQTTMNGR.MqttMessaging = func(message mqtt.Message) {
		var mqttmessage = map[string]interface{}{"mqttmessage": message}
		defer func() { mqttmessage = nil }()
		serve.ProcessRequestPath(message.TopicPath(), mqttmessage)
	}
}
