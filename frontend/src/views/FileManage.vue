<template>
  <div style="padding: 24px">
    <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px">
      <div style="display: flex; align-items: center; gap: 12px">
        <h3 style="margin: 0; color: #e0e0e0">文件管理</h3>
        <span style="color: #999; font-size: 13px">共 {{ files.length }} 个文件，{{ totalSize }}</span>
      </div>
      <div style="display: flex; gap: 8px">
        <el-button size="small" type="danger" :disabled="!selected.length" @click="batchDelete">
          批量删除{{ selected.length ? ` (${selected.length})` : '' }}
        </el-button>
        <el-button size="small" :loading="uploading" type="primary" @click="triggerUpload">上传文件</el-button>
        <input ref="fileInput" type="file" multiple style="display: none" @change="onFileSelect" />
        <el-button size="small" :loading="loading" @click="loadFiles">刷新</el-button>
      </div>
    </div>

    <el-table :data="files" style="width: 100%" row-key="name" @selection-change="onSelChange" stripe>
      <el-table-column type="selection" width="45" />
      <el-table-column label="文件名" prop="name" min-width="240" show-overflow-tooltip />
      <el-table-column label="大小" width="120" align="right">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="修改时间" width="180">
        <template #default="{ row }">{{ row.modTime }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160" align="center">
        <template #default="{ row }">
          <el-button size="small" type="primary" text @click="downloadFile(row.name)">下载</el-button>
          <el-popconfirm :title="`确认删除 ${row.name}？`" @confirm="deleteFile(row.name)">
            <template #reference>
              <el-button size="small" type="danger" text>删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="files.length === 0 && !loading" description="暂无文件，点击上传按钮添加" />

    <!-- 上传进度 -->
    <div v-if="uploadQueue.length" style="margin-top: 16px">
      <div v-for="item in uploadQueue" :key="item.name" style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px">
        <span style="color: #e0e0e0; font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ item.name }}</span>
        <el-progress :percentage="item.progress" :status="item.status" style="width: 200px" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api/index.js'
import { useAuthStore } from '../stores/auth.js'

const auth = useAuthStore()
const loading = ref(false)
const uploading = ref(false)
const files = ref([])
const selected = ref([])
const fileInput = ref(null)
const uploadQueue = reactive([])

const totalSize = computed(() => formatSize(files.value.reduce((s, f) => s + f.size, 0)))

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
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '获取文件列表失败')
  } finally {
    loading.value = false
  }
}

function onSelChange(rows) {
  selected.value = rows
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
    } catch (err) {
      item.status = 'exception'
      ElMessage.error(`${file.name} 上传失败`)
    }
  }

  uploading.value = false
  if (fileInput.value) fileInput.value.value = ''
  loadFiles()
  // 3秒后清空进度
  setTimeout(() => { uploadQueue.length = 0 }, 3000)
}

async function deleteFile(name) {
  try {
    await api.delete('/file/delete', { params: { name } })
    ElMessage.success('已删除')
    loadFiles()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '删除失败')
  }
}

async function batchDelete() {
  const names = selected.value.map(r => r.name)
  if (!names.length) return
  try {
    await ElMessageBox.confirm(`确认删除选中的 ${names.length} 个文件？`, '批量删除', { type: 'warning' })
  } catch { return }

  let ok = 0, fail = 0
  for (const name of names) {
    try {
      await api.delete('/file/delete', { params: { name } })
      ok++
    } catch { fail++ }
  }
  ElMessage.success(`删除完成: ${ok} 成功${fail ? `, ${fail} 失败` : ''}`)
  loadFiles()
}

function downloadFile(name) {
  const token = auth.token
  const url = `/api/file/download?name=${encodeURIComponent(name)}&token=${token}`
  window.open(url, '_blank')
}

onMounted(() => { loadFiles() })
</script>
