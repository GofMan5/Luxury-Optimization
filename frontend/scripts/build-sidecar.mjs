import { execFileSync } from 'node:child_process'
import { mkdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontend = dirname(dirname(fileURLToPath(import.meta.url)))
const repository = dirname(frontend)
const packageJSON = JSON.parse(readFileSync(join(frontend, 'package.json'), 'utf8'))
const rustc = execFileSync('rustc', ['-vV'], { encoding: 'utf8' })
const target = process.env.TAURI_ENV_TARGET_TRIPLE || rustc.match(/^host:\s+(\S+)$/mu)?.[1]
if (!target) throw new Error('Unable to determine the Rust target triple')

const targets = {
  'x86_64-pc-windows-msvc': ['windows', 'amd64'],
  'aarch64-pc-windows-msvc': ['windows', 'arm64'],
  'i686-pc-windows-msvc': ['windows', '386'],
  'x86_64-unknown-linux-gnu': ['linux', 'amd64'],
  'aarch64-unknown-linux-gnu': ['linux', 'arm64'],
}
const platform = targets[target]
if (!platform) throw new Error(`Unsupported desktop target: ${target}`)
const [goos, goarch] = platform
const binaries = join(frontend, 'src-tauri', 'binaries')
const output = join(binaries, `luxury-optimization-backend-${target}${goos === 'windows' ? '.exe' : ''}`)
const ldflags = [`-s -w`, `-X github.com/GofMan5/Luxury-Optimization/internal/optimizer.version=${packageJSON.version}`]
if (goos === 'windows') ldflags.unshift('-H=windowsgui')

mkdirSync(binaries, { recursive: true })
execFileSync('go', ['build', '-mod=readonly', '-trimpath', '-ldflags', ldflags.join(' '), '-o', output, './cmd/luxury-optimization-backend'], {
  cwd: join(repository, 'backend'),
  env: { ...process.env, CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch },
  stdio: 'inherit',
})
console.log(`Built ${output}`)
