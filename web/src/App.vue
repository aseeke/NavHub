<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import Sortable from 'sortablejs'
import Card from './Card.vue'
import Icon from './Icon.vue'
import { groups, newID } from './utils'

const data = ref(null), status = ref({}), ready = ref(false), loading = ref(false), saving = ref(false), notice = ref(''), error = ref('')
const now = ref(new Date()), query = ref(''), searchOpen = ref(false), searchInput = ref(null), expanded = ref(false), managing = ref(false), panel = ref(''), confirmAction = ref(null)
const item = reactive({}), draft = reactive({}), credentials = reactive({ username: '', password: '' }), categoryName = ref(''), editCategory = ref(''), fileInput = ref(null), categoriesRoot = ref(null)
const palette = ['#ffffff', '#dceafa', '#dff1e5', '#e9e0f7', '#f8e0eb', '#f9e8d3', '#f6dcdc', '#e5e8ec']
const time = computed(() => now.value.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }))
const date = computed(() => now.value.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' }))
const visibleGroups = computed(() => data.value ? groups(data.value, query.value) : [])
const total = computed(() => visibleGroups.value.reduce((n, c) => n + c.items.length, 0))
const settings = computed(() => data.value?.settings || { title: status.value.title || 'NavHub', gradient: true, backgroundColor: '#f5f7fa', overlay: 0, showClock: true, showDate: true, radius: 12, gap: 14 })
const background = computed(() => ({ backgroundColor: settings.value.backgroundColor, backgroundImage: settings.value.background ? `url(${JSON.stringify(settings.value.background)})` : settings.value.gradient ? 'linear-gradient(180deg, #f8fafc 0%, #f1f4f8 100%)' : 'none' }))
const clone = v => JSON.parse(JSON.stringify(v))
watch(settings, s => { document.body.style.backgroundColor = s.backgroundColor }, { immediate: true })
let timer, toastTimer, sortables = [], categorySortable
async function api(path, options = {}) {
  const r = await fetch('/api/' + path, { ...options, headers: options.body instanceof FormData ? {} : { 'Content-Type': 'application/json' } })
  if (!r.ok) { const body = await r.json().catch(() => ({})); if (r.status === 401 && status.value.authenticated) { status.value.authenticated = false; managing.value = false; panel.value = 'login' }; throw new Error(body.error || '请求失败，请重试') }
  return r.json()
}
function toast(s) { notice.value = s; clearTimeout(toastTimer); toastTimer = setTimeout(() => notice.value = '', 3500) }
async function refresh() {
  status.value = await api('status'); document.title = status.value.title
  data.value = status.value.authenticated || status.value.public ? await api('data') : null
}
async function save(next, message = '已保存') {
  if (saving.value) return false
  saving.value = true; error.value = ''
  try { data.value = await api('data', { method: 'PUT', body: JSON.stringify(next) }); document.title = data.value.settings.title; toast(message); return true }
  catch (e) { error.value = e.message; return false }
  finally { saving.value = false; await nextTick(); initSort() }
}
function open(name) {
  error.value = ''; expanded.value = false
  if (!status.value.authenticated && name !== 'login') { panel.value = 'login'; return }
  panel.value = name
  if (name === 'settings') Object.assign(draft, clone(data.value.settings))
}
function newItem() {
  if (!status.value.authenticated) { open('login'); return }
  if (!data.value.categories.length) { open('categories'); toast('请先添加一个分类'); return }
  Object.assign(item, { id: '', name: '', url: '', categoryId: data.value.categories.find(c => c.visible)?.id || data.value.categories[0].id, description: '', icon: '', color: '', iconColor: '', target: '' }); open('item')
}
function editItem(i) { Object.assign(item, clone(i)); open('item') }
async function submitItem() {
  const next = clone(data.value), v = { ...item, name: item.name.trim(), url: item.url.trim(), id: item.id || newID() }
  const index = next.items.findIndex(i => i.id === v.id); if (index === -1) next.items.push(v); else next.items[index] = v
  if (await save(next, item.id ? '项目已更新' : '项目已添加')) panel.value = ''
}
function askDeleteItem() { confirmAction.value = { text: `删除“${item.name}”？此操作无法撤销。`, run: async () => { const next = clone(data.value); next.items = next.items.filter(i => i.id !== item.id); if (await save(next, '项目已删除')) panel.value = '' } } }
async function auth() {
  loading.value = true; error.value = ''
  try { await api(status.value.initialized ? 'login' : 'setup', { method: 'POST', body: JSON.stringify(credentials) }); credentials.password = ''; await refresh(); panel.value = ''; toast('欢迎回来') }
  catch (e) { error.value = e.message } finally { loading.value = false }
}
async function logout() {
  try { await api('logout', { method: 'POST' }); managing.value = false; panel.value = ''; await refresh(); toast('已退出登录') } catch (e) { error.value = e.message }
}
function toggleManage() { expanded.value = false; if (!status.value.authenticated) { open('login'); return }; managing.value = !managing.value; query.value = ''; searchOpen.value = false }
async function openSearch() { expanded.value = false; searchOpen.value = true; await nextTick(); searchInput.value?.focus() }
function keyboard(e) {
  if (e.key === 'Escape') { if (confirmAction.value) confirmAction.value = null; else if (panel.value) panel.value = ''; else { searchOpen.value = false; query.value = ''; expanded.value = false }; return }
  if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey && !e.target.closest('input,textarea,select,[contenteditable="true"]') && !panel.value && data.value) { e.preventDefault(); openSearch() }
}
async function upload(event, field, target = item) {
  const file = event.target.files[0]; if (!file) return
  loading.value = true; error.value = ''
  try { const form = new FormData(); form.append('file', file); target[field] = (await api('upload', { method: 'POST', body: form })).url; toast('图片已上传，保存后生效') } catch (e) { error.value = e.message } finally { loading.value = false; event.target.value = '' }
}
async function saveCategory() {
  if (!categoryName.value.trim()) return
  const next = clone(data.value)
  if (editCategory.value) next.categories.find(c => c.id === editCategory.value).name = categoryName.value.trim()
  else next.categories.push({ id: newID(), name: categoryName.value.trim(), visible: true })
  if (await save(next, '分类已保存')) { categoryName.value = ''; editCategory.value = '' }
}
async function toggleCategory(id) { const next = clone(data.value); const c = next.categories.find(c => c.id === id); c.visible = !c.visible; await save(next) }
function deleteCategory(c) {
  if (data.value.items.some(i => i.categoryId === c.id)) { error.value = '分类内还有项目，请先通过项目编辑窗口迁移到其他分类，或单独删除项目。'; return }
  confirmAction.value = { text: `删除空分类“${c.name}”？`, run: async () => { const next = clone(data.value); next.categories = next.categories.filter(v => v.id !== c.id); await save(next, '分类已删除') } }
}
function clearSort() { sortables.forEach(s => s.destroy()); sortables = [] }
function initSort() {
  clearSort()
  if (!managing.value || query.value || saving.value) return
  document.querySelectorAll('.cards').forEach(el => sortables.push(new Sortable(el, { group: 'items', animation: 140, draggable: '.nav-card', dataIdAttr: 'data-id', delay: 160, delayOnTouchOnly: true, touchStartThreshold: 5, ghostClass: 'drag-ghost', onEnd: async () => {
    const next = clone(data.value), ordered = []
    document.querySelectorAll('.cards').forEach(grid => grid.querySelectorAll('.nav-card').forEach(card => { const original = next.items.find(i => i.id === card.dataset.id); if (original) ordered.push({ ...original, categoryId: grid.dataset.category }) }))
    const ids = new Set(ordered.map(i => i.id)); next.items = [...ordered, ...next.items.filter(i => !ids.has(i.id))]
    // Restore DOM ownership to Vue before applying server state.
    clearSort(); data.value = null; await nextTick(); data.value = next
    if (!(await save(next, '排序已保存'))) await refresh().catch(e => error.value = e.message)
    await nextTick(); initSort()
  } })))
}
watch([managing, visibleGroups], async () => { await nextTick(); initSort() })
watch(panel, async name => {
  categorySortable?.destroy(); categorySortable = null
  await nextTick()
  if (name === 'categories' && categoriesRoot.value) categorySortable = new Sortable(categoriesRoot.value, { handle: '.drag-handle', animation: 140, dataIdAttr: 'data-id', onEnd: async () => {
    const next = clone(data.value); next.categories = categorySortable.toArray().map(id => next.categories.find(c => c.id === id)); panel.value = ''; await save(next, '分类排序已保存'); open('categories')
  } })
  const dialog = document.querySelector('.modal, .drawer'); dialog?.querySelector('input,button,select')?.focus()
})
watch(confirmAction, async action => { if (action) { await nextTick(); document.querySelector('.confirmation button')?.focus() } })
async function download(kind) {
  loading.value = true; error.value = ''
  try { const r = await fetch('/api/' + kind, { method: kind === 'backup' ? 'POST' : 'GET' }); if (!r.ok) throw new Error((await r.json()).error); const url = URL.createObjectURL(await r.blob()); const a = document.createElement('a'); a.href = url; a.download = kind === 'backup' ? 'navhub-backup.zip' : 'navhub.json'; a.click(); setTimeout(() => URL.revokeObjectURL(url), 10000); toast('文件已导出') } catch (e) { error.value = e.message } finally { loading.value = false }
}
async function importFile(e) {
 const file = e.target.files[0]; if (!file) return
 try { if (file.size > 4 * 1024 * 1024) throw new Error('JSON 文件不能超过 4MB'); const incoming = JSON.parse(await file.text()); confirmAction.value = { text: `将用文件中的 ${incoming.categories?.length || 0} 个分类和 ${incoming.items?.length || 0} 个项目替换当前数据及设置。建议先导出备份。`, run: async () => { loading.value = true; try { incoming.revision = data.value.revision; data.value = await api('import', { method: 'POST', body: JSON.stringify(incoming) }); toast('导入成功'); panel.value = ''; await refresh() } catch (e) { error.value = e.message } finally { loading.value = false } } } } catch (e) { error.value = '导入失败：' + e.message } finally { e.target.value = '' }
}
async function confirmRun() { const action = confirmAction.value; confirmAction.value = null; await action.run() }
function trapFocus(e) { if (e.key !== 'Tab') return; const els = Array.from(e.currentTarget.querySelectorAll('button,input,select,textarea,a[href]')).filter(el => !el.disabled && el.getClientRects().length); if (!els.length) return; const first = els[0], last = els.at(-1); if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus() } else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus() } }
onMounted(async () => { window.addEventListener('keydown', keyboard); timer = setInterval(() => now.value = new Date(), 60000); try { await refresh() } catch (e) { error.value = e.message } finally { ready.value = true } })
onUnmounted(() => { window.removeEventListener('keydown', keyboard); clearInterval(timer); clearTimeout(toastTimer); clearSort(); categorySortable?.destroy() })
</script>

<template>
  <div class="wallpaper" :style="background"></div>
  <div class="shade" :style="{ background: `rgba(0,0,0,${settings.overlay / 100})` }"></div>
  <main :inert="!!panel || !!confirmAction" :style="{ '--radius': settings.radius + 'px', '--gap': settings.gap + 'px' }">
    <header class="hero">
      <div class="identity"><span class="brand-mark" aria-hidden="true">N</span><span>{{ settings.title }}</span></div>
      <div v-if="settings.showClock" class="clock">{{ time }}</div>
      <div v-if="settings.showDate" class="date">{{ date }}</div>
      <button v-if="data && !searchOpen" class="search-trigger" @click="openSearch"><Icon name="search"/><span>搜索导航</span><kbd>/</kbd></button>
      <div v-if="data && searchOpen" class="search-field"><Icon name="search"/><input ref="searchInput" v-model="query" placeholder="搜索名称、网址或描述" aria-label="搜索导航"><button aria-label="关闭搜索" @click="searchOpen = false; query = ''">Esc</button></div>
    </header>

    <div v-if="!ready" class="empty">正在打开你的导航…</div>
    <section v-else-if="!data" class="welcome">
      <h1>{{ status.initialized ? '登录你的导航' : '创建导航空间' }}</h1><p>{{ status.initialized ? '此空间为私有，登录后即可访问。' : '设置管理员账户，开始整理常用网站和服务。' }}</p>
      <form @submit.prevent="auth"><label>管理员账户<input v-model="credentials.username" required maxlength="80" autocomplete="username" placeholder="输入账户名称"></label><label>密码<input v-model="credentials.password" type="password" required :minlength="status.initialized ? 1 : 10" maxlength="72" :autocomplete="status.initialized ? 'current-password' : 'new-password'" :placeholder="status.initialized ? '输入密码' : '至少 10 个字符'"></label><p v-if="error" class="error" role="alert">{{ error }}</p><button class="primary" :disabled="loading">{{ loading ? '请稍候…' : status.initialized ? '登录导航空间' : '创建我的导航空间' }}</button></form>
    </section>

    <template v-else>
      <div v-if="managing" class="manage-bar"><span><b><Icon name="edit"/>管理模式</b><small>点击卡片编辑，拖动调整顺序或分类</small></span><div><button @click="open('categories')">分类管理</button><button class="primary" @click="toggleManage">完成</button></div></div>
      <p v-if="searchOpen && query" class="search-summary">找到 {{ total }} 个项目 <button @click="query = ''">清除搜索</button></p>
      <section v-for="category in visibleGroups" :key="category.id" class="category">
        <div class="category-heading"><h2>{{ category.name }}</h2><span>{{ category.items.length }}</span><button v-if="managing" @click="newItem(); item.categoryId = category.id" :aria-label="'添加到' + category.name"><Icon name="plus"/></button></div>
        <div class="cards" :data-category="category.id"><Card v-for="i in category.items" :key="i.id" :data-id="i.id" :item="i" :settings="settings" :managing="managing" @edit="editItem"/><button v-if="managing && !category.items.length" class="empty-drop" @click="newItem(); item.categoryId = category.id">拖入项目，或点击添加</button></div>
        <p v-if="!managing && !category.items.length" class="category-empty">还没有项目</p>
      </section>
      <div v-if="!visibleGroups.length" class="empty"><span class="empty-symbol"><Icon :name="query ? 'search' : 'folder'"/></span><h2>{{ query ? '没有找到匹配的项目' : '还没有导航项目' }}</h2><p>{{ query ? '试试其他名称、网址或描述。' : '先创建一个分类，再添加常用网站和服务。' }}</p><button v-if="!query" class="primary" @click="open('categories')"><Icon name="plus"/>添加分类</button></div>
      <p v-if="managing && data.categories.some(c => !c.visible)" class="hidden-hint">隐藏分类及其中的项目，可在「分类管理」中重新显示；也可在下方选择项目编辑。</p>
      <div v-if="managing" class="hidden-items"><template v-for="c in data.categories.filter(c => !c.visible)" :key="c.id"><h3>{{ c.name }} · 已隐藏</h3><button v-for="i in data.items.filter(i => i.categoryId === c.id)" :key="i.id" @click="editItem(i)">{{ i.name }}<Icon name="edit"/></button></template></div>
      <footer><span class="footer-brand">NavHub</span><span>个人导航中心</span><button v-if="!status.authenticated" @click="open('login')">管理员登录</button></footer>
    </template>
  </main>

  <div v-if="data" class="floating" :inert="!!panel || !!confirmAction"><Transition name="toolbar"><div v-if="expanded" class="tools"><button @click="openSearch" aria-label="搜索"><Icon name="search"/>搜索 <kbd>/</kbd></button><button @click="newItem" aria-label="添加项目"><Icon name="plus"/>添加项目</button><button @click="toggleManage" :aria-label="managing ? '退出管理' : '管理模式'"><Icon name="edit"/>{{ managing ? '退出管理' : '管理模式' }}</button><button @click="open('settings')" aria-label="系统设置"><Icon name="settings"/>系统设置</button></div></Transition><button class="fab" :class="{ active: expanded }" :aria-expanded="expanded" aria-label="展开或收起操作栏" @click="expanded = !expanded"><Icon :name="expanded ? 'close' : 'menu'"/></button></div>
  <Transition name="toast"><div v-if="notice" class="toast" role="status"><Icon name="check"/>{{ notice }}</div></Transition>
  <div v-if="error && data && !panel" class="global-error" role="alert">{{ error }} <button @click="error = ''; refresh().catch(e => error = e.message)">重新加载</button><button aria-label="关闭错误" @click="error = ''">×</button></div>

  <div v-if="panel" class="scrim" @click.self="!saving && (panel = '')">
    <section :inert="!!confirmAction" :class="panel === 'settings' || panel === 'categories' ? 'drawer' : 'modal'" role="dialog" aria-modal="true" :aria-label="({ item: item.id ? '编辑项目' : '添加项目', settings: '系统设置', categories: '分类管理', login: '管理员登录' })[panel]" @keydown="trapFocus">
      <div class="panel-head"><div><h2>{{ ({ item: item.id ? '编辑项目' : '添加项目', settings: '系统设置', categories: '分类管理', login: '管理员登录' })[panel] }}</h2></div><div class="panel-head-actions"><button v-if="panel === 'settings'" type="submit" form="settings-form" class="primary" :disabled="saving || loading">保存设置</button><button class="close" aria-label="关闭窗口" @click="panel = ''"><Icon name="close"/></button></div></div>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <form v-if="panel === 'item'" @submit.prevent="submitItem">
        <div class="preview"><span>实时预览</span><Card :item="item" :settings="settings" preview/></div>
        <label>项目名称 <span>*</span><input v-model="item.name" required maxlength="100" placeholder="例如：我的工作台"></label>
        <label>访问网址 <span>*</span><input v-model="item.url" type="url" required maxlength="2048" placeholder="https://example.com"></label>
        <label>所属分类 <span>*</span><select v-model="item.categoryId" required><option v-for="c in data.categories" :key="c.id" :value="c.id">{{ c.name }}{{ c.visible ? '' : '（隐藏）' }}</option></select></label>
        <details class="appearance-options"><summary>外观与打开方式<Icon name="chevron"/></summary>
        <label>描述 <small>仅用于搜索，不显示在卡片上</small><input v-model="item.description" maxlength="500" placeholder="可选，输入一些搜索关键词"></label>
        <label>图标地址<input v-model="item.icon" placeholder="留空使用自动文字图标"></label><div class="inline-actions"><label class="upload-button">上传图标<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" @change="upload($event, 'icon')"></label><button type="button" @click="item.icon = ''">恢复自动图标</button><small>PNG / JPG / GIF / WebP，≤ 5MB</small></div>
        <label>卡片颜色 <small>留空使用系统默认</small></label><div class="swatches"><button v-for="color in palette" :key="color" type="button" :style="{ background: color }" :aria-label="color" :class="{ selected: item.color === color }" @click="item.color = color"></button><input v-model="item.color" placeholder="#ffffff" pattern="#[0-9a-fA-F]{6}" aria-label="自定义卡片 HEX 颜色"><button type="button" @click="item.color = ''" aria-label="恢复默认颜色"><Icon name="reset"/></button></div>
        <div class="form-row"><label>图标底色<input v-model="item.iconColor" placeholder="使用默认色" pattern="#[0-9a-fA-F]{6}"></label><label>打开方式<select v-model="item.target"><option value="">使用系统默认</option><option value="_blank">新标签页</option><option value="_self">当前页面</option></select></label></div>
        </details>
        <div class="panel-actions"><button v-if="item.id" type="button" class="danger" @click="askDeleteItem">删除项目</button><div></div><button type="button" @click="panel = ''">取消</button><button class="primary" :disabled="saving || loading">{{ saving ? '保存中…' : '保存项目' }}</button></div>
      </form>

      <template v-if="panel === 'categories'">
        <p class="help">为常用项目分组。拖动左侧手柄调整分类顺序。</p><form class="category-form" @submit.prevent="saveCategory"><input v-model="categoryName" required maxlength="60" placeholder="输入分类名称" aria-label="分类名称"><button class="primary" :disabled="saving">{{ editCategory ? '保存' : '添加' }}</button><button v-if="editCategory" type="button" @click="editCategory = ''; categoryName = ''">取消</button></form>
        <div ref="categoriesRoot" class="category-list"><div v-for="c in data.categories" :key="c.id" :data-id="c.id" class="category-row"><span class="drag-handle" title="拖动排序"><Icon name="grip"/></span><div><b>{{ c.name }}</b><small>{{ data.items.filter(i => i.categoryId === c.id).length }} 个项目 · {{ c.visible ? '显示中' : '已隐藏' }}</small></div><button :aria-label="'编辑' + c.name" @click="editCategory = c.id; categoryName = c.name">编辑</button><button :disabled="saving" @click="toggleCategory(c.id)">{{ c.visible ? '隐藏' : '显示' }}</button><button class="danger" :disabled="saving" @click="deleteCategory(c)">删除</button></div></div>
        <p v-if="!data.categories.length" class="help">还没有分类。在上方添加第一个分类。</p>
      </template>

      <form v-if="panel === 'settings'" id="settings-form" @submit.prevent="async () => { const next = clone(data); next.settings = clone(draft); if (await save(next, '设置已更新')) panel = '' }">
        <h3>页面外观</h3><label>页面标题<input v-model="draft.title" required maxlength="60"></label><label>背景图片地址<input v-model="draft.background" placeholder="留空使用默认背景"></label><div class="inline-actions"><label class="upload-button">上传背景<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" @change="upload($event, 'background', draft)"></label><button type="button" @click="draft.background = ''">恢复默认背景</button></div>
        <div class="form-row"><label>背景颜色<input v-model="draft.backgroundColor" type="color"></label><label class="check"><input v-model="draft.gradient" type="checkbox">使用柔和渐变</label></div>
        <label>背景遮罩 {{ draft.overlay }}%<input v-model.number="draft.overlay" type="range" min="0" max="80"></label><div class="form-row"><label class="check"><input v-model="draft.showClock" type="checkbox">显示时钟</label><label class="check"><input v-model="draft.showDate" type="checkbox">显示日期</label></div>
        <h3>卡片默认设置</h3><div class="form-row"><label>卡片背景<input v-model="draft.cardColor" type="color"></label><label>图标底色<input v-model="draft.iconColor" type="color"></label></div><div class="form-row"><label>圆角 {{ draft.radius }}px<input v-model.number="draft.radius" type="range" min="0" max="30"></label><label>间距 {{ draft.gap }}px<input v-model.number="draft.gap" type="range" min="6" max="30"></label></div><label>默认打开方式<select v-model="draft.target"><option value="_blank">新标签页</option><option value="_self">当前页面</option></select></label>
        <h3>访问权限</h3><label>导航访问模式<select v-model="draft.public"><option :value="false">私有 · 仅登录后可查看</option><option :value="true">公开 · 所有人可浏览，管理员可编辑</option></select></label><p class="help">隐藏分类不会向公开访客返回。公开模式也会公开背景图片。</p>
        <h3>数据与备份</h3><p class="help">JSON 包含分类、项目和页面设置，不含密码和会话。上传图片仅保存引用，迁移图片请下载完整备份。</p><div class="backup-actions"><button type="button" :disabled="loading" @click="download('export')">导出 JSON</button><button type="button" :disabled="loading" @click="fileInput.click()">导入 JSON</button><button type="button" :disabled="loading" @click="download('backup')">完整 ZIP 备份</button><input ref="fileInput" type="file" accept="application/json,.json" hidden @change="importFile"></div><p class="help">完整备份包含数据库、管理员密码哈希及上传图片，请妥善保存。恢复时停止程序，将 ZIP 解压至 data 目录。服务端备份保存在 data/backups，请定期清理。</p><div class="panel-actions"><button type="button" @click="logout">退出登录</button></div><p class="version">NavHub {{ status.version }} · 轻量级个人导航中心</p>
      </form>

      <form v-if="panel === 'login'" @submit.prevent="auth"><p class="help">登录管理员账户，管理你的导航空间。</p><label>账户<input v-model="credentials.username" autocomplete="username" required maxlength="80"></label><label>密码<input v-model="credentials.password" type="password" autocomplete="current-password" required maxlength="72"></label><div class="panel-actions"><button class="primary" :disabled="loading">{{ loading ? '登录中…' : '登录' }}</button></div></form>
    </section>
  </div>
  <div v-if="confirmAction" class="scrim confirmation" @click.self="confirmAction = null"><section class="modal small" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" @keydown="trapFocus"><h2 id="confirm-title">确认操作</h2><p>{{ confirmAction.text }}</p><div class="panel-actions"><button @click="confirmAction = null">取消</button><button class="primary" :disabled="saving || loading" @click="confirmRun">确认</button></div></section></div>
</template>
