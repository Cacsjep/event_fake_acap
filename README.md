# Eva - Event Faker for ACAP

Eva lets you create fake events on an Axis camera. You define events (with whatever data fields you want), and Eva declares them on the platform and fires them on a schedule or manually. Handy for testing action rules, VMS integrations, or anything that reacts to camera events without needing the real analytics to be running.

It ships as an ACAP, runs a small web UI on port **8746**, and stores everything in a local SQLite database that survives restarts.

## How it works

1. **On startup** Eva loads all saved events from the database and registers them with the Axis event system right away. Other applications and the camera's own rule engine can see them immediately.
2. **CRUD through the API or UI** - when you create, update, or delete an event, it's registered/re-registered/unregistered on the platform in real time. No restart needed.
3. **Simulation** - hit Start and every event that has an interval configured will fire repeatedly on that cadence. Stop ends the tickers but keeps the events registered.
4. **Manual trigger** - fire any single event on demand via the UI or API, regardless of whether the simulation is running.

Events can be **stateful** (have an active/inactive state the platform tracks) or **stateless** (fire-and-forget). Each event carries data fields you define - strings, ints, floats, bools - with optional randomization so the values look realistic.

## Demo events

On first launch (empty database), Eva seeds 10 events inspired by Axis Object Analytics:

| Event | Type | Interval | What it simulates |
|---|---|---|---|
| Object Count In Area | stateless | 5s | Object counts with random totals (0-25) and type (Person/Vehicle/Unknown) |
| Line Crossing Count | stateless | 8s | In/out crossing counters with random counts |
| Person Detection | stateful | 3s | Active/inactive with confidence score (0.5-1.0) |
| Vehicle Detection | stateful | 4s | Active/inactive with vehicle type (Car/Truck/Bus/Motorcycle/Bicycle) |
| Object Classification | stateless | 5s | Class label (Human/Vehicle/Animal/Unknown) with confidence and object ID |
| Motion Detection | stateful | 2s | Active/inactive with motion level (0-100) |
| Loitering Detection | stateful | 10s | Active/inactive with duration (30-600s) and object type |
| Area Occupancy | stateless | 5s | Occupancy count and percentage across zones |
| Speed Estimation | stateless | 3s | Speed in km/h (5-120) for Person/Vehicle/Bicycle |
| Crossline Detection | stateful | 6s | Active/inactive with direction (Left to Right / Right to Left) |

These are just starting points - edit or delete them, add your own.

## Project structure

```
event_fake_acap/
  eva/                    # ACAP backend (Go)
    main.go               # Entry point
    eva.go                # App lifecycle, routes, simulation, registration
    event.go              # Event model, platform setup, demo seeding
    utils.go              # Helpers (sanitize, random generators)
    manifest.json         # ACAP package manifest
    Makefile              # Build targets (goxisbuilder)
    localdata/            # SQLite database (created at runtime)
    html/                 # Built frontend (served by the ACAP)
  frontend/               # Web UI (Vue 3 + Vuetify)
    src/
      App.vue             # Main UI - event cards, create/edit dialog, sim controls
      api.ts              # Typed HTTP client for the backend API
      main.ts             # Vue app bootstrap
      plugins/            # Vuetify plugin setup
```

## Tech stack

**Backend:** Go 1.25, [goxis](https://github.com/Cacsjep/goxis) (ACAP SDK bindings), [Fiber v3](https://gofiber.io/) (HTTP), [GORM](https://gorm.io/) + SQLite (persistence)

**Frontend:** Vue 3 (Composition API, `<script setup>`), Vuetify 3, Vite 7, TypeScript

## REST API

All endpoints return JSON.

### Events

| Method | Path | Description |
|---|---|---|
| `GET` | `/events` | List all events |
| `GET` | `/events/:id` | Get a single event |
| `POST` | `/events` | Create an event (also registers it on the platform) |
| `PUT` | `/events/:id` | Update an event (re-registers on the platform) |
| `DELETE` | `/events/:id` | Delete an event (unregisters from the platform) |
| `POST` | `/events/:id/trigger` | Fire a single event immediately |

Create/update/delete return **409** if the simulation is running.

### Simulation

| Method | Path | Description |
|---|---|---|
| `POST` | `/simulation/start` | Start firing all interval-based events |
| `POST` | `/simulation/stop` | Stop the simulation |
| `GET` | `/simulation/status` | Check if simulation is running and event count |

### Event payload shape

```json
{
  "name": "Person Detection",
  "use_interval": true,
  "interval_seconds": 3,
  "stateless": false,
  "DataFields": [
    {
      "name": "Active",
      "value": true,
      "value_type": "bool",
      "use_random": true,
      "int_rand_start": 0,
      "int_rand_end": 0,
      "float_rand_start": 0,
      "float_rand_end": 0,
      "random_strings": []
    },
    {
      "name": "Confidence",
      "value": 0.85,
      "value_type": "float",
      "use_random": true,
      "float_rand_start": 0.5,
      "float_rand_end": 1.0,
      "int_rand_start": 0,
      "int_rand_end": 0,
      "random_strings": []
    }
  ]
}
```

**Supported `value_type`s:** `string`, `int`, `float`, `bool`

When `use_random` is `true`:
- **int** - random value between `int_rand_start` and `int_rand_end`
- **float** - random value between `float_rand_start` and `float_rand_end`
- **string** - random pick from `random_strings` array
- **bool** - coin flip

## Point it at your camera

Two files need your camera's IP before you can develop:

**`eva/Makefile`** -- goxisbuilder uses `-ip` and `-pwd` to deploy the ACAP straight to your camera over SSH. Change them to match your device:

```makefile
# in eva/Makefile
goxisbuilder -install -ip 10.0.0.48 -sdk "12.5.0" -pwd 1qay2wsx ...
#                         ^^^^^^^^^^                    ^^^^^^^^
#                         your camera IP                your root password
```

**`frontend/vite.config.ts`** -- the Vite dev server proxies `/api` requests to Eva running on the camera, so you get hot reload on the frontend while it talks to the real backend. Update the `target` to the same IP:

```ts
proxy: {
  '/api': {
    target: 'http://10.0.0.48:8746',  // <-- your camera IP
    ...
  },
},
```

## Development setup

### Prerequisites

- [goxisbuilder](https://github.com/Cacsjep/goxis) installed
- Docker (goxisbuilder uses it for cross-compilation)
- Node.js 24+ (for the frontend)

### Frontend

```bash
cd frontend
npm install
npm run build    # builds into eva/html/
npm run dev      # or run the Vite dev server with hot reload (proxies API to camera)
```

### Backend (ACAP)

```bash
cd eva

# Development - watches for changes, auto-builds and deploys to camera
make dev

# One-off production build and deploy
make build
```

Both Makefile targets use `goxisbuilder` which spins up an SDK Docker container, cross-compiles for the camera architecture, packages the `.eap`, and installs it on the device via SSH.

### CI

Push a `v*` tag and the GitHub Actions workflow builds `.eap` packages for **aarch64** and **armv7hf**, then creates a release with both artifacts zipped up.

## How events appear on the camera

Each event is registered under the application namespace `eva` with a sanitized name (lowercased, spaces removed). In the camera's event system they show up as:

```
Eva - Person Detection
Eva - Object Count In Area
Eva - Motion Detection
...
```

You can use them in the camera's action rule engine, subscribe to them via ONVIF, or consume them from a VMS - just like real analytics events.

## License

MIT - see [LICENSE](eva/LICENSE).
