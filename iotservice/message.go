package iotservice

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/deviceonbi/iothub/common"
)

// FromAMQPMessage converts a amqp.Message into common.Message.
//
// Exported to use with a custom stream when devices telemetry is
// routed for example to an EventhHub instance.
func FromAMQPMessage(msg *amqp.Message) *common.Message {
	m := &common.Message{
		Payload:    msg.GetData(),
		Properties: make(map[string]string, len(msg.ApplicationProperties)+5),
	}
	if msg.Properties != nil {
		m.UserID = string(msg.Properties.UserID)
		if msg.Properties.MessageID != nil {
			m.MessageID = msg.Properties.MessageID.(string)
		}
		if msg.Properties.CorrelationID != nil {
			switch v := msg.Properties.CorrelationID.(type) {
			case string:
				m.CorrelationID = v
			case amqp.UUID:
				m.CorrelationID = v.String()
			default:
				// Handle the case where CorrelationID is an unexpected type
				fmt.Printf("unexpected type for CorrelationID: %T\n", v)
			}
		}
		if msg.Properties.To != nil {
			m.To = *msg.Properties.To
		}
		m.ExpiryTime = msg.Properties.AbsoluteExpiryTime
	}
	for k, v := range msg.Annotations {
		switch k {
		case "iothub-enqueuedtime":
			t, _ := v.(time.Time)
			m.EnqueuedTime = &t
		case "iothub-connection-device-id":
			m.ConnectionDeviceID = v.(string)
		case "iothub-connection-auth-generation-id":
			m.ConnectionDeviceGenerationID = v.(string)
		case "iothub-connection-auth-method":
			var am common.ConnectionAuthMethod
			if err := json.Unmarshal([]byte(v.(string)), &am); err != nil {
				m.Properties[k.(string)] = fmt.Sprint(v)
				continue
			}
			m.ConnectionAuthMethod = &am
		case "iothub-message-source":
			m.MessageSource = v.(string)
		default:
			m.Properties[k.(string)] = fmt.Sprint(v)
		}
	}

	for k, v := range msg.ApplicationProperties {
		if v, ok := v.(string); ok {
			m.Properties[k] = v
		} else {
			m.Properties[k] = ""
		}
	}
	return m
}

// toAMQPMessage converts amqp.Message into common.Message.
func toAMQPMessage(msg *common.Message) *amqp.Message {
	props := make(map[string]interface{}, len(msg.Properties))
	for k, v := range msg.Properties {
		props[k] = v
	}
	var expiryTime time.Time
	if msg.ExpiryTime != nil {
		expiryTime = *msg.ExpiryTime
	}
	return &amqp.Message{
		Data: [][]byte{msg.Payload},
		Properties: &amqp.MessageProperties{
			To:                 &msg.To,
			UserID:             []byte(msg.UserID),
			MessageID:          msg.MessageID,
			CorrelationID:      msg.CorrelationID,
			AbsoluteExpiryTime: &expiryTime,
		},
		ApplicationProperties: props,
	}
}
