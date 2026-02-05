const BASE = import.meta.env.DEV ? '/api' : ''

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  const data = await res.json()
  if (!res.ok) {
    throw new ApiError(res.status, data.error ?? `Request failed (${res.status})`)
  }
  return data as T
}

export interface DataField {
  name: string
  value: any
  value_type: 'string' | 'int' | 'float' | 'bool'
  use_random: boolean
  int_rand_start: number
  int_rand_end: number
  float_rand_start: number
  float_rand_end: number
  random_strings: string[]
}

export interface EvaEvent {
  ID: number
  name: string
  use_interval: boolean
  interval_seconds: number
  stateless: boolean
  DataFields: DataField[]
}

export interface SimulationStatus {
  running: boolean
  event_count: number
}

export const api = {
  getEvents: () => request<EvaEvent[]>('/events'),
  getEvent: (id: number) => request<EvaEvent>(`/events/${id}`),
  createEvent: (event: Partial<EvaEvent>) =>
    request<EvaEvent>('/events', { method: 'POST', body: JSON.stringify(event) }),
  updateEvent: (id: number, event: Partial<EvaEvent>) =>
    request<EvaEvent>(`/events/${id}`, { method: 'PUT', body: JSON.stringify(event) }),
  deleteEvent: (id: number) =>
    request<{ status: string }>(`/events/${id}`, { method: 'DELETE' }),
  triggerEvent: (id: number) =>
    request<{ status: string; event: string }>(`/events/${id}/trigger`, { method: 'POST' }),
  startSimulation: () =>
    request<{ status: string; event_count: number }>('/simulation/start', { method: 'POST' }),
  stopSimulation: () =>
    request<{ status: string }>('/simulation/stop', { method: 'POST' }),
  getSimulationStatus: () => request<SimulationStatus>('/simulation/status'),
}
