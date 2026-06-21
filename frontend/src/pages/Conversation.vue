<template>
  <div class="conv-layout">
    <div class="card conv-list">
      <button @click="showNew = true" style="width: 100%">+ 新对话</button>
      <div v-if="showNew" class="new-conv">
        <select v-model="newScene">
          <option value="travel">旅行</option>
          <option value="restaurant">餐厅点餐</option>
          <option value="campus">校园交流</option>
          <option value="interview">求职面试</option>
          <option value="cet">CET 口语</option>
        </select>
        <select v-model="newDifficulty">
          <option value="cet4">四级难度</option>
          <option value="cet6">六级难度</option>
        </select>
        <button class="small" @click="create">开始</button>
      </div>
      <div
        v-for="c in conversations"
        :key="c.id"
        class="conv-item"
        :class="{ active: c.id === currentId }"
        @click="select(c.id)"
      >
        <div>{{ c.title }}</div>
        <span class="muted" style="font-size: 12px">{{ c.scene }} · {{ c.status === 'finished' ? '已结束' : '进行中' }}</span>
      </div>
    </div>

    <div class="card conv-main">
      <div v-if="!currentId" class="muted">选择或新建一个对话开始练习。</div>
      <template v-else>
        <div class="messages">
          <div v-for="m in messages" :key="m.id" class="msg" :class="m.role">
            <div class="bubble">{{ m.content }}</div>
            <div v-if="m.feedback" class="feedback muted">💡 {{ m.feedback }}</div>
          </div>
          <div v-if="sending" class="muted">AI 正在回复...</div>
        </div>
        <div class="row input-row">
          <input v-model="draft" placeholder="用英文回复..." :disabled="sending" @keyup.enter="send" />
          <button :disabled="sending" @click="send">发送</button>
          <button class="ghost" @click="finish">结束</button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Conv { id: number; title: string; scene: string; status: string }
interface Msg { id: number; role: string; content: string; feedback: string | null }

const conversations = ref<Conv[]>([])
const messages = ref<Msg[]>([])
const currentId = ref<number | null>(null)
const draft = ref('')
const sending = ref(false)
const showNew = ref(false)
const newScene = ref('restaurant')
const newDifficulty = ref('cet4')

async function loadConvs() {
  const resp = await api.get('/conversations')
  conversations.value = resp.data || []
}
async function create() {
  const resp = await api.post('/conversations', { scene: newScene.value, difficulty: newDifficulty.value, title: newScene.value })
  showNew.value = false
  await loadConvs()
  await select(resp.data.id)
}
async function select(id: number) {
  currentId.value = id
  const resp = await api.get(`/conversations/${id}/messages`)
  messages.value = resp.data || []
}
async function send() {
  if (!draft.value.trim() || !currentId.value) return
  sending.value = true
  const content = draft.value
  draft.value = ''
  try {
    const resp = await api.post(`/conversations/${currentId.value}/messages`, { content })
    messages.value.push(resp.data.user_message, resp.data.ai_message)
  } finally {
    sending.value = false
  }
}
async function finish() {
  if (!currentId.value) return
  await api.post(`/conversations/${currentId.value}/finish`)
  await loadConvs()
}

onMounted(loadConvs)
</script>

<style scoped>
.conv-layout { display: flex; gap: 16px; height: calc(100vh - 100px); }
.conv-list { width: 240px; overflow-y: auto; }
.conv-main { flex: 1; display: flex; flex-direction: column; }
.new-conv { display: flex; flex-direction: column; gap: 6px; margin: 8px 0; }
.conv-item { padding: 8px; border-radius: 8px; cursor: pointer; margin-top: 4px; }
.conv-item:hover, .conv-item.active { background: #eef2fb; }
.messages { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 10px; padding-bottom: 12px; }
.msg.user { align-items: flex-end; display: flex; flex-direction: column; }
.msg.assistant { align-items: flex-start; display: flex; flex-direction: column; }
.bubble { max-width: 75%; padding: 10px 14px; border-radius: 12px; }
.msg.user .bubble { background: var(--primary); color: #fff; }
.msg.assistant .bubble { background: #eef2fb; }
.feedback { font-size: 13px; margin-top: 2px; }
.input-row { border-top: 1px solid var(--border); padding-top: 12px; }
</style>
