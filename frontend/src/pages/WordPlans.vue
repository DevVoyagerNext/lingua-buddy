<template>
  <div>
    <h2 class="title">我的单词书</h2>
    <p class="muted">点击已有的书进入背单词；点「＋ 添加单词书」可以再创建一本。同一时间只学一本，切换会自动暂停另一本。</p>

    <div class="bookshelf">
      <!-- 已创建的书 -->
      <div
        v-for="p in plans"
        :key="p.id"
        class="book"
        :class="{ active: p.status === 'active' }"
        :style="{ '--cover': meta(p.source_value).color }"
        @click="enterPlan(p)"
      >
        <div class="spine" />
        <div class="cover">
          <span class="badge">{{ p.status === 'active' ? '进行中' : '已暂停' }}</span>
          <div class="b-title">{{ p.name }}</div>
          <div class="b-sub">{{ meta(p.source_value).subtitle }}</div>
          <div class="b-progress">
            <div>每组 {{ p.group_size }} 词</div>
            <div v-if="p.counts">共 {{ p.counts.total }} 词</div>
          </div>
          <div class="b-enter">进入背单词 →</div>
        </div>
      </div>

      <!-- 添加单词书 -->
      <div class="book add-tile" @click="openPicker">
        <div class="plus">＋</div>
        <div class="b-title dark">添加单词书</div>
      </div>
    </div>

    <p v-if="!plans.length" class="muted" style="margin-top: 12px">还没有单词书，点「＋ 添加单词书」选一本开始吧。</p>
    <p v-if="msg" :class="msgClass" style="margin-top: 14px">{{ msg }}</p>

    <!-- 选择要创建的单词书 -->
    <div v-if="showPicker" class="modal-mask" @click.self="showPicker = false">
      <div class="modal">
        <div class="row">
          <h3 style="margin: 0">选择要创建的单词书</h3>
          <div class="spacer" />
          <button class="ghost small" @click="showPicker = false">关闭</button>
        </div>
        <p class="muted" style="margin-top: 8px">每组固定 10 个单词，按组学习，想学多少学多少。</p>
        <div class="catalog">
          <div
            v-for="c in availableCatalog"
            :key="c.value"
            class="cat-item"
            :style="{ '--cover': c.color }"
            @click="create(c)"
          >
            <div class="cat-title">{{ c.title }}</div>
            <div class="cat-words">{{ c.subtitle }} · {{ c.words }} 词</div>
          </div>
          <p v-if="!availableCatalog.length" class="muted">所有单词书都已创建。</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { api, ApiError } from '@/api/client'

interface Counts {
  total: number
  learning: number
  first_mastered: number
}
interface Plan {
  id: number
  source_value: string
  name: string
  status: string
  group_size: number
  completed_at: string | null
  counts?: Counts | null
}
interface CatalogItem {
  value: string
  title: string
  subtitle: string
  words: number
  color: string
}

// 全部可创建的单词书（对应 ecdict 的 8 个考试标签）。
const CATALOG: CatalogItem[] = [
  { value: 'zk', title: '中考词汇', subtitle: '中考', words: 1599, color: '#16a34a' },
  { value: 'gk', title: '高考词汇', subtitle: '高考', words: 3677, color: '#0891b2' },
  { value: 'cet4', title: '四级词汇', subtitle: 'CET-4', words: 3840, color: '#3b6cf6' },
  { value: 'cet6', title: '六级词汇', subtitle: 'CET-6', words: 5397, color: '#7c3aed' },
  { value: 'ky', title: '考研词汇', subtitle: '考研', words: 4801, color: '#db2777' },
  { value: 'toefl', title: '托福词汇', subtitle: 'TOEFL', words: 6951, color: '#ea580c' },
  { value: 'ielts', title: '雅思词汇', subtitle: 'IELTS', words: 5013, color: '#0d9488' },
  { value: 'gre', title: 'GRE 词汇', subtitle: 'GRE', words: 7479, color: '#b91c1c' },
]
const DEFAULT_META: CatalogItem = { value: '', title: '单词书', subtitle: '', words: 0, color: '#64748b' }

const router = useRouter()
const plans = ref<Plan[]>([])
const showPicker = ref(false)
const msg = ref('')
const msgClass = ref('ok')
const busy = ref(false)

function meta(value: string): CatalogItem {
  return CATALOG.find((c) => c.value === value) || DEFAULT_META
}

// 尚未创建的单词书。
const availableCatalog = computed(() =>
  CATALOG.filter((c) => !plans.value.some((p) => p.source_value === c.value)),
)

async function load() {
  const resp = await api.get<Plan[]>('/word-learning/plans')
  const list = resp.data || []
  for (const p of list) {
    try {
      const d = await api.get(`/word-learning/plans/${p.id}`)
      p.counts = d.data.counts
    } catch {
      p.counts = null
    }
  }
  plans.value = list
}

function activePlan(exceptId?: number) {
  return plans.value.find((p) => p.status === 'active' && p.id !== exceptId)
}

function openPicker() {
  msg.value = ''
  showPicker.value = true
}

// 创建一本新书（若已有进行中的书，先暂停它），创建后留在书架。
async function create(c: CatalogItem) {
  if (busy.value) return
  busy.value = true
  msg.value = ''
  try {
    const active = activePlan()
    if (active) await api.post(`/word-learning/plans/${active.id}/pause`)
    await api.post('/word-learning/plans', { source_value: c.value, name: c.title })
    await load()
    showPicker.value = false
    msg.value = `已创建《${c.title}》（每组 10 词），点击这本书开始背单词。`
    msgClass.value = 'ok'
  } catch (e) {
    msg.value = e instanceof ApiError ? e.message : '创建失败'
    msgClass.value = 'error'
  } finally {
    busy.value = false
  }
}

// 进入某本书：不是进行中就先激活（自动暂停另一本），再去背单词。
async function enterPlan(p: Plan) {
  if (busy.value) return
  busy.value = true
  msg.value = ''
  try {
    if (p.status !== 'active') {
      const active = activePlan(p.id)
      if (active) await api.post(`/word-learning/plans/${active.id}/pause`)
      await api.post(`/word-learning/plans/${p.id}/activate`)
    }
    router.push({ path: '/word-learning', query: { plan: String(p.id) } })
  } catch (e) {
    msg.value = e instanceof ApiError ? e.message : '操作失败'
    msgClass.value = 'error'
  } finally {
    busy.value = false
  }
}

onActivated(load)
</script>

<style scoped>
.bookshelf {
  display: flex;
  gap: 32px;
  flex-wrap: wrap;
  margin-top: 22px;
}
.book {
  position: relative;
  width: 190px;
  height: 250px;
  border-radius: 6px 12px 12px 6px;
  cursor: pointer;
  background: var(--cover);
  color: #fff;
  box-shadow: 5px 7px 18px rgba(0, 0, 0, 0.22);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.book:hover {
  transform: translateY(-8px);
  box-shadow: 7px 12px 26px rgba(0, 0, 0, 0.3);
}
.spine {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 16px;
  background: rgba(0, 0, 0, 0.18);
  border-radius: 6px 0 0 6px;
}
.cover {
  position: absolute;
  inset: 0;
  padding: 26px 18px 18px 32px;
  display: flex;
  flex-direction: column;
}
.badge {
  position: absolute;
  top: 12px;
  right: 12px;
  background: rgba(255, 255, 255, 0.25);
  padding: 2px 9px;
  border-radius: 10px;
  font-size: 12px;
}
.b-title {
  font-size: 24px;
  font-weight: 700;
  margin-top: 28px;
}
.b-sub {
  opacity: 0.85;
  margin-top: 4px;
}
.b-progress {
  margin-top: auto;
  font-size: 13px;
  line-height: 1.7;
  opacity: 0.95;
}
.b-enter {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
}
.book.active {
  outline: 3px solid #ffd54a;
  outline-offset: 2px;
}

/* 添加单词书磁贴 */
.add-tile {
  background: #fff;
  border: 2px dashed var(--border);
  box-shadow: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.add-tile:hover {
  border-color: var(--primary);
  box-shadow: none;
}
.plus {
  font-size: 72px;
  font-weight: 200;
  color: var(--primary);
  line-height: 1;
}
.b-title.dark {
  color: var(--text);
  font-size: 18px;
  margin-top: 12px;
}

/* 选择单词书弹窗 */
.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
}
.modal {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  width: 560px;
  max-width: 92vw;
  max-height: 84vh;
  overflow-y: auto;
}
.catalog {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  margin-top: 14px;
}
.cat-item {
  cursor: pointer;
  border-radius: 8px;
  padding: 16px;
  color: #fff;
  background: var(--cover);
  transition: transform 0.12s ease;
}
.cat-item:hover {
  transform: translateY(-3px);
}
.cat-title {
  font-size: 17px;
  font-weight: 700;
}
.cat-words {
  font-size: 13px;
  opacity: 0.9;
  margin-top: 4px;
}
</style>
