package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/Cacsjep/goxis/pkg/acapapp"
	"github.com/Cacsjep/goxis/pkg/axevent"
	"github.com/gofiber/fiber/v3"
)

// EvaApplication represents the main application structure.
type EvaApplication struct {
	acapp     acapapp.AcapApplication
	webserver *fiber.App
	events    []*EvaEvent
	mu        sync.Mutex
	wg        sync.WaitGroup
}

// NewEvaApplication creates a new instance of EvaApplication.
func NewEvaApplication() *EvaApplication {
	return &EvaApplication{
		webserver: fiber.New(),
		acapp:     *acapapp.NewAcapApplication(),
	}
}

func (eva *EvaApplication) Start() {
	eva.acapp.Syslog.Info("Starting Eva Application on :8746")

	eva.acapp.OnCloseCleaners = append(eva.acapp.OnCloseCleaners, func() {
		eva.webserver.Shutdown()
		eva.acapp.Syslog.Info("Shutting down Eva Application")
	})

	eva.RegisterRoutes()
	eva.acapp.RunInBackground() // This runs the gmain loop in the background
	eva.acapp.Syslog.Critf("Webserver error: %v", eva.webserver.Listen(":8746"))
}

func (eva *EvaApplication) RegisterRoutes() {
	eva.webserver.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	eva.webserver.Get("/events", func(c fiber.Ctx) error {
		eva.mu.Lock()
		defer eva.mu.Unlock()
		return c.JSON(eva.events)
	})

	eva.webserver.Post("/events", func(c fiber.Ctx) error {
		var newEvent acapapp.CameraPlatformEvent
		if err := c.Bind().Body(&newEvent); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		eva.mu.Lock()
		eva.events = append(eva.events, newEvent)
		eva.mu.Unlock()

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "event added"})
	})
}

func (eva *EvaApplication) RegisterEvents() error {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		reg_id, err := eva.acapp.AddCameraPlatformEvent(&event.PlatformEvent)
		if err != nil {
			return fmt.Errorf("error adding event %s: %s", event.Name, err.Error())
		}
		event.EventId = reg_id
	}
	return nil
}

func (eva *EvaApplication) UnregisterEvents() error {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		if event.EventId != 0 {
			if err := eva.acapp.EventHandler.Undeclare(event.EventId); err != nil {
				return fmt.Errorf("error removing event %s: %s", event.Name, err.Error())
			}
		}
	}
	return nil
}

func (eva *EvaApplication) StartEventSimulation() {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		if event.UseInterval != nil && *event.UseInterval {
			eva.wg.Add(1)
			go func(ev *EvaEvent) {
				defer eva.wg.Done()
				ticker := time.NewTicker(time.Duration(ev.IntervalSeconds) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						eva.acapp.SendPlatformEvent(ev.EventId, func() (*axevent.AXEvent, error) {
							kvmap := acapapp.KeyValueMap{}
							for _, field := range ev.DataFields {
								kvmap[field.Key] = field.Value
								if field.UseRandom {
									if field.ValueType == IntType {
										kvmap[field.Key] = RandomIntInRange(field.IntRandStart, field.IntRandEnd)
									} else if field.ValueType == FloatType {
										kvmap[field.Key] = RandomFloatInRange(field.FloatRandStart, field.FloatRandEnd)
									} else if field.ValueType == StringType && len(field.RandomStrings) > 0 {
										kvmap[field.Key] = RandomStringFromSlice(field.RandomStrings)
									} else if field.ValueType == BoolType {
										kvmap[field.Key] = RandomBool()
									}
								}
							}
							return ev.PlatformEvent.NewEvent(kvmap)
						})
					}
				}
			}(event)
		}
	}
}
