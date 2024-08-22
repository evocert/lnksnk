package mqtt

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/iorw"
)

type Topic interface {
	Topic() string
	TopicPath() string
}

type activeTopic struct {
	topic     string
	topicpath string
}

func (atvtpc *activeTopic) Topic() string {
	return atvtpc.topic
}

func (atvtpc *activeTopic) TopicPath() string {
	return atvtpc.topicpath
}

type MqttMessaging func(message Message)

type MqttEvent interface {
	Event() string
	EventPath() string
	Connection() *MQTTConnection
	Manager() MQTTManagerAPI
	Args() map[string]interface{}
}

type mqttEvent struct {
	event     string
	eventpath string
	mqttcn    *MQTTConnection
	mqttmngr  *MQTTManager
	args      map[string]interface{}
}

func (mqttevnt *mqttEvent) Event() string {
	return mqttevnt.event
}

func (mqttevnt *mqttEvent) EventPath() string {
	return mqttevnt.eventpath
}

func (mqttevnt *mqttEvent) Connection() *MQTTConnection {
	return mqttevnt.mqttcn
}

func (mqttevnt *mqttEvent) Manager() MQTTManagerAPI {
	return mqttevnt.mqttmngr
}

func (mqttevnt *mqttEvent) Args() map[string]interface{} {
	return mqttevnt.args
}

type mqttEventContainer struct {
	event     string
	eventpath string
	args      map[string]interface{}
}

type MqttEventing func(event MqttEvent)

func (atvpc *activeTopic) processMessage(mqttmsng MqttMessaging, message Message) (err error) {
	if mqttmsng != nil {
		mqttmsng(message)
	}
	return
}

type MQTTManagerAPI interface {
	ActivateTopic(topic string, topicpath ...string)
	DeactivateTopic(topic string)
	ActiveTopics() (atvtpcs map[string]string)
	Connections() (aliases []string)
	Connection(alias string) (mqttcn *MQTTConnection)
	ConnectionInfo(alias string) (mqttcninfo string)
	Register(alias string, a ...interface{})
	Fprint(w io.Writer)
	String() (s string)
	Unregister(alias ...string)
	IsConnect(alias string) (connected bool)
	Connect(alias string) (err error)
	Disconnect(alias string, quiesce uint) (err error)
	IsSubscribed(alias string, topic string) (issbscrbed bool)
	Subscriptions(alias string) (subscrptns []*mqttsubscription)
	Subscribe(alias string, topic string, qos byte) (err error)
	Unsubscribe(alias string, topic string) (err error)
	Publish(alias string, topic string, qos byte, retained bool, message string) (err error)
}

type MQTTManager struct {
	lck          *sync.RWMutex
	cntns        map[string]*MQTTConnection
	activeTopics map[string]*activeTopic
	//defaulttopicpath string
	lcktpcs       *sync.RWMutex
	MqttMessaging MqttMessaging
	mqttevents    map[string]*mqttEventContainer
	lckevents     *sync.RWMutex
	MqttEventing  MqttEventing
	//defaulteventpath string
}

func NewMQTTManager(a ...interface{}) (mqttmngr *MQTTManager) {
	var mqttmsng MqttMessaging = nil
	var mqttevntng MqttEventing = nil
	if al := len(a); al > 0 {
		for al > 0 {
			d := a[0]
			if mqttmsng == nil {
				if mqttmsng, _ = d.(MqttMessaging); mqttmsng != nil {
					al--
					continue
				}
			} else if mqttevntng == nil {
				if mqttevntng, _ = d.(MqttEventing); mqttevntng != nil {
					al--
					continue
				}
			}
			al--
		}
	}

	mqttmngr = &MQTTManager{lck: &sync.RWMutex{}, cntns: map[string]*MQTTConnection{},
		activeTopics: map[string]*activeTopic{}, lcktpcs: &sync.RWMutex{}, MqttMessaging: mqttmsng,
		mqttevents: map[string]*mqttEventContainer{}, lckevents: &sync.RWMutex{}, MqttEventing: mqttevntng}
	return
}

func (mqttmngr *MQTTManager) ActiveTopics() (atvtpcs map[string]string) {
	if mqttmngr != nil {
		func() {
			mqttmngr.lcktpcs.RLock()
			defer mqttmngr.lcktpcs.RUnlock()
			for tpck := range mqttmngr.activeTopics {
				if tpc := mqttmngr.activeTopics[tpck]; tpc != nil {
					atvtpcs[tpck] = tpc.topicpath
				}
			}
		}()
	}
	return
}

func (mqttmngr *MQTTManager) Connections() (aliases []string) {
	if mqttmngr != nil {
		if len(mqttmngr.cntns) > 0 {
			func() {
				mqttmngr.lck.RLock()
				defer mqttmngr.lck.RUnlock()
				aliases = make([]string, len(mqttmngr.cntns))
				aliasi := 0
				for alias := range mqttmngr.cntns {
					aliases[aliasi] = alias
					aliasi++
				}
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Connection(alias string) (mqttcn *MQTTConnection) {
	if mqttmngr != nil {
		if mqttmngr.ConnectionExist(alias) {
			if mqttmngr != nil && alias != "" {
				func() {
					mqttmngr.lck.RLock()
					defer mqttmngr.lck.RUnlock()
					mqttcn = mqttmngr.cntns[alias]
				}()
			}
		}
	}
	return
}

func (mqttmngr *MQTTManager) ConnectionInfo(alias string) (mqttcninfo string) {
	if mqttmngr != nil {
		if mqttmngr.ConnectionExist(alias) {
			if mqttmngr != nil && alias != "" {
				func() {
					mqttmngr.lck.RLock()
					defer mqttmngr.lck.RUnlock()
					mqttcninfo = mqttmngr.cntns[alias].String()
				}()
			}
		}
	}
	return
}

func (mqttmngr *MQTTManager) ConnectionExist(alias string) (exists bool) {
	if mqttmngr != nil && alias != "" {
		func() {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			if len(mqttmngr.cntns) > 0 {
				_, exists = mqttmngr.cntns[alias]
			}
		}()
	}
	return
}

func (mqttmngr *MQTTManager) Register(alias string, a ...interface{}) {
	if alias != "" {
		if !func() (exists bool) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			_, exists = mqttmngr.cntns[alias]
			return
		}() {
			func() {
				mqttmngr.lck.Lock()
				if mqttcn := NewMQTTConnections(alias, a...); mqttcn != nil {
					mqttmngr.cntns[alias] = mqttcn
					mqttcn.mqttmngr = mqttmngr
				}
				mqttmngr.lck.Unlock()
				mqttmngr.fireAliasEvent(alias, "Registered", nil)
			}()
		}
	}
}

func (mqttmngr *MQTTManager) Fprint(w io.Writer) {
	if mqttmngr != nil && w != nil {
		enc := json.NewEncoder(w)
		iorw.Fprint(w, "{")

		iorw.Fprint(w, "\"connections\":")
		iorw.Fprint(w, "[")
		if cntns := mqttmngr.Connections(); len(cntns) > 0 {
			cntnsl := len(cntns)
			for cntn := range cntns {
				if cntn < cntnsl-1 {
					iorw.Fprint(w, mqttmngr.ConnectionInfo(cntns[cntn]))
					iorw.Fprint(w, ",")
				}
			}
		}
		iorw.Fprint(w, "],")
		iorw.Fprint(w, "\"activetopics\":")
		iorw.Fprint(w, "[")
		func() {
			mqttmngr.lcktpcs.RLock()
			defer mqttmngr.lcktpcs.RUnlock()
			if tpcsl := len(mqttmngr.activeTopics); tpcsl > 0 {
				tpcsi := 0
				for tpck := range mqttmngr.activeTopics {
					if tpc := mqttmngr.activeTopics[tpck]; tpc != nil {
						iorw.Fprint(w, "{")
						iorw.Fprint(w, "\"topic\":")
						enc.Encode(tpc.topic)
						iorw.Fprint(w, ",\"topicpath\":")
						enc.Encode(tpc.topicpath)
						iorw.Fprint(w, "}")
					} else {
						iorw.Fprint(w, "null")
					}
					tpcsi++
					if tpcsi < tpcsl {
						iorw.Fprint(w, ",")
					}
				}
			}
		}()
		iorw.Fprint(w, "]")
		iorw.Fprint(w, "}")
	}
}

func (mqttmngr *MQTTManager) String() (s string) {
	if mqttmngr != nil {
		pr, pw := io.Pipe()
		ctx, ctxcancel := context.WithCancel(context.Background())
		go func() {
			defer pw.Close()
			ctxcancel()
			mqttmngr.Fprint(pw)
		}()
		<-ctx.Done()
		if s, _ = iorw.ReaderToString(pr); s != "" {
			s = strings.Replace(s, "\n", "", -1)
		}
	}
	return
}

func (mqttmngr *MQTTManager) Unregister(alias ...string) {
	if mqttmngr != nil && len(alias) > 0 {
		for alsn := range alias {
			if als := alias[alsn]; als != "" {
				if func() (exists bool) {
					mqttmngr.lck.RLock()
					defer mqttmngr.lck.RUnlock()
					_, exists = mqttmngr.cntns[als]
					return
				}() {
					func() {
						mqttmngr.lck.Lock()
						defer mqttmngr.lck.Unlock()
						if mqttcn := mqttmngr.cntns[als]; mqttcn != nil {
							mqttcn.Unsubscribe(mqttcn.SubscribedTopics()...)
							mqttmngr.cntns[als] = nil
							delete(mqttmngr.cntns, als)
						}
					}()
					mqttmngr.fireAliasEvent(als, "Unregistered", nil)
				}
			}
		}
	}
}

func (mqttmngr *MQTTManager) messageReceived(mqttcn *MQTTConnection, alias string, msg *mqttMessage) {
	if mqttcn.autoack && msg != nil {
		msg.Ack()
	}
	if mqttmngr.MqttMessaging != nil && len(mqttmngr.activeTopics) > 0 {
		var atvtpc *activeTopic = nil
		func() {
			mqttmngr.lcktpcs.RLock()
			defer mqttmngr.lcktpcs.RUnlock()
			atvtpc = mqttmngr.activeTopics[msg.Topic()]
		}()
		//go func() {
		if atvtpc != nil {
			msg.tokenpath = atvtpc.topicpath
			atvtpc.processMessage(mqttmngr.MqttMessaging, msg)
		}
		//}()
	}
}

func (mqttmngr *MQTTManager) Connected(alias string) {
	if mqttmngr != nil {
		mqttmngr.fireAliasEvent(alias, "Connected", nil)
	}
}

func (mqttmngr *MQTTManager) fireAliasEvent(alias string, event string, err error) {
	if mqttmngr != nil && alias != "" && event != "" && mqttmngr.MqttEventing != nil {

		if mqttevntcnr := func() *mqttEventContainer {
			mqttmngr.lckevents.RLock()
			defer mqttmngr.lckevents.RUnlock()

			return mqttmngr.mqttevents[event]
		}(); mqttevntcnr != nil {
			func() {
				mqttevnt := &mqttEvent{mqttcn: mqttmngr.Connection(alias), mqttmngr: mqttmngr, event: event, eventpath: mqttevntcnr.eventpath, args: mqttevntcnr.args}
				defer func() {
					mqttevnt.args = nil
					mqttevnt.mqttcn = nil
					mqttevnt.mqttmngr = nil
					mqttevnt = nil
				}()
				mqttmngr.MqttEventing(mqttevnt)
			}()
		}
	}
}

func (mqttmngr *MQTTManager) Disconnected(alias string, err error) {
	if mqttmngr != nil {
		mqttmngr.fireAliasEvent(alias, "Disconnected", err)
	}
}

func (mqttmngr *MQTTManager) IsConnect(alias string) (connected bool) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				connected = mqttnc.IsConnected()
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Connect(alias string) (err error) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				err = mqttnc.Connect()
			}()
		}
		if mqttmngr != nil {
			mqttmngr.fireAliasEvent(alias, "Connected", err)
		}
	}
	return
}

func (mqttmngr *MQTTManager) Disconnect(alias string, quiesce uint) (err error) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				err = mqttnc.Disconnect(quiesce)
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) IsSubscribed(alias string, topic string) (issbscrbed bool) {
	if alias != "" && topic != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				issbscrbed = mqttnc.IsSubscribed(topic)
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Subscriptions(alias string) (subscrptns []*mqttsubscription) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				subscrptns = mqttnc.Subscriptions()
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Subscribe(alias string, topic string, qos byte) (err error) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				err = mqttnc.Subscribe(topic, qos)
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Unsubscribe(alias string, topic string) (err error) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				err = mqttnc.Unsubscribe(topic)
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) Publish(alias string, topic string, qos byte, retained bool, message string) (err error) {
	if alias != "" {
		if exsist, mqttnc := func() (exists bool, mqttcn *MQTTConnection) {
			mqttmngr.lck.RLock()
			defer mqttmngr.lck.RUnlock()
			mqttcn, exists = mqttmngr.cntns[alias]
			return
		}(); exsist {
			func() {
				err = mqttnc.Publish(topic, qos, retained, message)
			}()
		}
	}
	return
}

func (mqttmngr *MQTTManager) ActivateTopic(topic string, topicpath ...string) {
	if topic != "" {
		func() {
			var atvtpc *activeTopic = nil
			mqttmngr.lcktpcs.Lock()
			defer mqttmngr.lcktpcs.Unlock()
			if atvtpc = mqttmngr.activeTopics[topic]; atvtpc == nil {
				var topicpth = topic

				if len(topicpath) == 1 && topicpath[0] != "" {
					topicpth = topicpath[0]
				}
				atvtpc = &activeTopic{topic: topic, topicpath: topicpth}
				mqttmngr.activeTopics[topic] = atvtpc
			}
		}()
	}
}

func (mqttmngr *MQTTManager) DeactivateTopic(topic string) {
	if topic != "" {
		func() {
			mqttmngr.lcktpcs.Lock()
			defer mqttmngr.lcktpcs.Unlock()
			if atvtpc := mqttmngr.activeTopics[topic]; atvtpc != nil {
				mqttmngr.activeTopics[topic] = nil
				delete(mqttmngr.activeTopics, topic)
				atvtpc = nil
			}
		}()
	}
}

var validEvents []string = []string{"Connected", "Disconnected", "Registered", "Unregistered"}

func (mqttmngr *MQTTManager) ValidEvents() (events []string) {
	if mqttmngr != nil {
		events = validEvents[:]
	}
	return
}

func (mqttmngr *MQTTManager) ActivateEvent(event string, eventpath string, args ...map[string]interface{}) {
	if event != "" && strings.Contains(strings.Join(validEvents, "|"), event) {
		func() {
			var atvevnt *mqttEventContainer = nil
			mqttmngr.lckevents.Lock()
			defer mqttmngr.lckevents.Unlock()
			if atvevnt = mqttmngr.mqttevents[event]; atvevnt == nil {
				if eventpath == "" {
					eventpath = event
				}
				atvevnt = &mqttEventContainer{event: event, eventpath: eventpath, args: map[string]interface{}{}}
				if len(args) == 1 && len(args[0]) > 0 {
					for argk, argv := range args[0] {
						atvevnt.args[argk] = argv
					}
				}
				mqttmngr.mqttevents[event] = atvevnt
			}
		}()
	}
}

func (mqttmngr *MQTTManager) DeactivateEvent(event string) {
	if event != "" {
		func() {
			mqttmngr.lckevents.Lock()
			defer mqttmngr.lckevents.Unlock()
			if atvevnt := mqttmngr.mqttevents[event]; atvevnt != nil {
				mqttmngr.mqttevents[event] = nil
				delete(mqttmngr.mqttevents, event)
				if argsl := len(atvevnt.args); argsl > 0 {
					for atvarg := range atvevnt.args {
						atvevnt.args[atvarg] = nil
						delete(atvevnt.args, atvarg)
					}
				}
				atvevnt.args = nil
			}
		}()
	}
}

var gblmqttmngr *MQTTManager

func GLOBALMQTTMANAGER() *MQTTManager {
	return gblmqttmngr
}

func init() {
	gblmqttmngr = NewMQTTManager()
}
