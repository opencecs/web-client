<template>
  <div class="mobile-file">
    <van-nav-bar title="文件管理" left-arrow @click-left="$router.back()" :border="false" />

    <!-- 上传按钮 -->
    <div style="padding: 12px 16px; display: flex; gap: 8px">
      <van-button type="primary" size="small" icon="upgrade" :loading="uploading" @click="triggerUpload">上传文件</van-button>
      <van-button size="small" icon="delete" :disabled="!selected.length" @click="batchDelete">
        删除{{ selected.length ? `(${selected.length})` : '' }}
      </van-button>
      <van-button size="small" icon="replay" @click="loadFiles">刷新</van-button>
      <input ref="fileInput" type="file" multiple style="display: none" @change="onFileSelect" />
    </div>

    <!-- 上传进度 -->
    <div v-if="uploadQueue.length" style="padding: 0 16px 8px">
      <div v-for="item in uploadQueue" :key="item.name" style="margin-bottom: 6px">
        <div style="display: flex; justify-content: space-between; color: #999; font-size: 12px; margin-bottom: 2px">
          <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1">{{ item.name }}</span>
          <span>{{ item.progress }}%</span>
        </div>
        <van-progress :percentage="item.progress" :show-pivot="false" :color="item.status === 'exception' ? '#f56c6c' : '#409eff'" />
      </div>
    </div>

    <!-- 文件列表 -->
    <van-list :loading="loading" finished>
      <van-checkbox-group v-model="selected">
        <van-cell-group inset>
          <van-cell v-for="f in files" :key="f.name" clickable>
            <template #title>
              <div style="display: flex; align-items: center; gap: 8px">
                <van-checkbox :name="f.name" @click.stop />
                <div>
                  <div style="color: #e0e0e0; font-size: 14px">{{ f.name }}</div>
                  <div style="color: #999; font-size: 12px">{{ formatSize(f.size) }} · {{ f.modTime }}</div>
                </div>
              </div>
            </template>
            <template #value>
              <div style="display: flex; gap: 8px">
                <van-icon name="down" size="18" color="#409eff" @click="downloadFile(f.name)" />
                <van-icon name="link" size="18" color="#67c23a" @click="copyLink(f.name)" />
                <van-icon name="delete-o" size="18" color="#f56c6c" @click="confirmDelete(f.name)" />
              </div>
            </template>
          </van-cell>
        </van-cell-group>
      </van-checkbox-group>
    </van-list>

    <van-empty v-if="files.length === 0 && !loading" description="暂无文件" />
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { showToast, showConfirmDialog } from 'vant'
import api from '../../api/index.js'
import { copyText } from '../../utils/clipboard.js'

const loading = ref(false)
const uploading = ref(false)
const files = ref([])
const selected = ref([])
const fileInput = ref(null)
const uploadQueue = reactive([])

function formatSize(b) {
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  if (b < 1024 * 1024 * 1024) return (b / (1024 * 1024)).toFixed(1) + ' MB'
  return (b / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

async function loadFiles() {
  loading.value = true
  try {
    const { data } = await api.get('/file/list')
    files.value = data.data || []
  } catch {
    showToast('获取文件列表失败')
  } finally {
    loading.value = false
  }
}

function triggerUpload() {
  fileInput.value?.click()
}

async function onFileSelect(e) {
  const fileList = Array.from(e.target.files || [])
  if (!fileList.length) return

  uploading.value = true
  uploadQueue.length = 0

  for (const file of fileList) {
    const item = reactive({ name: file.name, progress: 0, status: '' })
    uploadQueue.push(item)

    const form = new FormData()
    form.append('file', file)

    try {
      await api.post('/file/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 600000,
        onUploadProgress(e) {
          item.progress = Math.round((e.loaded / e.total) * 100)
        }
      })
      item.status = 'success'
    } catch {
      item.status = 'exception'
      showToast(`${file.name} 上传失败`)
    }
  }

  uploading.value = false
  if (fileInput.value) fileInput.value.value = ''
  loadFiles()
  setTimeout(() => { uploadQueue.length = 0 }, 3000)
}

async function confirmDelete(name) {
  try {
    await showConfirmDialog({ title: '确认', message: `删除 ${name}？` })
    await api.delete('/file/delete', { params: { name } })
    showToast('已删除')
    loadFiles()
  } catch {}
}

async function batchDelete() {
  if (!selected.value.length) return
  try {
    await showConfirmDialog({ title: '批量删除', message: `删除选中的 ${selected.value.length} 个文件？` })
  } catch { return }

  let ok = 0
  for (const name of selected.value) {
    try {
      await api.delete('/file/delete', { params: { name } })
      ok++
    } catch {}
  }
  showToast(`删除 ${ok} 个文件`)
  selected.value = []
  loadFiles()
}

async function getDownloadUrl(name) {
  // 用登录态换取下载专用长期 token（30 天有效），避免链接因 JWT 过期而失效
  const { data } = await api.get('/file/download-token')
  const token = data?.token
  if (!token) throw new Error('获取下载 token 失败')
  // 拼接绝对地址，便于分享给他人访问
  return `${location.origin}/api/file/download?name=${encodeURIComponent(name)}&token=${token}`
}

async function downloadFile(name) {
  try {
    const url = await getDownloadUrl(name)
    window.open(url, '_blank')
  } catch {
    showToast('获取下载 token 失败')
  }
}

async function copyLink(name) {
  try {
    const url = await getDownloadUrl(name)
    const ok = await copyText(url)
    showToast(ok ? '下载链接已复制' : '复制链接失败')
  } catch {
    showToast('复制链接失败')
  }
}

onMounted(() => { loadFiles() })
</script>

<style scoped>
.mobile-file {
  background: #0a0a0a;
  min-height: 100vh;
}
</style>
