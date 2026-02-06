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
  if (editingEvent.value.use_interval) {
    if (editingEvent.value.use_random_interval) {
      if (!editingEvent.value.interval_min_seconds || editingEvent.value.interval_min_seconds < 1) return false
      if (!editingEvent.value.interval_max_seconds || editingEvent.value.interval_max_seconds <= editingEvent.value.interval_min_seconds) return false
    } else {
      if (!editingEvent.value.interval_seconds || editingEvent.value.interval_seconds < 1) return false
    }
  }
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
const intervalMinRules = [
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
  use_random_interval: false,
  interval_min_seconds: 1,
  interval_max_seconds: 10,
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
      <v-app-bar-title class="text-primary font-weight-bold">EVA - Event Virtualizer for ACAP</v-app-bar-title>
      <template #append>
        <v-chip size="small" label :color="simStatus.running ? 'success' : 'grey'" class="mr-2" variant="elevated">
          {{ simStatus.running ? `Running (${simStatus.event_count})` : 'Stopped' }}
        </v-chip>
        <v-btn
          :color="simStatus.running ? 'error' : 'success'"
          variant="outlined"
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
            <v-btn color="primary" size="small" prepend-icon="mdi-plus" @click="openCreate">New</v-btn>
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

        <v-card v-else flat>
          <v-table density="compact">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Interval</th>
                <th>Fields</th>
                <th class="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ev in events" :key="ev.ID">
                <td class="font-weight-medium">{{ ev.name }}</td>
                <td>
                  <v-chip size="x-small" :color="ev.stateless ? 'primary' : 'warning'" variant="outlined">{{ ev.stateless ? 'Stateless' : 'Stateful' }}</v-chip>
                </td>
                <td>
                  <v-chip v-if="ev.use_interval && ev.use_random_interval" size="x-small" color="secondary" variant="outlined">
                    {{ ev.interval_min_seconds }}-{{ ev.interval_max_seconds }}s (random)
                  </v-chip>
                  <v-chip v-else-if="ev.use_interval" size="x-small" color="secondary" variant="outlined">
                    {{ ev.interval_seconds }}s
                  </v-chip>
                  <span v-else class="text-medium-emphasis">-</span>
                </td>
                <td>
                  <v-chip v-for="(f, i) in (ev.DataFields ?? [])" :key="i" size="x-small" class="mr-1 mb-1" :color="f.use_random ? 'success' : 'default'" variant="tonal">
                    {{ f.name }} <span class="text-medium-emphasis ml-1">{{ f.value_type }}</span>
                  </v-chip>
                </td>
                <td class="text-right text-no-wrap">
                  <v-btn size="small" color="primary" variant="text" @click="triggerEvent(ev.ID)">
                    <v-icon start size="small">mdi-flash</v-icon>Trigger
                  </v-btn>
                  <v-btn size="small" icon="mdi-pencil" variant="text" @click="openEdit(ev)" />
                  <v-btn size="small" icon="mdi-delete" variant="text" color="error" @click="deleteEvent(ev.ID)" />
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </v-container>
    </v-main>

    <!-- Create / Edit Dialog -->
    <v-dialog v-model="showDialog" max-width="1050" persistent>
      <v-card v-if="editingEvent" color="#121212">
        <v-card-title class="d-flex align-center pb-0">
          {{ editingId ? 'Edit' : 'Create' }} Event
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" size="small" @click="showDialog = false" />
        </v-card-title>
        <v-card-text class="pt-4">
          <!-- Row 1: Name + Stateless -->
          <v-row dense>
            <v-col>
              <v-text-field v-model="editingEvent.name" variant="solo-filled" density="compact" :rules="nameRules" hide-details="auto" label="Event Name" placeholder="e.g. Motion Detected" />
            </v-col>
            <v-col cols="auto" class="d-flex align-center">
              <v-checkbox v-model="editingEvent.stateless" label="Stateless" density="compact" color="primary" hide-details />
            </v-col>
          </v-row>
          <!-- Row 2: Interval settings - all inline -->
          <v-row dense class="mt-2 align-center">
            <v-col cols="auto">
              <v-checkbox v-model="editingEvent.use_interval" label="Interval" density="compact" color="primary" hide-details />
            </v-col>
            <template v-if="editingEvent.use_interval">
              <v-col cols="auto">
                <v-checkbox v-model="editingEvent.use_random_interval" label="Random" density="compact" color="primary" hide-details />
              </v-col>
              <template v-if="editingEvent.use_random_interval">
                <v-col cols="2">
                  <v-text-field v-model.number="editingEvent.interval_min_seconds" variant="solo-filled" density="compact" type="number" :rules="intervalMinRules" min="1" hide-details="auto">
                    <template #append-inner><v-chip size="x-small" label>min-sec</v-chip></template>
                  </v-text-field>
                </v-col>
                <v-col cols="2">
                  <v-text-field v-model.number="editingEvent.interval_max_seconds" variant="solo-filled" density="compact" type="number" min="2" hide-details="auto"
                    :rules="[() => (editingEvent?.interval_max_seconds ?? 0) > (editingEvent?.interval_min_seconds ?? 0) || 'Max > Min']"
                  >
                    <template #append-inner><v-chip size="x-small" label>max-sec</v-chip></template>
                  </v-text-field>
                </v-col>
              </template>
              <v-col v-else cols="2">
                <v-text-field v-model.number="editingEvent.interval_seconds" variant="solo-filled" density="compact" type="number" :rules="intervalRules" min="1" hide-details="auto">
                  <template #append-inner><v-chip size="x-small" label>sec</v-chip></template>
                </v-text-field>
              </v-col>
            </template>
          </v-row>
          <div class="text-caption text-medium-emphasis mt-1">Automatically send the event at a fixed or random interval. Random picks a new delay each time between min and max.</div>

          <v-divider class="my-3" />

          <!-- Data fields header -->
          <div class="d-flex align-center mb-2">
            <span class="text-body-2 font-weight-medium">Data Fields</span>
            <v-spacer />
            <v-btn size="x-small" color="primary" variant="tonal" @click="addField" prepend-icon="mdi-plus">Add</v-btn>
          </div>
          <div class="text-caption text-medium-emphasis mb-2">Key-value pairs sent with each event. Enable Rng to randomize a field's value on each send.</div>

          <!-- Data field column headers -->
          <template v-if="editingEvent.DataFields && editingEvent.DataFields.length > 0">
            <v-row dense class="mb-1">
              <v-col cols="3"><span class="text-caption text-medium-emphasis">Name</span></v-col>
              <v-col cols="2"><span class="text-caption text-medium-emphasis">Type</span></v-col>
              <v-col cols="2"><span class="text-caption text-medium-emphasis">Default</span></v-col>
              <v-col cols="auto" style="width:40px"><span class="text-caption text-medium-emphasis">Rng</span></v-col>
              <v-col><span class="text-caption text-medium-emphasis">Range / Values</span></v-col>
              <v-col cols="auto" style="width:36px"></v-col>
            </v-row>
            <v-row v-for="(field, idx) in editingEvent.DataFields" :key="idx" dense class="align-center mb-1">
              <v-col cols="3">
                <v-text-field v-model="field.name" variant="solo-filled" density="compact" :rules="fieldNameRules" hide-details="auto" placeholder="Field name" />
              </v-col>
              <v-col cols="2">
                <v-select v-model="field.value_type" :items="['string', 'int', 'float', 'bool']" variant="solo-filled" density="compact" hide-details />
              </v-col>
              <v-col cols="2">
                <v-text-field v-model="field.value" variant="solo-filled" density="compact" hide-details placeholder="-" />
              </v-col>
              <v-col cols="auto" style="width:40px" class="d-flex justify-center">
                <v-checkbox-btn v-model="field.use_random" color="primary" density="compact" />
              </v-col>
              <v-col>
                <template v-if="field.use_random">
                  <v-row v-if="field.value_type === 'int'" dense class="align-center">
                    <v-col>
                      <v-text-field v-model.number="field.int_rand_start" variant="solo-filled" density="compact" type="number" hide-details placeholder="min" />
                    </v-col>
                    <v-col>
                      <v-text-field v-model.number="field.int_rand_end" variant="solo-filled" density="compact" type="number" hide-details="auto" placeholder="max"
                        :rules="[() => field.int_rand_end > field.int_rand_start || 'Max>Min']"
                      />
                    </v-col>
                  </v-row>
                  <v-row v-else-if="field.value_type === 'float'" dense class="align-center">
                    <v-col>
                      <v-text-field v-model.number="field.float_rand_start" variant="solo-filled" density="compact" type="number" step="0.01" hide-details placeholder="min" />
                    </v-col>
                    <v-col>
                      <v-text-field v-model.number="field.float_rand_end" variant="solo-filled" density="compact" type="number" step="0.01" hide-details="auto" placeholder="max"
                        :rules="[() => field.float_rand_end > field.float_rand_start || 'Max>Min']"
                      />
                    </v-col>
                  </v-row>
                  <v-text-field v-else-if="field.value_type === 'string'"
                    :model-value="getRandomStringsStr(field)"
                    @update:model-value="setRandomStringsStr(field, $event)"
                    variant="solo-filled" density="compact" hide-details="auto" placeholder="a, b, c"
                    :rules="[() => field.random_strings.length > 0 || 'Need values']"
                  />
                  <span v-else class="text-caption text-medium-emphasis">50/50</span>
                </template>
              </v-col>
              <v-col cols="auto" style="width:36px">
                <v-btn icon="mdi-close" size="x-small" variant="text" color="error" @click="removeField(idx)" />
              </v-col>
            </v-row>
          </template>
          <div v-else class="text-caption text-medium-emphasis text-center py-3">No data fields. Click Add to create one.</div>
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
