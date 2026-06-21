<template>
  <div class="conv-layout">
    <div class="card conv-list">
      <div class="conv-scroll">
        <div
          v-for="c in conversations"
          :key="c.id"
          class="conv-item"
          :class="{ active: c.id === currentId }"
          @click="select(c.id)"
        >
          <div class="conv-title">{{ c.title }}</div>
          <div class="conv-preview">{{ c.last_message || '还没有聊天内容' }}</div>
        </div>

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
          <div class="new-conv-actions">
            <button class="small" @click="create">开始</button>
            <button class="ghost small" @click="showNew = false">取消</button>
          </div>
        </div>
        <button v-else class="new-btn" @click="showNew = true">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          新对话
        </button>
      </div>
    </div>

    <div class="card conv-main">
      <div v-if="!currentId" class="empty-hint muted">选择或新建一个对话开始练习。</div>
      <template v-else>
        <div class="messages">
          <div v-for="m in messages" :key="m.id" class="msg" :class="m.role">
            <div class="bubble">{{ m.content }}</div>
            <div v-if="m.feedback" class="feedback muted">💡 {{ m.feedback }}</div>
          </div>
          <div v-if="sending" class="muted typing">AI 正在回复…</div>
        </div>
        <div class="input-row">
          <input v-model="draft" placeholder="用英文回复…" :disabled="sending" @keyup.enter="send" />
          <button class="send-btn" :disabled="sending || !draft.trim()" title="发送" @click="send">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round">
              <line x1="22" y1="2" x2="11" y2="13" />
              <polygon points="22 2 15 22 11 13 2 9 22 2" />
            </svg>
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import { api } from '@/api/client'

interface Conv { id: number; title: string; scene: string; status: string; last_message: string }
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
  const resp = await api.post('/conversations', { scene: newScene.value, difficulty: newDifficulty.value })
  showNew.value = false
  await loadConvs()
  await select(resp.data.id)
}
async function select(id: number) {
  currentId.value = id
  const resp = await api.get(`/conversations/${id}/messages`)
  messages.value = resp.data || []
  await scrollToBottom()
}
async function send() {
  if (!draft.value.trim() || !currentId.value) return
  sending.value = true
  const content = draft.value
  draft.value = ''
  try {
    const resp = await api.post(`/conversations/${currentId.value}/messages`, { content })
    messages.value.push(resp.data.user_message, resp.data.ai_message)
    await scrollToBottom()
    await loadConvs()
  } finally {
    sending.value = false
  }
}

async function scrollToBottom() {
  await nextTick()
  const el = document.querySelector('.messages')
  if (el) el.scrollTop = el.scrollHeight
}

onMounted(loadConvs)
</script>

<style scoped>
.conv-layout { display: flex; gap: 16px; height: calc(100vh - 100px); }

.conv-list { width: 260px; padding: 8px; display: flex; flex-direction: column; }
.conv-scroll { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; }

.conv-item {
  padding: 10px 12px;
  border-radius: 10px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.12s, border-color 0.12s;
}
.conv-item:hover { background: #f3f6fd; }
.conv-item.active { background: #eef2fb; border-color: var(--primary); }
.conv-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.conv-preview {
  margin-top: 3px;
  font-size: 12px;
  color: #8a93a6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.new-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  margin-top: 4px;
  background: transparent;
  border: 1px dashed var(--border);
  color: var(--primary);
}
.new-btn:hover { background: #eef2fb; border-color: var(--primary); }
.new-conv {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  margin-top: 4px;
  border: 1px solid var(--border);
  border-radius: 10px;
}
.new-conv-actions { display: flex; gap: 6px; }
.new-conv-actions button { flex: 1; }

.conv-main { flex: 1; display: flex; flex-direction: column; }
.empty-hint {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
.messages {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 4px 4px 12px;
}
.msg { display: flex; flex-direction: column; max-width: 78%; }
.msg.user { align-self: flex-end; align-items: flex-end; }
.msg.assistant { align-self: flex-start; align-items: flex-start; }
.bubble {
  padding: 10px 14px;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
}
.msg.user .bubble { background: var(--primary); color: #fff; border-bottom-right-radius: 4px; }
.msg.assistant .bubble {
  background: #f1f4fb;
  color: var(--text);
  border-bottom-left-radius: 4px;
}
.feedback { font-size: 13px; margin-top: 4px; line-height: 1.5; }
.typing { font-size: 13px; padding: 2px 4px; }

.input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  border-top: 1px solid var(--border);
  padding-top: 12px;
  margin-top: 4px;
}
.input-row input { flex: 1; }
.send-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  padding: 0;
  border-radius: 50%;
  flex: 0 0 auto;
}
.send-btn:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
