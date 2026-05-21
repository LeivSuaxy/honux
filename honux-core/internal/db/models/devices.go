package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Microcontroller
type Controller struct {
	Base
	ZoneID        uuid.UUID  `json:"zone_id"`
	InducedID     *string    `json:"induced_id,omitempty"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	DeviceType    string     `json:"device_type"`
	LastIPAddress *string    `json:"last_ip_address,omitempty"`
	MQTTTopic     *string    `json:"mqtt_topic,omitempty"`
	IsOnline      bool       `json:"is_online"`
	LastPing      *time.Time `json:"last_ping,omitempty"`
	PosX          int        `json:"pos_x"`
	PosY          int        `json:"pos_y"`
	Zone          *Zone      `json:"zone,omitempty"`
}

// Active Components on Microcontroller
type Component struct {
	Base
	ControllerID uuid.UUID       `json:"controller_id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	GPIOPin      int             `json:"gpio_pin"`
	GPIOType     GPIOType        `json:"gpio_type"`
	PosX         int             `json:"pos_x"`
	PosY         int             `json:"pos_y"`
	CurrentState json.RawMessage `json:"current_state,omitempty"`
	Controller   *Controller     `json:"controller,omitempty"`
}
