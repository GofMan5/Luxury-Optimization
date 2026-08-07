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

export interface BackgroundThresholds {
  medium_cpu_percent: number
  high_cpu_percent: number
  medium_io_mb_s: number
  high_io_mb_s: number
}

export interface BackgroundStartupLink {
  scope: string
  name: string
  state: string
}

export interface BackgroundServiceLink {
  name: string
  display_name: string
  system: boolean
  critical: boolean
  manageable: boolean
}

export interface BackgroundProcess {
  pid: number
  name: string
  executable?: string
  cpu_percent: number
  working_set_mb: number
  read_mb_s: number
  write_mb_s: number
  threads: number
  impact: 'low' | 'medium' | 'high'
  advice: 'observe' | 'review_startup' | 'review_service' | 'protected_service'
  startup: BackgroundStartupLink[]
  services: BackgroundServiceLink[]
}

export interface BackgroundReport {
  generated_at: string
  sample_ms: number
  logical_processors: number
  observed_processes: number
  measured_processes: number
  correlated_processes: number
  skipped_processes: number
  observed_cpu_percent: number
  read_mb_s: number
  write_mb_s: number
  thresholds: BackgroundThresholds
  processes: BackgroundProcess[]
  warnings?: string[]
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

export interface UDPLatencyReport extends LatencyReport {
  protocol: 'dns_rfc1035'
  question: string
}

export interface BufferbloatPhase {
  supported: boolean
  reason?: string
  bytes: number
  throughput_mbps: number
  latency: LatencyReport
  p95_increase_ms: number
  median_increase_ms: number
  rating?: 'low' | 'moderate' | 'high' | 'severe'
}

export interface BufferbloatReport {
  probe_address: string
  duration_ms: number
  streams: number
  baseline: LatencyReport
  download: BufferbloatPhase
  upload: BufferbloatPhase
  warnings?: string[]
}

export interface StorageVolume {
  path: string
  name?: string
  file_system: string
  kind: string
  total_bytes: number
  available_bytes: number
  read_only: boolean
}

export interface StorageVolumesReport {
  volumes: StorageVolume[]
  skipped: number
  warnings?: string[]
}

export interface StoragePathReport {
  path: string
  volume: StorageVolume
  size_bytes: number
  block_bytes: number
  buffered_write_mb_s: number
  durable_write_mb_s: number
  sync_ms: number
  buffered_read_mb_s: number
  sha256: string
  verified: boolean
  temporary_file_removed: boolean
}

export interface StorageScanStart {
  scan_id: string
  root: string
  started_at: string
	cached?: boolean
}

export type StorageScanState = 'scanning' | 'complete' | 'cancelled' | 'failed'

export interface StorageScanNode {
  id?: string
	deletable: boolean
  name: string
  kind: 'directory' | 'file' | 'other'
  size_bytes: number
  files: number
  directories: number
}

export interface StorageScanFile {
	id?: string
	deletable: boolean
	name: string
  relative_path: string
  extension: string
  size_bytes: number
}

export interface StorageDeletePreview {
	confirmation_token: string
	name: string
	kind: 'directory' | 'file'
	size_bytes: number
	files: number
	directories: number
	modified_at: string
	expires_at: string
	requires_typed_name: boolean
}

export interface StorageDeleteResult {
	deleted: boolean
	recycled: boolean
	name: string
	kind: 'directory' | 'file'
}

export interface StorageScanExtension {
  extension: string
  size_bytes: number
  files: number
}

export interface StorageScanReport {
  root: string
  volume: StorageVolume
  generated_at: string
  elapsed_ms: number
  total_bytes: number
  files: number
  directories: number
  skipped: number
  partial: boolean
  parent?: StorageScanNode
  children: StorageScanNode[]
  largest_files: StorageScanFile[]
  extensions: StorageScanExtension[]
  warnings?: string[]
}

export interface StorageScanStatus {
  scan_id: string
  state: StorageScanState
  root: string
  started_at: string
  elapsed_ms: number
  files_scanned: number
  directories_scanned: number
  bytes_scanned: number
  skipped: number
  current_path?: string
  error?: string
  report?: StorageScanReport
	cached?: boolean
}

export interface BenchmarkRun {
  average_fps: number
  one_percent_low_fps: number
  p95_frame_ms: number
}

export interface BenchmarkSet {
  label: string
  runs: BenchmarkRun[]
}

export interface MetricComparison {
  before_median: number
  after_median: number
  delta_percent: number
  noise_percent: number
  meaningful: boolean
}

export interface BenchmarkComparison {
  before_label: string
  after_label: string
  average_fps: MetricComparison
  one_percent_low_fps: MetricComparison
  p95_frame_ms: MetricComparison
  verdict: 'measurably_improved' | 'measurably_regressed' | 'mixed_result' | 'within_run_to_run_variance'
}

export interface GameInstall {
  source: string
  id: string
  name: string
  install_dir: string
  executables?: string[]
}

export interface GamesReport {
  games: GameInstall[]
  warnings?: string[]
}

export interface SavedGame {
  id: string
  name: string
  path: string
  profile: 'lite' | 'medium' | 'max' | 'recommended' | 'maximum'
  priority: 'normal' | 'above-normal' | 'high'
  affinity?: number
  args?: string[]
}

export interface SavedGames {
  version: number
  games: SavedGame[]
}

export interface GameLaunchResult {
  pid: number
  name: string
  launch_id?: string
  started_at?: string
  warning?: string
}

export interface GameLaunchRecord {
  id: string
  game_id: string
  game_name: string
  started_at: string
  launcher_pid: number
  profile: string
  priority: string
  affinity?: number
}

export interface GameBenchmarkAttachment {
  id: string
  game_id: string
  created_at: string
  before: BenchmarkSet
  after: BenchmarkSet
  comparison: BenchmarkComparison
}

export interface GameHistoryReport {
  game_id: string
  launches: GameLaunchRecord[]
  benchmarks: GameBenchmarkAttachment[]
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
