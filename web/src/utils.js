export function initials(name) {
  const s = name.trim()
  if (!s) return 'NA'
  const chinese = s.match(/\p{Script=Han}/gu)
  if (chinese?.length) return chinese.slice(0, 2).join('')
  const words = s.split(/\s+/u)
  return (words.length > 1 ? words.slice(0, 2).map(w => Array.from(w)[0]).join('') : Array.from(s).slice(0, 2).join('')).toUpperCase()
}
export function newID() {
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join('')
}
export function ink(hex) {
  const n = parseInt(hex.replace('#', ''), 16)
  return ((n >> 16) * .299 + ((n >> 8) & 255) * .587 + (n & 255) * .114) > 155 ? '#22354b' : '#ffffff'
}
export function groups(data, query = '') {
  const q = query.trim().toLocaleLowerCase()
  return data.categories.filter(c => c.visible).map(c => ({ ...c, items: data.items.filter(i => i.categoryId === c.id && (!q || `${i.name} ${i.url} ${i.description}`.toLocaleLowerCase().includes(q))) })).filter(c => !q || c.items.length)
}
