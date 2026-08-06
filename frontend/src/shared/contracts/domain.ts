export interface OSInfo {
  caption: string
  version: string
  build_number: string
  architecture: string
}

export interface Hardware {
  os: OSInfo
  cpus: Array<{ name: string; manufacturer: string; cores: number; logical_processors: number }>
  gpus: Array<{ name: string; pnp_device_id: string; driver_version: string; vendor: string }>
  has_battery: boolean
  go_arch: string
}

export interface Capability {
  id: string
  available: boolean
  mode: string
  detail: string
}

export interface Finding {
  id: string
  title: string
  evidence: string
  action: string
}

export interface Audit {
  generated_at: string
  version: string
  hardware: Hardware
  administrator: boolean
  active_power_guid?: string
  capabilities: Capability[]
  optimization_findings: Finding[]
  warnings?: string[]
}

export interface PlanItem {
  id: string
  category: string
  name: string
  current: string
  desired: string
  changed: boolean
  effect: string
  benefit: string
  risk: string
  risk_level: 'low' | 'medium' | 'high'
  reversible: boolean
  manual_available: boolean
  restore_available: boolean
}

export interface Plan {
  profile: { id: string; name: string; description: string; performance_plan: boolean; network_latency: boolean }
  hardware: Hardware
  items: PlanItem[]
  warnings?: string[]
}

export interface MutationResult {
  changed: boolean
  message: string
  artifact?: string
}

export interface CheckpointStatus {
  ready: boolean
  profile: string
  created_at?: string
  expires_at?: string
}

export interface StartupEntry {
  scope: string
  name: string
  command: string
  state: string
}

export interface StartupReport {
  entries: StartupEntry[]
  warnings?: string[]
}

export interface ServiceEntry {
  name: string
  display_name: string
  state: string
  start_type: string
  process_id?: number
  binary_path?: string
  system: boolean
  description?: string
  dependencies?: string[]
  critical: boolean
  manageable: boolean
}

export interface ServicesReport {
  services: ServiceEntry[]
  skipped: number
}

export interface NetworkInterface {
  index: number
  name: string
  mtu: number
  flags: string
  addresses: string[]
}

export interface LatencyReport {
  address: string
  attempts: number
  succeeded: number
  failed: number
  min_ms: number
  median_ms: number
  p95_ms: number
  max_ms: number
  jitter_ms: number
  samples_ms: number[]
}

export interface BackupSummary {
  id: string
  created_at: string
  profile: string
  tweak_id?: string
  status: string
  restorable: boolean
}

export interface SystemRestorePoint {
  sequence_number: number
  description: string
  created_at: string
  restore_point_type: number
}

export interface UpdateStatus {
  last_check?: string
  channel: string
  current: string
  latest?: string
  update_ready: boolean
}

export interface CleanResult {
  files: number
  dirs: number
  bytes: number
  skipped: number
}

export interface Handshake {
  product: string
  version: string
  protocol: number
  os: string
  arch: string
  methods: string[]
}
