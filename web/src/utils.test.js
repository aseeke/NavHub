import { test } from 'node:test'
import assert from 'node:assert/strict'
import { initials, groups, ink, newID } from './utils.js'
test('automatic icons match the specification', () => {
  for (const [name, icon] of Object.entries({ GitHub: 'GI', Docker: 'DO', 'Oracle Cloud': 'OC', ChatGPT: 'CH', 数据管理: '数据', 我的系统: '我的', NavHub: 'NA', ' A ': 'A', '': 'NA' })) assert.equal(initials(name), icon)
})
test('search uses name, URL and description and retains groups', () => {
 const d = { categories: [{ id: 'a', visible: true }, { id: 'b', visible: false }], items: [{ categoryId: 'a', name: 'GitHub', url: 'https://github.com', description: '代码' }] }
 for (const q of ['github', 'https', '代码']) assert.equal(groups(d, q)[0].items.length, 1)
 assert.equal(groups(d, 'missing').length, 0)
 assert.equal(ink('#ffffff'), '#22354b'); assert.equal(ink('#000000'), '#ffffff')
})
test('IDs work without the secure-context randomUUID API', () => {
 const first = newID(); assert.match(first, /^[a-f0-9]{32}$/); assert.notEqual(newID(), first)
})
