import type { BackendClient } from './client'
import type { Audit, BackupSummary, BenchmarkComparison, BenchmarkSet, CheckpointStatus, GameBenchmarkAttachment, GameHistoryReport, GameLaunchRecord, GamesReport, Handshake, NetworkInterface, Plan, SavedGame, SavedGames, ServicesReport, StartupReport, SystemRestorePoint, UpdateStatus } from '../../shared/contracts/domain'

const hardware = {
  os: { caption: 'Windows 11 Pro', version: '24H2', build_number: '26100', architecture: '64-bit' },
  cpus: [{ name: 'AMD Ryzen 7 7800X3D', manufacturer: 'AMD', cores: 8, logical_processors: 16 }],
  gpus: [{ name: 'NVIDIA GeForce RTX 4070', pnp_device_id: 'PCI\\VEN_10DE', driver_version: 'preview', vendor: 'NVIDIA' }],
  has_battery: false,
  go_arch: 'amd64',
}

const audit: Audit = {
  generated_at: new Date().toISOString(),
  version: '1.0.4-preview',
  hardware,
  administrator: false,
  active_power_guid: 'Balanced',
  capabilities: [
    { id: 'persistent-profile', available: true, mode: 'reversible', detail: 'Backup · apply · read-back · rollback' },
    { id: 'game-boost', available: true, mode: 'session', detail: 'Non-elevated game with automatic restore' },
    { id: 'startup', available: true, mode: 'reversible', detail: 'Current-user startup entries' },
    { id: 'services', available: true, mode: 'read-only', detail: 'System service inventory' },
  ],
  optimization_findings: [{ id: 'lite-profile-drift', title: 'Lite profile differs', evidence: '2 of 6 settings differ', action: 'Review exact Lite plan' }],
}

const liteItems = [
  previewTweak('game-mode-enable', 'Gaming', 'Windows Game Mode', 'Enabled', 'Enabled', false, 'low'),
  previewTweak('game-mode-allow', 'Gaming', 'Automatic Game Mode', 'Enabled', 'Enabled', false, 'low'),
  previewTweak('ui-transparency', 'Desktop', 'Interface transparency', 'On', 'Off', true, 'low'),
  previewTweak('ui-taskbar-animation', 'Desktop', 'Taskbar animations', 'On', 'Off', true, 'low'),
  previewTweak('ui-window-animation', 'Desktop', 'Window animation', 'Off', 'Off', false, 'low'),
  previewTweak('menu-show-delay', 'Desktop', 'Menu opening delay', '400', '0', true, 'low'),
]
const mediumRegistryItems = [...liteItems,
  previewTweak('capture-game-dvr', 'Gaming', 'Background game capture', 'Enabled', 'Disabled', true),
  previewTweak('capture-app', 'Gaming', 'App capture', 'Enabled', 'Disabled', true),
  previewTweak('mouse-speed', 'Input', 'Mouse acceleration', 'On', 'Off', true, 'low'),
  previewTweak('mouse-threshold-1', 'Input', 'Mouse threshold 1', '6', '0', true, 'low'),
  previewTweak('mouse-threshold-2', 'Input', 'Mouse threshold 2', '10', '0', true, 'low'),
]
const mediumPowerItems = Array.from({ length: 11 }, (_, index) => previewTweak(`power-medium-${index + 1}`, 'Processor', `Reviewed CPU policy ${index + 1}`, 'Balanced', 'High Performance', index < 3, 'medium'))
const additionalMaxPowerItems = Array.from({ length: 96 }, (_, index) => previewTweak(`power-max-${index + 1}`, index < 10 ? 'Storage' : 'Processor', `Native Max policy ${index + 1}`, 'Balanced', 'High Performance', false, 'high'))
const ethernetItems = Array.from({ length: 4 }, (_, index) => previewTweak(`ethernet-preview-${index + 1}`, 'Ethernet', `Adapter policy ${index + 1}`, '1', '0', index === 0, 'medium'))

const litePlan: Plan = { profile: { id: 'lite', name: 'Lite', description: 'Six low-risk gaming and interface improvements.', performance_plan: false, network_latency: false }, hardware, items: liteItems }
const mediumPlan: Plan = { profile: { id: 'medium', name: 'Medium', description: 'Lite plus capture/input tuning and 11 reviewed CPU policies.', performance_plan: true, network_latency: false }, hardware, items: [...mediumRegistryItems, previewTweak('power-plan', 'Power', 'Medium power plan', 'Balanced', 'Luxury Medium', true, 'medium'), ...mediumPowerItems] }
const maxPlan: Plan = { profile: { id: 'max', name: 'Max', description: 'All supported reviewed native actions.', performance_plan: true, network_latency: true }, hardware, items: [...mediumRegistryItems, previewTweak('startup-delay', 'Desktop', 'Startup delay', '1000', '0', true), previewTweak('power-plan', 'Power', 'Max power plan', 'Balanced', 'Luxury Max', true, 'high'), ...mediumPowerItems, ...additionalMaxPowerItems, ...ethernetItems] }

export class PreviewBackendClient implements BackendClient {
  #checkpoints = new Set<string>()
  #tweakBackups = new Set<string>()
  #games: SavedGame[] = [{ id: previewGameID('C:\\Games\\Example Game\\game.exe'), name: 'Example Game', path: 'C:\\Games\\Example Game\\game.exe', profile: 'medium', priority: 'normal' }]
  #launches: GameLaunchRecord[] = []
  #benchmarks: GameBenchmarkAttachment[] = []

  async call<T>(method: string, payload?: unknown, signal?: AbortSignal): Promise<T> {
    await pause(90, signal)
    const body = (payload ?? {}) as Record<string, unknown>
    let result: unknown
    switch (method) {
      case 'system.handshake':
        result = { product: 'Luxury Optimization', version: '1.0.4-preview', protocol: 1, os: 'windows', arch: 'amd64', methods: [] } satisfies Handshake
        break
      case 'optimization.audit': result = audit; break
      case 'optimization.plan': result = previewPlan(body.profile === 'max' ? maxPlan : body.profile === 'medium' ? mediumPlan : litePlan, this.#tweakBackups); break
      case 'optimization.apply': result = { changed: true, message: 'Profile applied and verified.' }; break
      case 'optimization.apply_tweak': {
        const id = String(body.id ?? '')
        this.#tweakBackups.add(id)
        result = { changed: true, message: 'Tweak applied and verified.', artifact: '20260804T120000.000000000Z' }
        break
      }
      case 'optimization.restore_tweak': {
        this.#tweakBackups.delete(String(body.id ?? ''))
        result = { changed: true, message: 'Tweak restored and verified.' }
        break
      }
      case 'optimization.restore': result = { changed: true, message: 'Original state restored and verified.' }; break
      case 'optimization.checkpoint_status': {
        const profile = String(body.profile ?? 'lite')
        result = previewCheckpoint(profile, this.#checkpoints.has(profile))
        break
      }
      case 'optimization.create_checkpoint': {
        const profile = String(body.profile ?? 'lite')
        this.#checkpoints.add(profile)
        result = previewCheckpoint(profile, true)
        break
      }
      case 'startup.list': result = previewStartup; break
      case 'startup.set': result = { enabled: body.enabled }; break
      case 'services.list': result = previewServices; break
      case 'services.set': result = { changed: true, message: 'Service startup setting updated and verified.' }; break
      case 'network.interfaces': result = previewInterfaces; break
      case 'network.test': result = { address: body.address, attempts: body.count, succeeded: body.count, failed: 0, min_ms: 8.2, median_ms: 9.1, p95_ms: 10.4, max_ms: 10.4, jitter_ms: 0.7, samples_ms: [8.2, 9.1, 10.4] }; break
      case 'benchmark.compare': result = previewBenchmark(body.before as BenchmarkSet, body.after as BenchmarkSet); break
      case 'gaming.scan': result = previewGames; break
      case 'gaming.saved': result = { version: 1, games: this.#games } satisfies SavedGames; break
      case 'gaming.save': {
        const path = String(body.path ?? '')
        const game: SavedGame = { id: previewGameID(path), name: String(body.name || path.split(/[\\/]/).pop() || 'Game'), path, profile: (body.profile || 'medium') as SavedGame['profile'], priority: (body.priority || 'normal') as SavedGame['priority'], ...(body.affinity ? { affinity: Number.parseInt(String(body.affinity).replace(/^0x/i, ''), 16) } : {}), ...(Array.isArray(body.args) ? { args: body.args.map(String) } : {}) }
        this.#games = [...this.#games.filter((item) => item.id !== game.id), game]
        result = game
        break
      }
      case 'gaming.remove': this.#games = this.#games.filter((game) => game.id !== body.id); result = { changed: true, message: 'Game profile removed.' }; break
      case 'gaming.launch': {
        const game = this.#games.find((item) => item.id === body.id)
        if (!game) throw new Error('Game profile not found')
        const started = new Date().toISOString()
        const launch: GameLaunchRecord = { id: previewHistoryID(this.#launches.length + 1), game_id: game.id, game_name: game.name, started_at: started, launcher_pid: 4242, profile: game.profile, priority: game.priority, ...(game.affinity ? { affinity: game.affinity } : {}) }
        this.#launches = [launch, ...this.#launches]
        result = { pid: launch.launcher_pid, name: game.name, launch_id: launch.id, started_at: started }
        break
      }
      case 'gaming.history': result = { game_id: String(body.id), launches: this.#launches.filter((item) => item.game_id === body.id), benchmarks: this.#benchmarks.filter((item) => item.game_id === body.id) } satisfies GameHistoryReport; break
      case 'gaming.attach_benchmark': {
        const before = body.before as BenchmarkSet
        const after = body.after as BenchmarkSet
        const attachment: GameBenchmarkAttachment = { id: previewHistoryID(1000 + this.#benchmarks.length), game_id: String(body.game_id), created_at: new Date().toISOString(), before, after, comparison: previewBenchmark(before, after) }
        this.#benchmarks = [attachment, ...this.#benchmarks]
        result = attachment
        break
      }
      case 'backups.list': result = previewBackups; break
      case 'restore.system_points': result = previewSystemPoints; break
      case 'restore.open_system': result = { changed: false, message: 'Windows System Restore opened.' }; break
      case 'cleanup.run': result = { files: 18, dirs: 2, bytes: 14_680_064, skipped: 1 }; break
      case 'updates.status': result = updateStatus(); break
      case 'updates.check': result = { ...updateStatus(), latest: 'v1.0.4', update_ready: false }; break
      case 'updates.install': result = { changed: false, message: 'Installed version is current.' }; break
      default: throw new Error(`Preview backend does not implement ${method}`)
    }
    return result as T
  }

  async stop(): Promise<void> {}

  invalidate(): void {}
}

const previewStartup: StartupReport = { entries: [{ scope: 'HKCU', name: 'Steam', command: 'steam.exe -silent', state: 'present' }, { scope: 'HKCU', name: 'Discord', command: 'Update.exe --processStart Discord.exe', state: 'disabled_by_luxury_optimization' }] }
const previewServices: ServicesReport = { services: [{ name: 'BFE', display_name: 'Base Filtering Engine', description: 'Manages firewall and Internet Protocol security policies.', dependencies: [], state: 'running', start_type: 'automatic', process_id: 1024, system: true, critical: true, manageable: false }, { name: 'mpssvc', display_name: 'Windows Defender Firewall', description: 'Helps protect the computer by preventing unauthorized network access.', dependencies: ['BFE'], state: 'running', start_type: 'automatic', process_id: 1024, system: true, critical: true, manageable: false }, { name: 'VendorAgent', display_name: 'Vendor Update Agent', description: 'Checks for optional vendor software updates.', dependencies: [], state: 'stopped', start_type: 'manual', system: false, critical: false, manageable: true }], skipped: 0 }
const previewInterfaces: NetworkInterface[] = [{ index: 4, name: 'Ethernet', mtu: 1500, flags: 'up|broadcast|multicast', addresses: ['192.168.1.20/24'] }]
const previewGames: GamesReport = { games: [{ source: 'Steam', id: '730', name: 'Example Game', install_dir: 'C:\\Games\\Example Game', executables: ['C:\\Games\\Example Game\\game.exe'] }] }
const previewBackups: BackupSummary[] = [{ id: '20260804T120000.000000000Z', created_at: new Date().toISOString(), profile: 'lite', status: 'applied', restorable: true }]
const previewSystemPoints: SystemRestorePoint[] = [{ sequence_number: 42, description: 'Before driver update', created_at: new Date(Date.now() - 86_400_000).toISOString(), restore_point_type: 0 }]
function updateStatus(): UpdateStatus {
  return { last_check: new Date().toISOString(), channel: '1.0', current: '1.0.4-preview', update_ready: false }
}

function previewBenchmark(before: BenchmarkSet, after: BenchmarkSet): BenchmarkComparison {
  return {
    before_label: before.label,
    after_label: after.label,
    average_fps: { before_median: 144, after_median: 151, delta_percent: 4.86, noise_percent: 2, meaningful: true },
    one_percent_low_fps: { before_median: 100, after_median: 108, delta_percent: 8, noise_percent: 2, meaningful: true },
    p95_frame_ms: { before_median: 8.5, after_median: 7.9, delta_percent: 7.06, noise_percent: 2, meaningful: true },
    verdict: 'measurably_improved',
  }
}

function previewCheckpoint(profile: string, ready = false): CheckpointStatus {
  const created = new Date().toISOString()
  return { ready, profile, created_at: created, expires_at: new Date(Date.now() + 86_400_000).toISOString() }
}

function previewGameID(path: string): string {
  let hash = 2166136261
  for (const value of path.toLowerCase()) hash = Math.imul(hash ^ value.charCodeAt(0), 16777619)
  return Math.abs(hash >>> 0).toString(16).padStart(12, '0').slice(-12)
}

function previewHistoryID(value: number): string {
  return value.toString(16).padStart(16, '0')
}

function previewTweak(id: string, category: string, name: string, current: string, desired: string, changed: boolean, risk: 'low' | 'medium' | 'high' = changed ? 'medium' : 'low') {
  return { id, category, name, current, desired, changed, effect: `Changes only the bounded ${name} target.`, benefit: 'Potential benefit is workload-dependent and should be measured.', risk: 'The original value is captured and can be restored.', risk_level: risk, reversible: true, manual_available: true, restore_available: false }
}

function previewPlan(plan: Plan, backups: Set<string>): Plan {
  return { ...plan, items: plan.items.map((item) => ({ ...item, changed: backups.has(item.id) ? false : item.changed, restore_available: backups.has(item.id) })) }
}

function pause(milliseconds: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeout = window.setTimeout(resolve, milliseconds)
    signal?.addEventListener('abort', () => { window.clearTimeout(timeout); reject(new DOMException('Aborted', 'AbortError')) }, { once: true })
  })
}
