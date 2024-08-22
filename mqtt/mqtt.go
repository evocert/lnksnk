package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/lnksnk/lnksnk/iorw"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttsubscription struct {
	topic string
	qos   byte
}

func (mqttsubscrptn *mqttsubscription) String() (s string) {
	if mqttsubscrptn != nil {
		pr, pw := io.Pipe()
		ctx, ctxcanncel := context.WithCancel(context.Background())
		go func() {
			var perr error = nil

			defer func() {
				if perr != nil {
					pw.CloseWithError(perr)
				} else {
					pw.Close()
				}
			}()
			ctxcanncel()
			mqttsubscrptn.Fprint(pw)
		}()
		<-ctx.Done()
		if s, _ = iorw.ReaderToString(pr); s != "" {
			s = strings.Replace(s, "\n", "", -1)
		}
	}
	return
}

func (mqttsubscrptn *mqttsubscription) Fprint(w io.Writer) (err error) {
	if mqttsubscrptn != nil && w != nil {
		enc := json.NewEncoder(w)
		if err = iorw.Fprint(w, "{"); err == nil {

			if err = iorw.Fprint(w, "\"topic\":"); err == nil {
				enc.Encode(mqttsubscrptn.topic)

				if err = iorw.Fprint(w, ",\"qos\":"); err == nil {
					enc.Encode(mqttsubscrptn.qos)
					err = iorw.Fprint(w, "}")
				}
			}
		}
	}
	return
}

type MQTTConnection struct {
	mqttmngr *MQTTManager
	pahomqtt mqtt.Client
	//pahomqtt event handlers
	connectHandler     mqtt.OnConnectHandler
	messagePubHandler  mqtt.MessageHandler
	connectLostHandler mqtt.ConnectionLostHandler
	ClientId           string
	broker             string
	port               int
	user               string
	password           string
	autoack            bool
	subscrptns         map[string]*mqttsubscription
	lcksubscrptns      *sync.RWMutex
}

func newMQTTOptions(clientid string, broker string, port int, user string, password string) (pahooptions *mqtt.ClientOptions) {
	pahooptions = mqtt.NewClientOptions()
	var schema = "tcp"
	if broker != "" && strings.HasPrefix(broker, "ws://") {
		schema = "ws"
		broker = broker[len("ws://"):]
	}
	pahooptions.AddBroker(fmt.Sprintf("%s://%s:%d", schema, broker, port))
	pahooptions.SetClientID(clientid)
	pahooptions.SetUsername(user)
	pahooptions.SetPassword(password)
	return
}

func NewMQTTConnections(clientid string, a ...interface{}) (mqttcn *MQTTConnection) {
	if clientid != "" {
		var broker string = ""
		var port int = 0
		var user string = ""
		var password string = ""
		var autoack bool = false
		for {
			if al := len(a); al > 0 {
				k := a[0]
				a = a[1:]
				if mp, mpok := k.(map[string]interface{}); mpok {
					for mk := range mp {
						mk = strings.ToLower(mk)
						if s, sok := mp[mk].(string); sok && s != "" {
							if mk == "broker" && broker == "" {
								broker = s
							} else if (mk == "user" || mk == "username") && user == "" {
								user = s
							} else if mk == "password" && password == "" {
								password = s
							}
						} else {
							if mk == "port" && port == 0 {
								if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 64); prsinterr == nil {
									port = int(prsint)
								} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 32); prsinterr == nil {
									port = int(prsint)
								} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 16); prsinterr == nil {
									port = int(prsint)
								}
							}
						}
					}
				} else if mp, mpok := k.(map[string]string); mpok {
					for mk := range mp {
						mk = strings.ToLower(mk)
						if mk == "broker" && mp[mk] != "" && broker == "" {
							broker = mp[mk]
						} else if (mk == "user" || mk == "username") && mp[mk] != "" && user == "" {
							user = mp[mk]
						} else if mk == "password" && mp[mk] != "" && password == "" {
							password = mp[mk]
						} else if mk == "port" && mp[mk] != "" && port == 0 {
							if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 64); prsinterr == nil {
								port = int(prsint)
							} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 32); prsinterr == nil {
								port = int(prsint)
							} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mp[mk]), 0, 16); prsinterr == nil {
								port = int(prsint)
							}
						}
					}
				} else if al > 1 {
					if mk, mkok := k.(string); mkok && mk != "" {
						mk = strings.ToLower(mk)
						if mv := a[0]; mv != nil {
							a = a[1:]
							if s, sok := mv.(string); sok && s != "" {
								if mk == "broker" && broker == "" {
									broker = s
								} else if mk == "password" && password == "" {
									password = s
								} else if (mk == "user" || mk == "username") && user == "" {
									user = s
								}
							} else if b, bok := mv.(bool); bok && b {
								if mk == "autoack" && b {
									autoack = b
								}
							} else {
								if mk == "port" && port == 0 {
									if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mv), 0, 64); prsinterr == nil {
										port = int(prsint)
									} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mv), 0, 32); prsinterr == nil {
										port = int(prsint)
									} else if prsint, prsinterr := strconv.ParseInt(fmt.Sprint(mv), 0, 16); prsinterr == nil {
										port = int(prsint)
									}
								}
							}
						} else {
							break
						}
					} else {
						break
					}
				}
			} else {
				break
			}
		}
		if pahooptions := newMQTTOptions(clientid, broker, port, user, password); pahooptions != nil {
			var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
				if mqttcn != nil && mqttcn.mqttmngr != nil {
					mqttcn.mqttmngr.Connected(clientid)
				}
			}

			var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
				if mqttcn != nil && mqttcn.mqttmngr != nil {
					mqttcn.mqttmngr.Disconnected(clientid, err)
				}
			}

			var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
				if mqttcn != nil && mqttcn.mqttmngr != nil {
					func() {
						var mqttmsg *mqttMessage = &mqttMessage{msg: msg, mqttcn: mqttcn, mqttmmng: mqttcn.mqttmngr}
						defer func() {
							mqttmsg.mqttcn = nil
							mqttmsg.msg = nil
							mqttmsg.tokenpath = ""
							mqttmsg = nil
							mqttmsg.mqttmmng = nil
						}()
						mqttcn.mqttmngr.messageReceived(mqttcn, clientid, mqttmsg)
					}()
				}
			}
			pahooptions.SetDefaultPublishHandler(messagePubHandler)
			pahooptions.OnConnect = connectHandler
			pahooptions.OnConnectionLost = connectLostHandler
			pahomqtt := newPahoMqttClient(pahooptions)
			mqttcn = &MQTTConnection{mqttmngr: nil, pahomqtt: pahomqtt, broker: broker, port: port, user: user, password: password, ClientId: clientid, autoack: autoack,
				connectHandler: connectHandler, messagePubHandler: messagePubHandler, connectLostHandler: connectLostHandler,
				subscrptns: map[string]*mqttsubscription{}, lcksubscrptns: &sync.RWMutex{}}
		}
	}
	return
}

func newPahoMqttClient(pahooptions *mqtt.ClientOptions) (pahomqtt mqtt.Client) {
	pahomqtt = mqtt.NewClient(pahooptions)
	return
}

type Message interface {
	Duplicate() bool
	Qos() byte
	Retained() bool
	Topic() string
	MessageID() uint16
	Payload() []byte
	Connection() *MQTTConnection
	TopicPath() string
	Manager() MQTTManagerAPI
	Ack()
}

type mqttMessage struct {
	mqttcn    *MQTTConnection
	mqttmmng  *MQTTManager
	msg       mqtt.Message
	tokenpath string
}

func (mqttmsg *mqttMessage) FPrint(w io.Writer) (err error) {
	if mqttmsg != nil && w != nil {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "")
		if err = iorw.Fprint(w, "{"); err != nil {
			return
		}
		enc.Encode("msgid")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.msg.MessageID())
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("clientid")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.mqttcn.ClientId)
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("duplicate")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.msg.Duplicate())
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		payload := mqttmsg.msg.Payload()
		enc.Encode("payload")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(string(payload))
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("bin-payload")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		arrpayload := make([]interface{}, len(payload))
		for pn := range payload {
			arrpayload[pn] = payload[pn]
		}
		enc.Encode(arrpayload)
		arrpayload = nil
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("topic")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.msg.Topic())
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("qos")
		iorw.Fprint(w, ":")
		if err = enc.Encode(mqttmsg.msg.Qos()); err != nil {
			return
		}
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("retained")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.msg.Retained())
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}
		enc.Encode("topicpath")
		if err = iorw.Fprint(w, ":"); err != nil {
			return
		}
		enc.Encode(mqttmsg.tokenpath)
		if err = iorw.Fprint(w, "}"); err != nil {
			return
		}
	}
	return
}

func (mqttmsg *mqttMessage) String() (s string) {
	pr, pw := io.Pipe()
	defer pr.Close()
	ctx, ctxcancel := context.WithCancel(context.Background())
	go func() {
		defer pw.Close()
		ctxcancel()
		mqttmsg.FPrint(pw)
	}()
	<-ctx.Done()
	s, _ = iorw.ReaderToString(pr)
	s = strings.Replace(s, "\n", "", -1)
	return
}

func (mqttmsg *mqttMessage) TopicPath() (topicpath string) {
	if mqttmsg != nil {
		topicpath = mqttmsg.tokenpath
	}
	return
}

func (mqttmsg *mqttMessage) Duplicate() bool {
	return mqttmsg.msg.Duplicate()
}

func (mqttmsg *mqttMessage) Qos() byte {
	return mqttmsg.msg.Qos()
}

func (mqttmsg *mqttMessage) Retained() bool {
	return mqttmsg.msg.Retained()
}

func (mqttmsg *mqttMessage) Topic() string {
	return mqttmsg.msg.Topic()
}

func (mqttmsg *mqttMessage) MessageID() uint16 {
	return mqttmsg.msg.MessageID()
}

func (mqttmsg *mqttMessage) Payload() []byte {
	return mqttmsg.msg.Payload()
}

func (mqttmsg *mqttMessage) Connection() *MQTTConnection {
	return mqttmsg.mqttcn
}

func (mqttmsg *mqttMessage) Manager() MQTTManagerAPI {
	return mqttmsg.mqttmmng
}

func (mqttmsg *mqttMessage) Ack() {
	mqttmsg.msg.Ack()
}

func (mqttcn *MQTTConnection) Fprint(w io.Writer) (err error) {
	if mqttcn != nil && w != nil {
		enc := json.NewEncoder(w)
		if err = iorw.Fprint(w, "{"); err != nil {
			return
		}
		if iorw.Fprint(w, "\"ClientID\":"); err != nil {
			return
		}
		enc.Encode(mqttcn.ClientId)
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}

		if err = iorw.Fprint(w, "\"broker\":"); err != nil {
			return
		}
		enc.Encode(mqttcn.broker)
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}

		if err = iorw.Fprint(w, "\"port\":"); err != nil {
			return
		}
		enc.Encode(mqttcn.port)
		if err = iorw.Fprint(w, ","); err != nil {
			return
		}

		if iorw.Fprint(w, "\"user\":"); err != nil {
			return
		}
		enc.Encode(mqttcn.user)
		iorw.Fprint(w, ",")

		iorw.Fprint(w, "\"password\":")
		enc.Encode(mqttcn.password)
		iorw.Fprint(w, ",")

		iorw.Fprint(w, "\"autoack\":")
		enc.Encode(mqttcn.autoack)
		iorw.Fprint(w, ",")

		iorw.Fprint(w, "\"status\":")
		if mqttcn.IsConnected() {
			iorw.Fprint(w, "\"connected\"")
		} else {
			iorw.Fprint(w, "\"disconnected\"")
		}
		iorw.Fprint(w, ",")
		iorw.Fprint(w, "\"subscriptions\":")
		iorw.Fprint(w, "[")
		func() {
			mqttcn.lcksubscrptns.RLock()
			defer mqttcn.lcksubscrptns.RUnlock()
			if subscrptns := mqttcn.Subscriptions(true); len(subscrptns) > 0 {
				if subscrbl := len(subscrptns); subscrbl > 0 {
					for nsubscrb := range subscrptns {
						subscrptns[nsubscrb].Fprint(w)
						if nsubscrb < subscrbl-1 {
							iorw.Fprint(w, ",")
						}
					}
				}
			}
		}()

		iorw.Fprint(w, "]")
		iorw.Fprint(w, "}")
	}
	return
}

func (mqttcn *MQTTConnection) String() (s string) {
	if mqttcn != nil {
		pr, pw := io.Pipe()
		ctx, ctxcancel := context.WithCancel(context.Background())
		go func() {
			defer pw.Close()
			ctxcancel()
			mqttcn.Fprint(pw)
		}()
		<-ctx.Done()
		if s, _ = iorw.ReaderToString(pr); s != "" {
			s = strings.Replace(s, "\n", "", -1)
		}
	}
	return
}

func (mqttcn *MQTTConnection) IsConnected() (connected bool) {
	if mqttcn != nil && mqttcn.pahomqtt != nil {
		connected = mqttcn.pahomqtt.IsConnected()
	}
	return
}

func (mqttcn *MQTTConnection) Disconnect(quiesce uint) (err error) {
	if mqttcn != nil {
		if mqttcn.IsConnected() {
			if client := mqttcn.pahomqtt; client != nil {
				client.Disconnect(quiesce)
			}
		}
	}
	return
}

func (mqttcn *MQTTConnection) Connect() (err error) {
	if mqttcn != nil {
		if !mqttcn.IsConnected() {
			if client := mqttcn.pahomqtt; client != nil {
				if token := client.Connect(); token.Wait() && token.Error() != nil {
					err = token.Error()
				}
			}
		}
	}
	return
}

func (mqttcn *MQTTConnection) UpdateConnection(autoconnect bool, broker string, port int, user string, password string) {
	if mqttcn != nil {
		if mqttcn.IsConnected() {

		} else {

		}
	}
}

func (mqttcn *MQTTConnection) Publish(topic string, qos byte, retained bool, message string) (err error) {
	if mqttcn != nil && mqttcn.pahomqtt != nil {
		tkn := mqttcn.pahomqtt.Publish(topic, qos, retained, message)
		tkn.Wait()
		err = tkn.Error()
	}
	return err
}

func (mqttcn *MQTTConnection) IsSubscribed(topic string) (issbscrbed bool) {
	if mqttcn != nil && topic != "" {
		func() {
			mqttcn.lcksubscrptns.RLock()
			defer mqttcn.lcksubscrptns.RUnlock()
			_, issbscrbed = mqttcn.subscrptns[topic]
		}()
	}
	return
}

func (mqttcn *MQTTConnection) Subscriptions(alreadylck ...bool) (subscrptns []*mqttsubscription) {
	if mqttcn != nil {
		func() {
			if len(alreadylck) == 0 || len(alreadylck) > 0 && !alreadylck[0] {
				mqttcn.lcksubscrptns.RLock()
				defer mqttcn.lcksubscrptns.RUnlock()
			}
			if subscrpl := len(mqttcn.subscrptns); subscrpl > 0 {
				subscrptns = make([]*mqttsubscription, subscrpl)
				subscrpi := 0
				for mqttsubscrptn := range mqttcn.subscrptns {
					subscrptns[subscrpi] = mqttcn.subscrptns[mqttsubscrptn]
					subscrpi++
				}
			}
		}()
	}
	return
}

func (mqttcn *MQTTConnection) SubscribedTopics(alreadylck ...bool) (subscrbdtpcs []string) {
	if mqttcn != nil {
		func() {
			if len(alreadylck) == 0 || len(alreadylck) > 0 && !alreadylck[0] {
				mqttcn.lcksubscrptns.RLock()
				defer mqttcn.lcksubscrptns.RUnlock()
			}
			if subscrpl := len(mqttcn.subscrptns); subscrpl > 0 {
				subscrbdtpcs = make([]string, subscrpl)
				subscrpi := 0
				for mqttsubscrptn := range mqttcn.subscrptns {
					subscrbdtpcs[subscrpi] = mqttcn.subscrptns[mqttsubscrptn].topic
					subscrpi++
				}
			}
		}()
	}
	return
}

func (mqttcn *MQTTConnection) Subscribe(topic string, qos byte) (err error) {
	if mqttcn != nil && mqttcn.pahomqtt != nil && topic != "" {
		if !func() bool {
			mqttcn.lcksubscrptns.RLock()
			defer mqttcn.lcksubscrptns.RUnlock()
			if sbcptn, subscrbed := mqttcn.subscrptns[topic]; subscrbed && sbcptn.topic == topic && sbcptn.qos == qos {
				return true
			}
			return false
		}() {

			if err = subscribetotopic(mqttcn, topic, qos); err == nil {
				func() {
					mqttcn.lcksubscrptns.Lock()
					defer mqttcn.lcksubscrptns.Unlock()
					mqttcn.subscrptns[topic] = &mqttsubscription{topic: topic, qos: qos}
				}()
			}
		}
	}
	return err
}

func subscribetotopic(mqttcn *MQTTConnection, topic string, qos byte) (err error) {
	tkn := mqttcn.pahomqtt.Subscribe(topic, qos, mqttcn.messagePubHandler)
	tkn.Wait()
	err = tkn.Error()
	return
}

func (mqttcn *MQTTConnection) Resubscribe(unsubscribefirst bool, topic ...string) (err error) {
	if mqttcn != nil && mqttcn.IsConnected() {
		if len(topic) > 0 {
			var mqttsbscrptns []*mqttsubscription = []*mqttsubscription{}
			for tpcn := range topic {
				if tpc := topic[tpcn]; tpc != "" && mqttcn.IsSubscribed(tpc) {
					func() {
						mqttcn.lcksubscrptns.Lock()
						defer mqttcn.lcksubscrptns.Unlock()
						mqttsbscrptn := mqttcn.subscrptns[tpc]
						mqttsbscrptns = append(mqttsbscrptns, &mqttsubscription{topic: mqttsbscrptn.topic, qos: mqttsbscrptn.qos})
						mqttsbscrptn = nil
					}()
				}
			}
			if unsubscribefirst {
				err = mqttcn.Unsubscribe(topic...)
			}
			var mqttsbscrptn *mqttsubscription = nil
			subn := 0
			for subn = range mqttsbscrptns {
				mqttsbscrptn = mqttsbscrptns[subn]
				if unsubscribefirst {
					func() {
						mqttcn.lcksubscrptns.Lock()
						defer mqttcn.lcksubscrptns.Unlock()
						mqttcn.subscrptns[mqttsbscrptn.topic] = mqttsbscrptn
						mqttsbscrptns[subn] = nil
					}()
				}
				subscribetotopic(mqttcn, mqttsbscrptn.topic, mqttsbscrptn.qos)
			}
		}
	}
	return
}

func (mqttcn *MQTTConnection) UnsubscribeAll() (err error) {
	if mqttcn != nil {
		err = mqttcn.Unsubscribe(mqttcn.SubscribedTopics()...)
	}
	return
}

func (mqttcn *MQTTConnection) Unsubscribe(topic ...string) (err error) {
	if mqttcn != nil && mqttcn.pahomqtt != nil && len(topic) > 0 {
		if func() bool {
			mqttcn.lcksubscrptns.RLock()
			defer mqttcn.lcksubscrptns.RUnlock()
			tpci := 0
			for tpci < len(topic) {
				if _, issbscrbed := mqttcn.subscrptns[topic[tpci]]; issbscrbed {
					tpci++
				} else {
					topic = append(topic[:tpci], topic[tpci+1:]...)
				}
			}
			return len(topic) > 0
		}() {
			tkn := mqttcn.pahomqtt.Unsubscribe(topic...)
			tkn.Wait()
			if err = tkn.Error(); err == nil {
				func() {
					mqttcn.lcksubscrptns.Lock()
					defer mqttcn.lcksubscrptns.Unlock()
					for tpcn := range topic {
						if tpc := topic[tpcn]; tpc != "" {
							if mqttsubscptn, mqttsubscptnok := mqttcn.subscrptns[tpc]; mqttsubscptnok {
								mqttcn.subscrptns[tpc] = nil
								if mqttsubscptn != nil {
									mqttsubscptn = nil
								}
								delete(mqttcn.subscrptns, tpc)
							}
						}
					}

				}()
			}
		}
	}
	return err
}

func init() {

}
