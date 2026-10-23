// Run against a built binary: node scripts/smoke.mjs release/navhub-windows-amd64.exe
import { spawn, spawnSync } from 'node:child_process'
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve, join } from 'node:path'
import assert from 'node:assert/strict'
const binary = resolve(process.argv[2] || 'release/navhub-windows-amd64.exe')
const temp = mkdtempSync(join(tmpdir(), 'navhub-smoke-'))
const config = join(temp, 'config.yaml')
writeFileSync(config, 'host: 127.0.0.1\nport: 3138\ndata_dir: data\nopen_browser: false\nlog_file: true\n')
assert.match(spawnSync(binary, ['--version'], { encoding: 'utf8' }).stdout, /1\.0\.0/)
assert.match(spawnSync(binary, ['--help'], { encoding: 'utf8' }).stderr, /config/)
let child
async function start() {
 let output = ''; child = spawn(binary, ['--config', config, '--no-browser'], { windowsHide: true }); child.stdout.on('data', b => output += b); child.stderr.on('data', b => output += b)
 for (let n = 0; n < 100; n++) { if (child.exitCode !== null) throw new Error(output); try { if ((await fetch('http://127.0.0.1:3138/api/status')).ok) return } catch {} await new Promise(r => setTimeout(r, 100)) }
 throw new Error('Startup timed out: ' + output)
}
async function stop() { const exited = new Promise(r => child.once('exit', r)); child.kill('SIGTERM'); await exited }
try {
 await start()
 const conflict = spawnSync(binary, ['--config', config, '--no-browser'], { encoding: 'utf8', timeout: 10000 })
 assert.equal(conflict.status, 1); assert.match(conflict.stdout + conflict.stderr, /端口不可用/)
 const setup = await fetch('http://127.0.0.1:3138/api/setup', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: 'smoke', password: 'smoke-test-password-123' }) })
 assert.equal(setup.status, 200)
 const cookie = setup.headers.get('set-cookie').split(';')[0]
 let data = await (await fetch('http://127.0.0.1:3138/api/data', { headers: { Cookie: cookie } })).json()
 data.categories.push({ id: 'smoke', name: '重启验收', visible: true })
 assert.equal((await fetch('http://127.0.0.1:3138/api/data', { method: 'PUT', headers: { Cookie: cookie, 'Content-Type': 'application/json' }, body: JSON.stringify(data) })).status, 200)
 await stop(); await start()
 data = await (await fetch('http://127.0.0.1:3138/api/data', { headers: { Cookie: cookie } })).json()
 assert.equal(data.categories[0].name, '重启验收')
 assert.equal((await fetch('http://127.0.0.1:3138/navigation/test')).status, 200)
 console.log('PASS: version, help, port conflict, startup, shutdown, restart, persisted session/data, SPA fallback')
 await stop()
} finally { if (child?.exitCode === null) child.kill('SIGTERM'); if (!temp.startsWith(join(tmpdir(), 'navhub-smoke-'))) throw new Error('Unexpected temporary path'); rmSync(temp, { recursive: true, force: true }) }
