<template>
  <div>
    <h2 class="title">语法工具</h2>
    <div class="card">
      <div class="row" style="margin-bottom: 10px">
        <button :class="tab === 'analysis' ? '' : 'ghost'" @click="tab = 'analysis'">语法分析</button>
        <button :class="tab === 'correct' ? '' : 'ghost'" @click="tab = 'correct'">语法纠错</button>
        <button :class="tab === 'polish' ? '' : 'ghost'" @click="tab = 'polish'">句子润色</button>
      </div>
      <textarea v-model="text" rows="4" placeholder="输入英文句子或短文"></textarea>
      <div class="row" style="margin-top: 10px">
        <select v-if="tab === 'polish'" v-model="style" style="width: 140px">
          <option value="natural">自然</option>
          <option value="concise">简洁</option>
          <option value="formal">正式</option>
          <option value="advanced">高级</option>
        </select>
        <button :disabled="loading" @click="run">{{ loading ? '处理中...' : '提交' }}</button>
        <span v-if="error" class="error">{{ error }}</span>
      </div>
    </div>

    <div v-if="analysis" class="card">
      <h3>结构分析</h3>
      <p>句型：{{ analysis.sentence_type }}｜时态：{{ analysis.tense }}｜语态：{{ analysis.voice }}</p>
      <p class="muted">主干：主语「{{ analysis.main_clause.subject }}」谓语「{{ analysis.main_clause.predicate }}」宾语「{{ analysis.main_clause.object }}」</p>
      <div v-if="analysis.grammar_points?.length">
        <p v-for="(g, i) in analysis.grammar_points" :key="i"><b>{{ g.name }}</b>：{{ g.explanation_zh }}</p>
      </div>
      <p>{{ analysis.explanation_zh }}</p>
    </div>

    <div v-if="correction" class="card">
      <h3>{{ tab === 'polish' ? '润色结果' : '纠错结果' }}</h3>
      <p style="font-size: 16px">{{ correction.corrected_text }}</p>
      <button class="ghost small" @click="collect(correction.corrected_text)">收藏句子</button>
      <div v-if="correction.issues?.length" style="margin-top: 12px">
        <h4>修改点</h4>
        <div v-for="(it, i) in correction.issues" :key="i" class="list-item" style="display:block">
          <span class="tag">{{ it.type }}</span>
          <span class="error">{{ it.original }}</span> → <span class="ok">{{ it.replacement }}</span>
          <p class="muted" style="margin: 4px 0">{{ it.explanation_zh }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { api, ApiError } from '@/api/client'

const tab = ref<'analysis' | 'correct' | 'polish'>('analysis')
const text = ref('')
const style = ref('natural')
const loading = ref(false)
const error = ref('')
const analysis = ref<any>(null)
const correction = ref<any>(null)

async function run() {
  error.value = ''
  analysis.value = null
  correction.value = null
  if (!text.value.trim()) {
    error.value = '请输入文本'
    return
  }
  loading.value = true
  try {
    if (tab.value === 'analysis') {
      const resp = await api.post('/grammar/analysis', { text: text.value })
      analysis.value = resp.data.analysis
    } else {
      const resp = await api.post('/corrections', {
        text: text.value,
        mode: tab.value === 'polish' ? 'polish' : 'correct',
        style: style.value,
      })
      correction.value = resp.data.result
    }
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '处理失败'
  } finally {
    loading.value = false
  }
}

async function collect(sentence: string) {
  try {
    await api.post('/sentences', { sentence })
  } catch {
    /* 忽略重复收藏 */
  }
}
</script>
