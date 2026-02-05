package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Cacsjep/goxis/pkg/acapapp"
	"github.com/Cacsjep/goxis/pkg/axevent"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// EvaApplication represents the main application structure.
type EvaApplication struct {
	acapp      acapapp.AcapApplication
	webserver  *fiber.App
	db         *gorm.DB
	events     []*EvaEvent
	mu         sync.Mutex
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	simRunning bool
}

// NewEvaApplication creates a new instance of EvaApplication.
func NewEvaApplication() *EvaApplication {
	return &EvaApplication{
		webserver: fiber.New(),
		acapp:     *acapapp.NewAcapApplication(),
	}
}

func (eva *EvaApplication) InitDB() error {
	if err := os.MkdirAll("./localdata", 0755); err != nil {
		return fmt.Errorf("failed to create localdata directory: %w", err)
	}
	db, err := gorm.Open(sqlite.Open("./localdata/db.sqlite"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.AutoMigrate(&EvaEvent{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	eva.db = db
	return nil
}

func (eva *EvaApplication) Start() {
	eva.acapp.Syslog.Info("Starting Eva - Event Virtualizer for ACAP on :8746")

	if err := eva.InitDB(); err != nil {
		eva.acapp.Syslog.Critf("Database error: %v", err)
		return
	}

	eva.SeedDemoEvents()

	if err := eva.LoadAndRegisterAllEvents(); err != nil {
		eva.acapp.Syslog.Critf("Failed to register events on startup: %v", err)
	}

	eva.acapp.OnCloseCleaners = append(eva.acapp.OnCloseCleaners, func() {
		eva.StopSimulation()
		if err := eva.UnregisterAllEvents(); err != nil {
			eva.acapp.Syslog.Critf("Failed to unregister events on shutdown: %v", err)
		}
		eva.webserver.Shutdown()
		eva.acapp.Syslog.Info("Shutting down Eva - Event Virtualizer for ACAP")
	})

	eva.webserver.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
	}))
	eva.RegisterRoutes()
	eva.acapp.RunInBackground()
	eva.acapp.Syslog.Critf("Webserver error: %v", eva.webserver.Listen(":8746"))
}

func jsonError(c fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

func (eva *EvaApplication) findEventByID(c fiber.Ctx) (*EvaEvent, error) {
	var event EvaEvent
	if err := eva.db.First(&event, c.Params("id")).Error; err != nil {
		return nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}
	return &event, nil
}

func (eva *EvaApplication) RegisterRoutes() {
	// List all events
	eva.webserver.Get("/events", func(c fiber.Ctx) error {
		var events []EvaEvent
		if err := eva.db.Find(&events).Error; err != nil {
			return jsonError(c, fiber.StatusInternalServerError, err)
		}
		return c.JSON(events)
	})

	// Get single event
	eva.webserver.Get("/events/:id", func(c fiber.Ctx) error {
		event, err := eva.findEventByID(c)
		if err != nil {
			return err
		}
		return c.JSON(event)
	})

	// Create event
	eva.webserver.Post("/events", func(c fiber.Ctx) error {
		eva.mu.Lock()
		if eva.simRunning {
			eva.mu.Unlock()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "cannot create events while simulation is running"})
		}
		eva.mu.Unlock()

		var newEvent EvaEvent
		if err := c.Bind().Body(&newEvent); err != nil {
			return jsonError(c, fiber.StatusBadRequest, err)
		}
		if err := eva.db.Create(&newEvent).Error; err != nil {
			return jsonError(c, fiber.StatusInternalServerError, err)
		}

		eva.mu.Lock()
		eva.events = append(eva.events, &newEvent)
		if err := eva.registerEvent(&newEvent); err != nil {
			eva.mu.Unlock()
			eva.acapp.Syslog.Critf("Failed to register new event %s: %v", newEvent.Name, err)
			return c.Status(fiber.StatusCreated).JSON(newEvent)
		}
		eva.mu.Unlock()

		return c.Status(fiber.StatusCreated).JSON(newEvent)
	})

	// Update event
	eva.webserver.Put("/events/:id", func(c fiber.Ctx) error {
		eva.mu.Lock()
		if eva.simRunning {
			eva.mu.Unlock()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "cannot update events while simulation is running"})
		}
		eva.mu.Unlock()

		event, err := eva.findEventByID(c)
		if err != nil {
			return err
		}
		if err := c.Bind().Body(event); err != nil {
			return jsonError(c, fiber.StatusBadRequest, err)
		}
		if err := eva.db.Save(event).Error; err != nil {
			return jsonError(c, fiber.StatusInternalServerError, err)
		}

		eva.mu.Lock()
		registered := eva.findRegisteredEvent(event.ID)
		if registered != nil {
			eva.unregisterEvent(registered)
			*registered = *event
			eva.registerEvent(registered)
		}
		eva.mu.Unlock()

		return c.JSON(event)
	})

	// Delete event
	eva.webserver.Delete("/events/:id", func(c fiber.Ctx) error {
		eva.mu.Lock()
		if eva.simRunning {
			eva.mu.Unlock()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "cannot delete events while simulation is running"})
		}
		eva.mu.Unlock()

		event, err := eva.findEventByID(c)
		if err != nil {
			return err
		}

		eva.mu.Lock()
		registered := eva.findRegisteredEvent(event.ID)
		if registered != nil {
			eva.unregisterEvent(registered)
			eva.removeRegisteredEvent(event.ID)
		}
		eva.mu.Unlock()

		if err := eva.db.Delete(&EvaEvent{}, event.ID).Error; err != nil {
			return jsonError(c, fiber.StatusInternalServerError, err)
		}
		return c.JSON(fiber.Map{"status": "event deleted"})
	})

	// Start simulation
	eva.webserver.Post("/simulation/start", func(c fiber.Ctx) error {
		eva.mu.Lock()
		if eva.simRunning {
			eva.mu.Unlock()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "simulation already running"})
		}
		if len(eva.events) == 0 {
			eva.mu.Unlock()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no events configured"})
		}
		eva.mu.Unlock()

		eva.ctx, eva.cancel = context.WithCancel(context.Background())
		eva.StartEventSimulation()

		eva.mu.Lock()
		eva.simRunning = true
		eventCount := len(eva.events)
		eva.mu.Unlock()

		return c.JSON(fiber.Map{"status": "simulation started", "event_count": eventCount})
	})

	// Stop simulation
	eva.webserver.Post("/simulation/stop", func(c fiber.Ctx) error {
		eva.mu.Lock()
		if !eva.simRunning {
			eva.mu.Unlock()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "simulation not running"})
		}
		eva.mu.Unlock()

		eva.StopSimulation()

		return c.JSON(fiber.Map{"status": "simulation stopped"})
	})

	// Manual trigger a single event by DB id
	eva.webserver.Post("/events/:id/trigger", func(c fiber.Ctx) error {
		event, err := eva.findEventByID(c)
		if err != nil {
			return err
		}

		eva.mu.Lock()
		registered := eva.findRegisteredEvent(event.ID)
		if registered == nil || registered.EventId == 0 {
			eva.mu.Unlock()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "event not registered with platform"})
		}
		eva.acapp.SendPlatformEvent(registered.EventId, func() (*axevent.AXEvent, error) {
			return registered.PlatformEvent.NewEvent(registered.BuildKeyValueMap())
		})
		eva.mu.Unlock()

		return c.JSON(fiber.Map{"status": "event triggered", "event": event.Name})
	})

	// Simulation status
	eva.webserver.Get("/simulation/status", func(c fiber.Ctx) error {
		eva.mu.Lock()
		defer eva.mu.Unlock()
		return c.JSON(fiber.Map{"running": eva.simRunning, "event_count": len(eva.events)})
	})

	// Serve frontend (must be last)
	eva.webserver.Use("/", static.New("./html", static.Config{
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.SendFile("./html/index.html")
		},
	}))
}

func (eva *EvaApplication) RegisterAllEvents() error {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		if err := eva.registerEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (eva *EvaApplication) UnregisterAllEvents() error {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		if err := eva.unregisterEvent(event); err != nil {
			return err
		}
	}
	return nil
}

// registerEvent registers a single event with the platform. Caller must hold eva.mu.
func (eva *EvaApplication) registerEvent(event *EvaEvent) error {
	event.SetupPlatformEvent(eva)
	regId, err := eva.acapp.AddCameraPlatformEvent(&event.PlatformEvent)
	if err != nil {
		return fmt.Errorf("error registering event %s: %s", event.Name, err.Error())
	}
	event.EventId = regId
	eva.acapp.Syslog.Infof("Registered event: %s (id=%d)", event.Name, regId)
	return nil
}

// unregisterEvent unregisters a single event from the platform. Caller must hold eva.mu.
func (eva *EvaApplication) unregisterEvent(event *EvaEvent) error {
	if event.EventId != 0 {
		if err := eva.acapp.EventHandler.Undeclare(event.EventId); err != nil {
			return fmt.Errorf("error unregistering event %s: %s", event.Name, err.Error())
		}
		eva.acapp.Syslog.Infof("Unregistered event: %s (id=%d)", event.Name, event.EventId)
		event.EventId = 0
	}
	return nil
}

// findRegisteredEvent finds an event in the in-memory list by DB ID. Caller must hold eva.mu.
func (eva *EvaApplication) findRegisteredEvent(dbID uint) *EvaEvent {
	for _, ev := range eva.events {
		if ev.ID == dbID {
			return ev
		}
	}
	return nil
}

// removeRegisteredEvent removes an event from the in-memory list by DB ID. Caller must hold eva.mu.
func (eva *EvaApplication) removeRegisteredEvent(dbID uint) {
	for i, ev := range eva.events {
		if ev.ID == dbID {
			eva.events = append(eva.events[:i], eva.events[i+1:]...)
			return
		}
	}
}

// LoadAndRegisterAllEvents loads all events from DB and registers them with the platform.
func (eva *EvaApplication) LoadAndRegisterAllEvents() error {
	var events []EvaEvent
	if err := eva.db.Find(&events).Error; err != nil {
		return fmt.Errorf("failed to load events: %w", err)
	}
	eva.mu.Lock()
	eva.events = make([]*EvaEvent, len(events))
	for i := range events {
		eva.events[i] = &events[i]
	}
	eva.mu.Unlock()

	if err := eva.RegisterAllEvents(); err != nil {
		return fmt.Errorf("failed to register events: %w", err)
	}
	eva.acapp.Syslog.Infof("Loaded and registered %d events", len(events))
	return nil
}

func (eva *EvaApplication) StartEventSimulation() {
	eva.mu.Lock()
	defer eva.mu.Unlock()
	for _, event := range eva.events {
		if event.UseInterval != nil && *event.UseInterval && event.IntervalSeconds > 0 {
			eva.wg.Add(1)
			go func(ev *EvaEvent) {
				defer eva.wg.Done()
				ticker := time.NewTicker(time.Duration(ev.IntervalSeconds) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-eva.ctx.Done():
						return
					case <-ticker.C:
						eva.acapp.SendPlatformEvent(ev.EventId, func() (*axevent.AXEvent, error) {
							return ev.PlatformEvent.NewEvent(ev.BuildKeyValueMap())
						})
					}
				}
			}(event)
		}
	}
}

func (eva *EvaApplication) StopSimulation() {
	eva.mu.Lock()
	if !eva.simRunning {
		eva.mu.Unlock()
		return
	}
	eva.simRunning = false
	eva.mu.Unlock()

	eva.cancel()
	eva.wg.Wait()
}
