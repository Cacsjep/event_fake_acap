package main

import (
	"fmt"

	"github.com/Cacsjep/goxis/pkg/acapapp"
	"github.com/Cacsjep/goxis/pkg/axevent"
	"github.com/Cacsjep/goxis/pkg/utils"
	"gorm.io/gorm"
)

type ValueType string

const (
	StringType ValueType = "string"
	IntType    ValueType = "int"
	FloatType  ValueType = "float"
	BoolType   ValueType = "bool"
)

type DataFields struct {
	Name           string      `json:"name"`
	Key            string      `json:"key"`
	Value          interface{} `json:"value"`
	ValueType      ValueType   `json:"value_type"`
	UseRandom      bool        `json:"use_random"`
	IntRandStart   int         `json:"int_rand_start"`
	IntRandEnd     int         `json:"int_rand_end"`
	FloatRandStart float64     `json:"float_rand_start"`
	FloatRandEnd   float64     `json:"float_rand_end"`
	RandomStrings  []string    `json:"random_strings"`
}

type EvaEvent struct {
	gorm.Model
	Name            string                      `json:"name"`
	UseInterval     *bool                       `json:"use_interval"`
	IntervalSeconds int                         `json:"interval_seconds"`
	DataFields      []DataFields                `gorm:"serializer:json"`
	Stateless       *bool                       `json:"stateless"`
	PlatformEvent   acapapp.CameraPlatformEvent `gorm:"-" json:"-"` // Filled at runtime after creation
	EventId         int                         `gorm:"-" json:"-"` // Filled at runtime after creation
}

func (e *EvaEvent) SetupPlatformEvent(eva *EvaApplication) {
	eavt := &acapapp.CameraPlatformEvent{
		Name:      sanitizeEventName(e.Name),
		NiceName:  utils.StrPtr(fmt.Sprintf("Eva - %s", e.Name)),
		Entries:   []*acapapp.EventEntry{},
		Stateless: *e.Stateless,
	}
	for _, dataField := range e.DataFields {
		var valueType axevent.AXEventValueType
		switch dataField.ValueType {
		case StringType:
			valueType = axevent.AXValueTypeString
		case IntType:
			valueType = axevent.AXValueTypeInt
		case FloatType:
			valueType = axevent.AXValueTypeDouble
		case BoolType:
			valueType = axevent.AXValueTypeBool
		default:
			valueType = axevent.AXValueTypeString
		}
		isData := true
		entry := &acapapp.EventEntry{
			Key:         dataField.Key,
			Value:       dataField.Value,
			ValueType:   valueType,
			KeyNiceName: &dataField.Name,
			IsData:      &isData,
		}
		eavt.Entries = append(eavt.Entries, entry)
	}
	e.PlatformEvent = *eavt
}
