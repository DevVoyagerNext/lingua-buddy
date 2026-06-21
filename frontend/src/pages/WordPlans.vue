<template>
  <div>
    <h2 class="title">词汇计划</h2>
    <div class="card">
      <div class="row">
        <select v-model="sourceValue" style="width: 200px">
          <option value="cet4">四级词汇（CET-4）</option>
          <option value="cet6">六级词汇（CET-6）</option>
        </select>
        <button :disabled="creating" @click="create">{{ creating ? '创建中...' : '创建计划' }}</button>
        <span v-if="msg" :class="msgClass">{{ msg }}</span>
      </div>
      <p class="muted" style="margin-top: 8px">
        创建后默认每天激活 10 个新词、活跃上限 20 个；同一时间只能有一个进行中的计划。
      </p>
    </div>

    <div v-for="p in plans" :key="p.id" class="card">
      <div class="row">
        <h3 style="margin: 0">{{ p.name }}</h3>
        <span class="tag">{{ statusLabel(p.status) }}</span>
        <span v-if="p.completed_at" class="tag ok">已完成首轮</span>
        <div class="spacer" />
        <button v-if="p.status !== 'active'" class="small" @click="activate(p.id)">激活</button>
        <button v-else class="ghost small" @click="pause(p.id)">暂停</button>
      </div>
      <PlanCounts :plan-id="p.id" :reload="reloadKey" />
    </div>
    <p v-if="!plans.length" class="muted">还没有计划，先创建一个吧。</p>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, defineComponent, h } from 'vue'
import { api, ApiError } from '@/api/client'

interface Plan {
  id: number
  name: string
  status: string
  completed_at: string | null
  source_snapshot_count: number
}

const plans = ref<Plan[]>([])
const sourceValue = ref('cet4')
const creating = ref(false)
const msg = ref('')
const msgClass = ref('ok')
const reloadKey = ref(0)

function statusLabel(s: string) {
  return { active: '进行中', paused: '已暂停', archived: '已归档' }[s] || s
}

async function load() {
  const resp = await api.get<Plan[]>('/word-learning/plans')
  plans.value = resp.data || []
}

async function create() {
  msg.value = ''
  creating.value = true
  try {
    await api.post('/word-learning/plans', { source_value: sourceValue.value })
    msg.value = '创建成功'
    msgClass.value = 'ok'
    await load()
  } catch (e) {
    msg.value = e instanceof ApiError ? e.message : '创建失败'
    msgClass.value = 'error'
  } finally {
    creating.value = false
  }
}

async function activate(id: number) {
  try {
    await api.post(`/word-learning/plans/${id}/activate`)
    await load()
  } catch (e) {
    msg.value = e instanceof ApiError ? e.message : '操作失败'
    msgClass.value = 'error'
  }
}
async function pause(id: number) {
  await api.post(`/word-learning/plans/${id}/pause`)
  await load()
}

const PlanCounts = defineComponent({
  props: { planId: { type: Number, required: true }, reload: Number },
  setup(props) {
    const counts = ref<any>(null)
    async function fetchCounts() {
      const resp = await api.get(`/word-learning/plans/${props.planId}`)
      counts.value = resp.data.counts
    }
    fetchCounts()
    return () =>
      counts.value
        ? h('div', { class: 'muted', style: 'margin-top:8px' }, [
            `总词数 ${counts.value.total}｜等待 ${counts.value.waiting}｜学习中 ${counts.value.learning}｜已掌握 ${counts.value.first_mastered}｜跳过 ${counts.value.skipped}`,
          ])
        : h('div', { class: 'muted' }, '加载进度...')
  },
})

onMounted(load)
</script>
