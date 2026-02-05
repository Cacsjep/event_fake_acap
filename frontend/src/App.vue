<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api, type EvaEvent, type DataField, type SimulationStatus } from './api'

const events = ref<EvaEvent[]>([])
const simStatus = ref<SimulationStatus>({ running: false, event_count: 0 })
const loading = ref(false)
const showDialog = ref(false)
const editingEvent = ref<Partial<EvaEvent> | null>(null)
const editingId = ref<number | null>(null)

// Snackbar toast
const snackbar = ref(false)
const snackbarText = ref('')
const snackbarColor = ref('success')

function toast(msg: string, color = 'success') {
  snackbarText.value = msg
  snackbarColor.value = color
  snackbar.value = true
}

function toastError(err: unknown) {
  const msg = err instanceof Error ? err.message : String(err)
  toast(msg, 'error')
}

// Form validation
const formValid = computed(() => {
  if (!editingEvent.value) return false
  if (!editingEvent.value.name?.trim()) return false
  if (editingEvent.value.use_interval && (!editingEvent.value.interval_seconds || editingEvent.value.interval_seconds < 1)) return false
  if (editingEvent.value.DataFields) {
    for (const f of editingEvent.value.DataFields) {
      if (!f.name?.trim()) return false
      if (f.use_random && f.value_type === 'int' && f.int_rand_start >= f.int_rand_end) return false
      if (f.use_random && f.value_type === 'float' && f.float_rand_start >= f.float_rand_end) return false
      if (f.use_random && f.value_type === 'string' && f.random_strings.length === 0) return false
    }
  }
  return true
})

const nameRules = [(v: string) => !!v?.trim() || 'Name is required']
const intervalRules = [
  (v: number) => (v !== undefined && v !== null) || 'Required',
  (v: number) => v >= 1 || 'Must be at least 1 second',
]
const fieldNameRules = [(v: string) => !!v?.trim() || 'Name is required']

const defaultDataField = (): DataField => ({
  name: '',
  value: '',
  value_type: 'string',
  use_random: false,
  int_rand_start: 0,
  int_rand_end: 100,
  float_rand_start: 0.0,
  float_rand_end: 1.0,
  random_strings: [],
})

const defaultEvent = (): Partial<EvaEvent> => ({
  name: '',
  use_interval: true,
  interval_seconds: 5,
  stateless: true,
  DataFields: [],
})

async function fetchAll() {
  loading.value = true
  try {
    const [evts, status] = await Promise.all([api.getEvents(), api.getSimulationStatus()])
    events.value = evts ?? []
    simStatus.value = status
  } catch (err) {
    toastError(err)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  editingEvent.value = defaultEvent()
  showDialog.value = true
}

function openEdit(ev: EvaEvent) {
  editingId.value = ev.ID
  editingEvent.value = { ...ev, DataFields: ev.DataFields ? ev.DataFields.map((f) => ({ ...f })) : [] }
  showDialog.value = true
}

function addField() {
  if (!editingEvent.value) return
  if (!editingEvent.value.DataFields) editingEvent.value.DataFields = []
  editingEvent.value.DataFields.push(defaultDataField())
}

function removeField(idx: number) {
  editingEvent.value?.DataFields?.splice(idx, 1)
}

function getRandomStringsStr(field: DataField): string {
  return (field.random_strings ?? []).join(', ')
}
function setRandomStringsStr(field: DataField, val: string) {
  field.random_strings = val
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

async function saveEvent() {
  if (!editingEvent.value || !formValid.value) return
  try {
    if (editingId.value) {
      await api.updateEvent(editingId.value, editingEvent.value)
      toast('Event updated')
    } else {
      await api.createEvent(editingEvent.value)
      toast('Event created')
    }
    showDialog.value = false
    await fetchAll()
  } catch (err) {
    toastError(err)
  }
}

async function deleteEvent(id: number) {
  try {
    await api.deleteEvent(id)
    toast('Event deleted')
    await fetchAll()
  } catch (err) {
    toastError(err)
  }
}

async function triggerEvent(id: number) {
  try {
    const res = await api.triggerEvent(id)
    toast(`Triggered: ${res.event}`)
  } catch (err) {
    toastError(err)
  }
}

async function toggleSimulation() {
  try {
    if (simStatus.value.running) {
      await api.stopSimulation()
      toast('Simulation stopped')
    } else {
      const res = await api.startSimulation()
      toast(`Simulation started (${res.event_count} events)`)
    }
    await fetchAll()
  } catch (err) {
    toastError(err)
  }
}

onMounted(fetchAll)
</script>

<template>
  <v-app>
    <v-app-bar density="compact" color="surface">
      <v-app-bar-title class="text-primary font-weight-bold">EVA - Event Simulator</v-app-bar-title>
      <template #append>
        <v-chip :color="simStatus.running ? 'success' : 'grey'" class="mr-2" variant="elevated">
          {{ simStatus.running ? `Running (${simStatus.event_count})` : 'Stopped' }}
        </v-chip>
        <v-btn
          :color="simStatus.running ? 'error' : 'success'"
          variant="elevated"
          size="small"
          @click="toggleSimulation"
        >
          {{ simStatus.running ? 'Stop' : 'Start' }} Simulation
        </v-btn>
      </template>
    </v-app-bar>

    <v-main>
      <v-container>
        <v-row>
          <v-col cols="12" class="d-flex align-center justify-space-between">
            <h2>Events</h2>
            <v-btn color="primary" prepend-icon="mdi-plus" @click="openCreate">New Event</v-btn>
          </v-col>
        </v-row>

        <v-row v-if="loading">
          <v-col cols="12" class="text-center">
            <v-progress-circular indeterminate color="primary" />
          </v-col>
        </v-row>

        <v-row v-else-if="events.length === 0">
          <v-col cols="12">
            <v-alert type="warning" variant="tonal">No events configured yet.</v-alert>
          </v-col>
        </v-row>

        <v-row v-else>
          <v-col v-for="ev in events" :key="ev.ID" cols="12" md="6" lg="4">
            <v-card>
              <v-card-title>{{ ev.name }}</v-card-title>
              <v-card-subtitle>
                <v-chip size="x-small" class="mr-1" color="primary" variant="outlined">{{ ev.stateless ? 'Stateless' : 'Stateful' }}</v-chip>
                <v-chip v-if="ev.use_interval" size="x-small" color="secondary" variant="outlined">
                  Every {{ ev.interval_seconds }}s
                </v-chip>
              </v-card-subtitle>
              <v-card-text v-if="ev.DataFields && ev.DataFields.length > 0">
                <v-table density="compact" class="bg-transparent">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Type</th>
                      <th>Random</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(f, i) in ev.DataFields" :key="i">
                      <td>{{ f.name }}</td>
                      <td>{{ f.value_type }}</td>
                      <td>
                        <v-icon :color="f.use_random ? 'success' : 'grey'" size="small">
                          {{ f.use_random ? 'mdi-check' : 'mdi-close' }}
                        </v-icon>
                      </td>
                    </tr>
                  </tbody>
                </v-table>
              </v-card-text>
              <v-card-actions>
                <v-btn size="small" color="primary" variant="text" @click="triggerEvent(ev.ID)">
                  <v-icon start>mdi-flash</v-icon>Trigger
                </v-btn>
                <v-spacer />
                <v-btn size="small" icon="mdi-pencil" variant="text" @click="openEdit(ev)" />
                <v-btn size="small" icon="mdi-delete" variant="text" color="error" @click="deleteEvent(ev.ID)" />
              </v-card-actions>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </v-main>

    <!-- Create / Edit Dialog -->
    <v-dialog v-model="showDialog" max-width="650" persistent>
      <v-card v-if="editingEvent">
        <v-card-title>{{ editingId ? 'Edit' : 'Create' }} Event</v-card-title>
        <v-card-text>
          <!-- Event config -->
          <v-card flat color="rgba(255,255,255,0.03)" class="pa-4 mb-4">
            <div class="text-caption text-medium-emphasis mb-1">Event Name</div>
            <v-text-field v-model="editingEvent.name" variant="filled" density="compact" :rules="nameRules" hide-details="auto" placeholder="e.g. Motion Detected" />

            <v-row dense class="mt-3">
              <v-col cols="6">
                <v-checkbox v-model="editingEvent.stateless" label="Stateless" density="compact" color="primary" hide-details />
              </v-col>
              <v-col cols="6">
                <v-checkbox v-model="editingEvent.use_interval" label="Send on interval" density="compact" color="primary" hide-details />
              </v-col>
            </v-row>

            <template v-if="editingEvent.use_interval">
              <div class="text-caption text-medium-emphasis mt-3 mb-1">Interval (seconds)</div>
              <v-text-field v-model.number="editingEvent.interval_seconds" variant="filled" density="compact" type="number" :rules="intervalRules" min="1" hide-details="auto" />
            </template>
          </v-card>

          <!-- Data fields -->
          <div class="d-flex align-center mb-2">
            <span class="text-caption text-medium-emphasis">DATA FIELDS</span>
            <v-spacer />
            <v-btn size="x-small" color="primary" variant="tonal" @click="addField" prepend-icon="mdi-plus">Add</v-btn>
          </div>

          <v-card
            v-for="(field, idx) in editingEvent.DataFields"
            :key="idx"
            flat
            color="rgba(255,255,255,0.03)"
            class="pa-4 mb-2"
          >
            <div class="d-flex align-center mb-2">
              <v-chip size="x-small" color="primary" variant="tonal" label>#{{ idx + 1 }}</v-chip>
              <v-spacer />
              <v-btn icon="mdi-delete-outline" size="x-small" variant="text" color="error" @click="removeField(idx)" />
            </div>

            <v-row dense>
              <v-col cols="6">
                <div class="text-caption text-medium-emphasis mb-1">Field Name</div>
                <v-text-field v-model="field.name" variant="filled" density="compact" :rules="fieldNameRules" hide-details="auto" placeholder="e.g. Temperature" />
              </v-col>
              <v-col cols="3">
                <div class="text-caption text-medium-emphasis mb-1">Type</div>
                <v-select v-model="field.value_type" :items="['string', 'int', 'float', 'bool']" variant="filled" density="compact" hide-details />
              </v-col>
              <v-col cols="3">
                <div class="text-caption text-medium-emphasis mb-1">Default</div>
                <v-text-field v-model="field.value" variant="filled" density="compact" hide-details />
              </v-col>
            </v-row>

            <v-checkbox v-model="field.use_random" label="Use random values" density="compact" color="primary" hide-details class="mt-2" />

            <template v-if="field.use_random">
              <v-row v-if="field.value_type === 'int' || field.value_type === 'float'" dense class="mt-1">
                <v-col cols="6">
                  <div class="text-caption text-medium-emphasis mb-1">Min</div>
                  <v-text-field
                    v-if="field.value_type === 'int'"
                    v-model.number="field.int_rand_start"
                    variant="filled"
                    density="compact"
                    type="number"
                    hide-details
                  />
                  <v-text-field
                    v-else
                    v-model.number="field.float_rand_start"
                    variant="filled"
                    density="compact"
                    type="number"
                    step="0.01"
                    hide-details
                  />
                </v-col>
                <v-col cols="6">
                  <div class="text-caption text-medium-emphasis mb-1">Max</div>
                  <v-text-field
                    v-if="field.value_type === 'int'"
                    v-model.number="field.int_rand_end"
                    variant="filled"
                    density="compact"
                    type="number"
                    hide-details="auto"
                    :rules="[() => field.int_rand_end > field.int_rand_start || 'Max > Min']"
                  />
                  <v-text-field
                    v-else
                    v-model.number="field.float_rand_end"
                    variant="filled"
                    density="compact"
                    type="number"
                    step="0.01"
                    hide-details="auto"
                    :rules="[() => field.float_rand_end > field.float_rand_start || 'Max > Min']"
                  />
                </v-col>
              </v-row>
              <div v-if="field.value_type === 'string'" class="mt-1">
                <div class="text-caption text-medium-emphasis mb-1">Random Strings (comma separated)</div>
                <v-text-field
                  :model-value="getRandomStringsStr(field)"
                  @update:model-value="setRandomStringsStr(field, $event)"
                  variant="filled"
                  density="compact"
                  hide-details="auto"
                  placeholder="e.g. car, person, bike"
                  :rules="[() => field.random_strings.length > 0 || 'At least one string']"
                />
              </div>
            </template>
          </v-card>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showDialog = false">Cancel</v-btn>
          <v-btn color="primary" variant="elevated" :disabled="!formValid" @click="saveEvent">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Toast snackbar -->
    <v-snackbar v-model="snackbar" :color="snackbarColor" :timeout="3000" location="bottom right">
      {{ snackbarText }}
      <template #actions>
        <v-btn variant="text" @click="snackbar = false">Close</v-btn>
      </template>
    </v-snackbar>
  </v-app>
</template>
