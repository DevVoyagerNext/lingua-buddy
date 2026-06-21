<template>
  <div>
    <h2 class="title">语音学习</h2>
    <div class="card">
      <div class="row">
        <button v-if="!recording" @click="startRecord">🎙️ 开始录音</button>
        <button v-else class="danger" @click="stopRecord">⏹️ 停止录音</button>
        <span class="muted">或上传音频：</span>
        <input type="file" accept="audio/*" @change="onFile" style="width: auto" />
        <select v-model="language" style="width: 120px">
          <option value="auto">自动检测</option>
          <option value="en">英文</option>
          <option value="zh">中文</option>
        </select>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="loading" class="muted">识别中...</p>
    </div>

    <div v-if="audioFileId" class="card">
      <h3>识别文本（可编辑）</h3>
      <textarea v-model="recognizedText" rows="3"></textarea>
      <div class="row" style="margin-top: 10px">
        <button :disabled="translating" @click="doTranslate">翻译</button>
        <button class="ghost" @click="save">保存到历史</button>
        <span v-if="saveMsg" :class="saveMsgClass">{{ saveMsg }}</span>
      </div>
      <p v-if="translation" style="margin-top: 10px"><b>译文：</b>{{ translation }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { api, ApiError } from '@/api/client'

const recording = ref(false)
const loading = ref(false)
const translating = ref(false)
const error = ref('')
const language = ref('auto')
const audioFileId = ref<number | null>(null)
const recognizedText = ref('')
const translation = ref('')
const saveMsg = ref('')
const saveMsgClass = ref('ok')

let mediaRecorder: MediaRecorder | null = null
let chunks: Blob[] = []

async function startRecord() {
  error.value = ''
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    mediaRecorder = new MediaRecorder(stream)
    chunks = []
    mediaRecorder.ondataavailable = (e) => chunks.push(e.data)
    mediaRecorder.onstop = async () => {
      const blob = new Blob(chunks, { type: 'audio/webm' })
      stream.getTracks().forEach((t) => t.stop())
      await upload(new File([blob], 'recording.webm', { type: 'audio/webm' }))
    }
    mediaRecorder.start()
    recording.value = true
  } catch {
    error.value = '无法访问麦克风，请检查浏览器权限，或改用文件上传。'
  }
}
function stopRecord() {
  mediaRecorder?.stop()
  recording.value = false
}
function onFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) upload(f)
}

async function upload(file: File) {
  loading.value = true
  error.value = ''
  translation.value = ''
  try {
    const form = new FormData()
    form.append('audio', file)
    form.append('language', language.value)
    const resp = await api.post('/speech/transcribe', form, { headers: { 'Content-Type': 'multipart/form-data' } })
    audioFileId.value = resp.data.audio_file_id
    recognizedText.value = resp.data.text
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '识别失败'
  } finally {
    loading.value = false
  }
}

async function doTranslate() {
  translating.value = true
  try {
    const resp = await api.post('/translations', { text: recognizedText.value })
    translation.value = resp.data.output.translated_text
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '翻译失败'
  } finally {
    translating.value = false
  }
}

async function save() {
  if (!audioFileId.value) return
  saveMsg.value = ''
  try {
    await api.post('/speech/results', {
      audio_file_id: audioFileId.value,
      text: recognizedText.value,
      translation: translation.value,
    })
    saveMsg.value = '已保存到历史'
    saveMsgClass.value = 'ok'
  } catch (e) {
    saveMsg.value = e instanceof ApiError ? e.message : '保存失败'
    saveMsgClass.value = 'error'
  }
}
</script>
