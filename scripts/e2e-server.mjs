import { mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawn, spawnSync } from 'node:child_process'
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const temp = mkdtempSync(join(tmpdir(), 'navhub-e2e-'))
const binary = join(temp, process.platform === 'win32' ? 'navhub.exe' : 'navhub')
const built = spawnSync('go', ['build', '-o', binary, '.'], { cwd: root, stdio: 'inherit' })
if (built.status) process.exit(built.status)
writeFileSync(join(temp, 'config.yaml'), 'host: 127.0.0.1\nport: 3137\ndata_dir: data\nopen_browser: false\nlog_file: false\n')
const child = spawn(binary, ['--config', join(temp, 'config.yaml'), '--no-browser'], { cwd: root, stdio: 'inherit', windowsHide: true })
let closed = false
function stop() { if (!closed) { closed = true; child.kill('SIGTERM') } }
process.on('SIGINT', stop); process.on('SIGTERM', stop)
child.on('exit', code => { if (!temp.startsWith(join(tmpdir(), 'navhub-e2e-'))) throw new Error('Unexpected temporary path'); rmSync(temp, { recursive: true, force: true }); process.exit(closed ? 0 : code || 0) })
