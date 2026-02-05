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
	Value          interface{} `json:"value"`
	ValueType      ValueType   `json:"value_type"`
	UseRandom      bool        `json:"use_random"`
	IntRandStart   int         `json:"int_rand_start"`
	IntRandEnd     int         `json:"int_rand_end"`
	FloatRandStart float64     `json:"float_rand_start"`
	FloatRandEnd   float64     `json:"float_rand_end"`
	RandomStrings  []string    `json:"random_strings"`
}

func (d *DataFields) SanitizedKey() string {
	return sanitizeEventName(d.Name)
}

// TypedValue casts the raw JSON value to the correct Go type expected by the AX event system.
// JSON deserializes all numbers as float64, so we must convert explicitly.
func (d *DataFields) TypedValue() interface{} {
	switch d.ValueType {
	case IntType:
		if f, ok := d.Value.(float64); ok {
			return int(f)
		}
		if i, ok := d.Value.(int); ok {
			return i
		}
		return 0
	case FloatType:
		if f, ok := d.Value.(float64); ok {
			return f
		}
		return 0.0
	case BoolType:
		if b, ok := d.Value.(bool); ok {
			return b
		}
		return false
	default:
		if s, ok := d.Value.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", d.Value)
	}
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
			Key:         dataField.SanitizedKey(),
			Value:       dataField.TypedValue(),
			ValueType:   valueType,
			KeyNiceName: &dataField.Name,
			IsData:      &isData,
		}
		eavt.Entries = append(eavt.Entries, entry)
	}
	e.PlatformEvent = *eavt
}

func (e *EvaEvent) BuildKeyValueMap() acapapp.KeyValueMap {
	kvmap := acapapp.KeyValueMap{}
	for _, field := range e.DataFields {
		key := field.SanitizedKey()
		kvmap[key] = field.TypedValue()
		if field.UseRandom {
			switch field.ValueType {
			case IntType:
				kvmap[key] = RandomIntInRange(field.IntRandStart, field.IntRandEnd)
			case FloatType:
				kvmap[key] = RandomFloatInRange(field.FloatRandStart, field.FloatRandEnd)
			case StringType:
				if len(field.RandomStrings) > 0 {
					kvmap[key] = RandomStringFromSlice(field.RandomStrings)
				}
			case BoolType:
				kvmap[key] = RandomBool()
			}
		}
	}
	return kvmap
}

func boolPtr(b bool) *bool {
	return &b
}

// SeedDemoEvents inserts demo events into the database if it's empty.
func (eva *EvaApplication) SeedDemoEvents() {
	var count int64
	eva.db.Model(&EvaEvent{}).Count(&count)
	if count > 0 {
		return
	}

	eva.acapp.Syslog.Info("Seeding demo events")

	demos := []EvaEvent{
		{
			Name:            "Object Count In Area",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 5,
			Stateless:       boolPtr(true),
			DataFields: []DataFields{
				{Name: "Total Count", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 0, IntRandEnd: 25},
				{Name: "Object Type", Value: "Person", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Person", "Vehicle", "Unknown"}},
				{Name: "Scenario", Value: "Counting Area 1", ValueType: StringType},
			},
		},
		{
			Name:            "Line Crossing Count",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 8,
			Stateless:       boolPtr(true),
			DataFields: []DataFields{
				{Name: "Crossings In", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 0, IntRandEnd: 50},
				{Name: "Crossings Out", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 0, IntRandEnd: 50},
				{Name: "Object Type", Value: "Person", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Person", "Vehicle"}},
			},
		},
		{
			Name:            "Person Detection",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 3,
			Stateless:       boolPtr(false),
			DataFields: []DataFields{
				{Name: "Active", Value: true, ValueType: BoolType, UseRandom: true},
				{Name: "Confidence", Value: 0.0, ValueType: FloatType, UseRandom: true, FloatRandStart: 0.5, FloatRandEnd: 1.0},
				{Name: "Scenario", Value: "Scenario 1", ValueType: StringType},
			},
		},
		{
			Name:            "Vehicle Detection",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 4,
			Stateless:       boolPtr(false),
			DataFields: []DataFields{
				{Name: "Active", Value: true, ValueType: BoolType, UseRandom: true},
				{Name: "Vehicle Type", Value: "Car", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Car", "Truck", "Bus", "Motorcycle", "Bicycle"}},
				{Name: "Confidence", Value: 0.0, ValueType: FloatType, UseRandom: true, FloatRandStart: 0.6, FloatRandEnd: 1.0},
			},
		},
		{
			Name:            "Object Classification",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 5,
			Stateless:       boolPtr(true),
			DataFields: []DataFields{
				{Name: "Class", Value: "Human", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Human", "Vehicle", "Animal", "Unknown"}},
				{Name: "Confidence", Value: 0.0, ValueType: FloatType, UseRandom: true, FloatRandStart: 0.4, FloatRandEnd: 1.0},
				{Name: "Object Id", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 1, IntRandEnd: 999},
			},
		},
		{
			Name:            "Motion Detection",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 2,
			Stateless:       boolPtr(false),
			DataFields: []DataFields{
				{Name: "Active", Value: true, ValueType: BoolType, UseRandom: true},
				{Name: "Motion Level", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 0, IntRandEnd: 100},
			},
		},
		{
			Name:            "Loitering Detection",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 10,
			Stateless:       boolPtr(false),
			DataFields: []DataFields{
				{Name: "Active", Value: false, ValueType: BoolType, UseRandom: true},
				{Name: "Duration Seconds", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 30, IntRandEnd: 600},
				{Name: "Object Type", Value: "Person", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Person", "Vehicle", "Unknown"}},
			},
		},
		{
			Name:            "Area Occupancy",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 5,
			Stateless:       boolPtr(true),
			DataFields: []DataFields{
				{Name: "Occupancy Count", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 0, IntRandEnd: 30},
				{Name: "Occupancy Percent", Value: 0.0, ValueType: FloatType, UseRandom: true, FloatRandStart: 0.0, FloatRandEnd: 100.0},
				{Name: "Scenario", Value: "Zone A", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Zone A", "Zone B", "Entrance", "Exit"}},
			},
		},
		{
			Name:            "Speed Estimation",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 3,
			Stateless:       boolPtr(true),
			DataFields: []DataFields{
				{Name: "Speed Kmh", Value: 0.0, ValueType: FloatType, UseRandom: true, FloatRandStart: 5.0, FloatRandEnd: 120.0},
				{Name: "Object Type", Value: "Vehicle", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Person", "Vehicle", "Bicycle"}},
				{Name: "Object Id", Value: 0, ValueType: IntType, UseRandom: true, IntRandStart: 1, IntRandEnd: 999},
			},
		},
		{
			Name:            "Crossline Detection",
			UseInterval:     boolPtr(true),
			IntervalSeconds: 6,
			Stateless:       boolPtr(false),
			DataFields: []DataFields{
				{Name: "Active", Value: true, ValueType: BoolType, UseRandom: true},
				{Name: "Direction", Value: "Left to Right", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Left to Right", "Right to Left"}},
				{Name: "Object Type", Value: "Person", ValueType: StringType, UseRandom: true, RandomStrings: []string{"Person", "Vehicle", "Unknown"}},
			},
		},
	}

	for i := range demos {
		if err := eva.db.Create(&demos[i]).Error; err != nil {
			eva.acapp.Syslog.Critf("Failed to seed event %s: %v", demos[i].Name, err)
		}
	}
	eva.acapp.Syslog.Infof("Seeded %d demo events", len(demos))
}
