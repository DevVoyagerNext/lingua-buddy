<template>
  <div>
    <!-- 没有进行中的单词书 -->
    <div v-if="view === 'no_plan'" class="card">
      <p>还没有进行中的单词书。</p>
      <RouterLink to="/word-plans"><button>去单词书选一本</button></RouterLink>
    </div>

    <!-- 正在加载下一组 -->
    <div v-else-if="view === 'loading'" class="card">
      <p class="ok">正在准备单词，请稍候...</p>
    </div>

    <!-- 全部学完 / 暂无待复习 -->
    <div v-else-if="view === 'all_done'" class="card" style="text-align: center">
      <h2>🎉 太棒了！</h2>
      <p class="muted">{{ planName }} 当前没有需要学习或复习的组了，稍后再来 👍</p>
      <RouterLink to="/word-plans"><button class="ghost">换一本单词书</button></RouterLink>
    </div>

    <!-- 一组完成，自动进入下一组 -->
    <div v-else-if="view === 'between'" class="card" style="text-align: center">
      <h2>🎉 第 {{ finishedNo }} 组完成！</h2>
      <p class="muted">即将进入第 {{ nextNo }} 组...</p>
      <button @click="autoStart">继续学习 →</button>
    </div>

    <!-- 学习会话 -->
    <div v-else-if="view === 'studying' && session" class="card session">
      <div class="row">
        <div class="spacer" />
        <span class="muted">第 {{ session.index + 1 }} 组 · 单词 {{ wordIdx + 1 }}/{{ session.words.length }}</span>
      </div>
      <div class="progress-bar"><div class="progress-fill" :style="{ width: progressPct + '%' }" /></div>

      <template v-if="cur">
        <!-- 步骤0：详细释义（仅首学） -->
        <div v-if="step === 0" class="study-card">
          <div class="word-line">
            <h1>{{ cur.word }}</h1>
            <button class="speak" title="发音" @click="speak(cur.word)">🔊</button>
            <span v-if="cur.phonetic" class="muted">/{{ cur.phonetic }}/</span>
          </div>
          <p v-for="(t, i) in cur.translations" :key="'t' + i" class="def-zh">{{ t }}</p>
          <p v-for="(d, i) in cur.definitions" :key="'d' + i" class="def-en muted">{{ d }}</p>
          <button @click="step = 1">记住了，开始练习 →</button>
        </div>

        <!-- 步骤1：英文选中文 -->
        <div v-else-if="step === 1">
          <p class="q-prompt">看英文，选中文意思</p>
          <div class="word-line">
            <h1>{{ cur.word }}</h1>
            <button class="speak" title="发音" @click="speak(cur.word)">🔊</button>
          </div>
          <button
            v-for="opt in cur.meaning_options"
            :key="opt"
            class="option-btn"
            :class="optClass(opt)"
            @click="choose(opt, cur.gloss)"
          >
            {{ opt }}
          </button>
        </div>

        <!-- 步骤2：中文选英文 -->
        <div v-else-if="step === 2">
          <p class="q-prompt">看中文，选英文单词</p>
          <h2 class="cn-prompt">{{ cur.gloss }}</h2>
          <button
            v-for="opt in cur.word_options"
            :key="opt"
            class="option-btn"
            :class="optClass(opt)"
            @click="choose(opt, cur.word)"
          >
            {{ opt }}
          </button>
        </div>

        <!-- 步骤3：默写 -->
        <div v-else-if="step === 3">
          <p class="q-prompt">看中文，默写英文单词</p>
          <h2 class="cn-prompt">{{ cur.gloss }}</h2>
          <input v-model="spellInput" :disabled="spellShowAnswer" placeholder="输入英文单词" @keyup.enter="submitSpell" />
          <div class="row" style="margin-top: 10px">
            <button v-if="!spellShowAnswer" @click="submitSpell">提交</button>
            <button v-else @click="afterStep">继续 →</button>
            <button class="ghost" @click="speak(cur.word)">🔊 发音</button>
          </div>
          <p v-if="spellShowAnswer" class="error">正确答案：<b>{{ cur.word }}</b></p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onActivated, onDeactivated } from 'vue'
import { useRoute } from 'vue-router'
import { api, ApiError } from '@/api/client'

const route = useRoute()

interface GroupSummary {
  index: number
  status: string // new/learned/due
}
interface GroupWord {
  word: string
  phonetic: string
  definitions: string[]
  translations: string[]
  gloss: string
  meaning_options: string[]
  word_options: string[]
}
interface Session {
  index: number
  is_first_time: boolean
  words: GroupWord[]
}

type View = 'loading' | 'no_plan' | 'all_done' | 'between' | 'studying'

const view = ref<View>('loading')
const planName = ref('')
const loadedPlanId = ref('') // 已加载的书（计划）ID，用于判断是否切换了书
const groups = ref<GroupSummary[]>([])
const session = ref<Session | null>(null)
const wordIdx = ref(0)
const step = ref(1)
const wrongPicks = ref<string[]>([])
const spellInput = ref('')
const spellShowAnswer = ref(false)
const finishedNo = ref(0)
const nextNo = ref(0)
let advanceTimer: ReturnType<typeof setTimeout> | null = null

function clearAdvance() {
  if (advanceTimer) {
    clearTimeout(advanceTimer)
    advanceTimer = null
  }
}

const cur = computed(() => session.value?.words[wordIdx.value])
const progressPct = computed(() => {
  if (!session.value) return 0
  return Math.round((wordIdx.value / session.value.words.length) * 100)
})

function speak(word: string) {
  try {
    window.speechSynthesis.cancel()
    const u = new SpeechSynthesisUtterance(word)
    u.lang = 'en-US'
    window.speechSynthesis.speak(u)
  } catch {
    /* 浏览器不支持则忽略 */
  }
}

// 下一个要学的组：优先未学，其次到期复习。
function pickNextIndex(): number {
  const newG = groups.value.find((g) => g.status === 'new')
  if (newG) return newG.index
  const dueG = groups.value.find((g) => g.status === 'due')
  if (dueG) return dueG.index
  return -1
}

async function loadGroups(): Promise<boolean> {
  try {
    const resp = await api.get('/word-learning/groups')
    planName.value = resp.data.plan_name
    groups.value = resp.data.groups || []
    return true
  } catch (e) {
    view.value = e instanceof ApiError && e.code === 'NO_ACTIVE_PLAN' ? 'no_plan' : 'no_plan'
    return false
  }
}

// 自动进入下一个该学的组。
async function autoStart() {
  clearAdvance()
  view.value = 'loading'
  if (!(await loadGroups())) return
  const idx = pickNextIndex()
  if (idx < 0) {
    view.value = 'all_done'
    return
  }
  await startGroup(idx)
}

async function startGroup(index: number) {
  view.value = 'loading'
  try {
    const resp = await api.get<Session>(`/word-learning/groups/${index}`)
    session.value = resp.data
    wordIdx.value = 0
    step.value = resp.data.is_first_time ? 0 : 1
    resetQuestion()
    view.value = 'studying'
  } catch {
    // 该组无法学习（如全部词条缺失），跳到下一组避免卡死
    groups.value = groups.value.map((g) => (g.index === index ? { ...g, status: 'learned' } : g))
    const next = pickNextIndex()
    if (next < 0) view.value = 'all_done'
    else await startGroup(next)
  }
}

function resetQuestion() {
  wrongPicks.value = []
  spellInput.value = ''
  spellShowAnswer.value = false
}

function optClass(opt: string) {
  return wrongPicks.value.includes(opt) ? 'wrong' : ''
}

function choose(opt: string, correct: string) {
  if (opt === correct) afterStep()
  else if (!wrongPicks.value.includes(opt)) wrongPicks.value.push(opt)
}

function submitSpell() {
  if (!cur.value) return
  if (spellInput.value.trim().toLowerCase() === cur.value.word.toLowerCase()) afterStep()
  else spellShowAnswer.value = true
}

function afterStep() {
  resetQuestion()
  if (step.value < 3) {
    step.value++
    return
  }
  if (session.value && wordIdx.value < session.value.words.length - 1) {
    wordIdx.value++
    step.value = session.value.is_first_time ? 0 : 1
  } else {
    finishGroup()
  }
}

async function finishGroup() {
  if (!session.value) return
  const idx = session.value.index
  try {
    await api.post(`/word-learning/groups/${idx}/complete`)
  } catch {
    /* 完成标记失败不阻断 */
  }
  session.value = null
  await loadGroups()
  const next = pickNextIndex()
  if (next < 0) {
    view.value = 'all_done'
    return
  }
  finishedNo.value = idx + 1
  nextNo.value = next + 1
  view.value = 'between'
  // 默认学习下一个组：短暂展示完成提示后自动进入。
  clearAdvance()
  advanceTimer = setTimeout(() => {
    if (view.value === 'between') autoStart()
  }, 1500)
}

onActivated(() => {
  window.speechSynthesis?.cancel()
  const enteredPlan = (route.query.plan as string) || ''
  // 进入了不同的书：重新调用接口刷新要学习的单词。
  if (enteredPlan && enteredPlan !== loadedPlanId.value) {
    loadedPlanId.value = enteredPlan
    autoStart()
    return
  }
  // 同一本书：保留正在进行的学习/完成现场，不重新拉取。
  if ((view.value === 'studying' && session.value) || view.value === 'between') return
  autoStart()
})

onDeactivated(() => {
  // 离开页面时停掉自动进入下一组的定时器与发音，回来时停留在原处。
  clearAdvance()
  window.speechSynthesis?.cancel()
})

</script>

<style scoped>
.session {
  max-width: 560px;
}
.progress-bar {
  height: 6px;
  background: var(--border);
  border-radius: 3px;
  margin: 12px 0 20px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: var(--primary);
  transition: width 0.2s ease;
}
.study-card {
  text-align: center;
}
.word-line {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
  margin: 10px 0;
}
.word-line h1 {
  margin: 0;
  font-size: 34px;
}
.speak {
  background: transparent;
  border: 1px solid var(--border);
  font-size: 18px;
  padding: 4px 10px;
  border-radius: 8px;
}
.speak:hover {
  background: #eef2fb;
}
.def-zh {
  font-size: 17px;
  margin: 6px 0;
}
.def-en {
  font-size: 14px;
}
.q-prompt {
  color: var(--muted);
  text-align: center;
}
.cn-prompt {
  text-align: center;
  font-size: 26px;
  margin: 10px 0 18px;
}
</style>
