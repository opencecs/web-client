<template>
  <div style="padding: var(--space-lg)">
    <el-tabs v-model="activeTab" type="border-card">
      <el-tab-pane label="云机管理" name="slots">
        <!-- 操作栏 -->
        <SlotActions
          :selected="selectedSlots"
          @create="openCreate"
          @projection="openProjection"
          @close-all-projections="closeAllProjections"
          @terminal="openTerminal"
          @backup-switch="openBackupSwitch"
          @rename="openAlias"
          @copy="openCopy"
          @s5proxy="openS5Proxy"
          @switch-model="openSwitchModel"
          @batch-upload="openBatchUpload"
        />
        <!-- 方块网格 -->
        <SlotGrid ref="slotGridRef" :max-slots="maxSlots" @selection-change="onSelectionChange" />

        <!-- 截图预览 -->
        <div style="margin-top: 20px">
          <h4 style="color: var(--text-primary); margin-bottom: 10px; font-weight: 600">实时截图</h4>
          <SlotScreenshots :max-slots="maxSlots" @projection="openProjection" />
        </div>
      </el-tab-pane>
      <el-tab-pane v-if="auth.can('image_view')" label="镜像管理" name="images" lazy>
        <ImageManage />
      </el-tab-pane>
      <el-tab-pane v-if="auth.can('network_bridge')" label="虚拟网卡" name="network" lazy>
        <NetworkTab />
      </el-tab-pane>
      <el-tab-pane v-if="auth.can('vpc_manage')" label="VPC 管理" name="vpc" lazy>
        <VpcManageTab />
      </el-tab-pane>
    </el-tabs>

    <!-- 创建容器弹窗 -->
    <CreateContainer v-model="createVisible" :max-slots="maxSlots" :default-slot="createDefaultSlot"
      @created="device.refreshContainers()" />

    <!-- 备份切换弹窗 -->
    <BackupSwitch v-model="backupSwitchVisible" :slot-num="backupSwitchSlot" />

    <!-- 设置别名弹窗 -->
    <el-dialog v-model="aliasVisible" title="设置别名" width="400px">
      <el-form label-width="80px">
        <el-form-item v-if="aliasTarget" label="容器 ID">
          <el-input :model-value="aliasTarget.name" readonly />
        </el-form-item>
        <el-form-item v-if="isBatchAlias" label="批量设置">
          <span style="color: #e6a23c; font-size: 13px">将为 {{ aliasBatchTargets.length }} 个容器设置别名（自动加坑位号后缀）</span>
        </el-form-item>
        <el-form-item v-if="aliasTarget && currentAlias" label="当前别名">
          <span style="color: #b0b0b0">{{ currentAlias }}</span>
        </el-form-item>
        <el-form-item label="新别名">
          <el-input v-model="aliasInput" :placeholder="isBatchAlias ? '输入别名前缀（如：游戏）' : '输入别名（支持中文、空格、符号）'" clearable />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="aliasVisible = false">取消</el-button>
        <el-button v-if="currentAlias || isBatchAlias" type="danger" :loading="aliasSaving" @click="doRemoveAlias">清除别名</el-button>
        <el-button type="primary" :loading="aliasSaving" @click="doSetAlias">保存</el-button>
      </template>
    </el-dialog>

    <!-- 复制弹窗 -->
    <el-dialog v-model="copyVisible" title="复制容器" width="400px">
      <el-form label-width="90px">
        <el-form-item label="源容器">{{ device.displayName(copyTarget?.name) }}</el-form-item>
        <el-form-item label="目标坑位">
          <el-input-number v-model="copySlot" :min="1" :max="maxSlots" :step="1" />
        </el-form-item>
        <el-form-item label="复制数量">
          <el-input-number v-model="copyCount" :min="1" :max="20" :step="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="copyVisible = false">取消</el-button>
        <el-button type="primary" :loading="copying" @click="doCopy">确认</el-button>
      </template>
    </el-dialog>

    <!-- 多投屏窗口 -->
    <ContainerProjection
      v-for="(p, idx) in projections"
      :key="p.name"
      v-model="p.visible"
      :container="p.container"
      :offset-index="idx"
      @close="removeProjection"
    />

    <!-- 终端弹窗 -->
    <ContainerTerminal v-model="terminalVisible" :container="terminalContainer" />

    <!-- S5 代理弹窗 -->
    <el-dialog v-model="s5Visible" title="S5 代理管理" width="550px" @opened="fetchS5Status">
      <div v-if="s5Container" style="margin-bottom: 12px">
        <span style="color: #b0b0b0">容器：</span>
        <span style="color: #f0f0f0">{{ device.displayName(s5Container.name) }}</span>
      </div>

      <!-- 当前状态 -->
      <div v-if="s5Status.status === 1" style="padding: 14px 16px; border-radius: 6px; margin-bottom: 16px; background: #162312; border: 1px solid #67c23a">
        <div style="display: flex; align-items: center; justify-content: space-between">
          <div style="display: flex; align-items: center; gap: 10px">
            <el-tag type="success" size="small" effect="dark">已启动</el-tag>
            <span style="color: #f0f0f0; font-size: 13px; font-family: monospace">{{ s5Status.addr }}</span>
          </div>
          <el-button type="danger" size="small" :loading="s5Stopping" @click="doStopS5">停止代理</el-button>
        </div>
      </div>
      <div v-else style="padding: 14px 16px; border-radius: 6px; margin-bottom: 16px; background: #1e1e1e; border: 1px dashed #555; text-align: center">
        <el-tag type="info" size="small">未启动</el-tag>
        <span style="color: #909090; margin-left: 8px; font-size: 13px">当前未配置 S5 代理</span>
      </div>

      <!-- 快捷解析 -->
      <div style="display: flex; gap: 8px; margin-bottom: 14px; align-items: center">
        <div style="color: #b0b0b0; font-size: 12px; white-space: nowrap">S5信息</div>
        <el-input v-model="s5QuickInput" placeholder="格式: 地址:端口:用户名:密码（用户名密码可省略）" size="small" />
        <el-button size="small" @click="parseS5Quick">解析并填写</el-button>
      </div>

      <!-- 设置表单 -->
      <div style="color: #f0f0f0; font-weight: bold; margin-bottom: 10px">{{ s5Status.status === 1 ? '修改代理' : '设置代理' }}</div>
      <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 12px">
        <div>
          <div style="color: #b0b0b0; font-size: 12px; margin-bottom: 4px">服务器 IP</div>
          <el-input v-model="s5Form.addr" placeholder="如 1.2.3.4" />
        </div>
        <div>
          <div style="color: #b0b0b0; font-size: 12px; margin-bottom: 4px">端口</div>
          <el-input v-model="s5Form.port" placeholder="如 1080" />
        </div>
        <div>
          <div style="color: #b0b0b0; font-size: 12px; margin-bottom: 4px">用户名</div>
          <el-input v-model="s5Form.usr" placeholder="无认证可留空" />
        </div>
        <div>
          <div style="color: #b0b0b0; font-size: 12px; margin-bottom: 4px">密码</div>
          <el-input v-model="s5Form.pwd" placeholder="无认证可留空" type="password" show-password />
        </div>
      </div>
      <div style="margin-bottom: 12px">
        <div style="color: #b0b0b0; font-size: 12px; margin-bottom: 4px">域名解析模式</div>
        <el-radio-group v-model="s5Form.type">
          <el-radio value="1">本地解析</el-radio>
          <el-radio value="2">服务端解析</el-radio>
        </el-radio-group>
      </div>

      <template #footer>
        <el-button @click="s5Visible = false">关闭</el-button>
        <el-button type="primary" :loading="s5Setting" @click="doSetS5">
          {{ s5Status.status === 1 ? '更新代理' : '启动代理' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 切换机型弹窗 -->
    <el-dialog v-model="switchModelVisible" title="切换机型" width="480px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="容器">
          <span style="color: #b0b0b0">{{ switchModelTarget?.name }}</span>
        </el-form-item>
        <el-form-item label="安卓版本">
          <span style="color: #b0b0b0">{{ switchModelVersion === 'and14' ? 'Android 14' : 'Android 16' }}</span>
        </el-form-item>
        <el-form-item label="手机型号">
          <el-select v-model="switchModelId" filterable clearable placeholder="留空随机分配"
            style="width: 100%" :loading="switchModelLoading">
            <el-option v-for="m in switchModelFiltered" :key="m.id || m.modelId"
              :label="m.name || m.modelName" :value="m.id || m.modelId" />
          </el-select>
          <div style="color: #b0b0b0; font-size: 11px; margin-top: 2px">
            {{ switchModelFiltered.length }} 个机型可选，留空随机分配
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="switchModelVisible = false">取消</el-button>
        <el-button type="primary" :loading="switchModelSaving" @click="doSwitchModel">确认</el-button>
      </template>
    </el-dialog>

    <!-- 批量上传弹窗 -->
    <el-dialog v-model="batchUploadVisible" title="批量上传文件" width="600px" destroy-on-close>
      <div style="margin-bottom: 12px">
        <span style="color: #b0b0b0">将文件推送到以下容器的 /sdcard/upload/ 目录：</span>
        <div style="margin-top: 8px; display: flex; flex-wrap: wrap; gap: 4px">
          <el-tag v-for="c in batchUploadContainers" :key="c.name" size="small"
            :type="batchUploadResult[c.name]?.success ? 'success' : batchUploadResult[c.name]?.fail ? 'danger' : 'info'">
            {{ device.displayName(c.name) }}
            <span v-if="batchUploadResult[c.name]?.success"> ✓</span>
            <span v-if="batchUploadResult[c.name]?.fail"> ✗</span>
            <span v-if="batchUploadResult[c.name]?.uploading" style="color: #409eff"> ↑</span>
          </el-tag>
        </div>
      </div>

      <!-- 从文件管理选取文件 -->
      <div style="margin-bottom: 12px; display: flex; align-items: center; justify-content: space-between">
        <span style="color: #f0f0f0; font-weight: 600">选择文件</span>
        <span style="color: #e6a23c; font-size: 12px; margin-left: 8px; font-weight: bold">APK 文件会自动安装</span>
        <el-button size="small" :loading="batchFileLoading" @click="loadBatchFiles">刷新</el-button>
      </div>
      <el-table ref="batchFileTableRef" :data="batchFileList" style="width: 100%" max-height="300"
        @selection-change="onBatchFileSelect" v-loading="batchFileLoading" stripe size="small">
        <el-table-column type="selection" width="45" />
        <el-table-column label="文件名" prop="name" min-width="200" show-overflow-tooltip />
        <el-table-column label="大小" width="100" align="right">
          <template #default="{ row }">{{ formatBatchSize(row.size) }}</template>
        </el-table-column>
      </el-table>
      <div v-if="!batchFileLoading && batchFileList.length === 0" style="color: #909090; font-size: 13px; text-align: center; padding: 20px 0">
        暂无文件，请先到文件管理页面上传
      </div>

      <div v-if="batchUploadProgress.total > 0" style="margin-top: 12px">
        <el-progress :percentage="Math.round(batchUploadProgress.done / batchUploadProgress.total * 100)"
          :status="batchUploadProgress.fail > 0 ? 'exception' : undefined" />
        <div style="color: #b0b0b0; font-size: 12px; margin-top: 4px">
          {{ batchUploadProgress.done }} / {{ batchUploadProgress.total }}
          <span v-if="batchUploadProgress.fail > 0" style="color: #f56c6c">（{{ batchUploadProgress.fail }} 失败）</span>
        </div>
      </div>
      <template #footer>
        <el-button @click="batchUploadVisible = false">{{ batchUploadRunning ? '关闭' : '取消' }}</el-button>
        <el-button type="primary" :loading="batchUploadRunning" :disabled="!batchSelectedFiles.length"
          @click="doBatchUpload">开始上传（{{ batchSelectedFiles.length }} 个文件 × {{ batchUploadContainers.length }} 个容器）</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth.js'
import { useDeviceStore } from '../stores/device.js'
import SlotGrid from '../components/android/SlotGrid.vue'
import SlotActions from '../components/android/SlotActions.vue'
import SlotScreenshots from '../components/android/SlotScreenshots.vue'
import CreateContainer from '../components/android/CreateContainer.vue'
import BackupSwitch from '../components/android/BackupSwitch.vue'
import ImageManage from '../components/android/ImageManage.vue'
import NetworkTab from '../components/android/NetworkTab.vue'
import VpcManageTab from '../components/android/VpcManageTab.vue'
import ContainerProjection from '../components/android/ContainerProjection.vue'
import ContainerTerminal from '../components/android/ContainerTerminal.vue'
import { reactive } from 'vue'
import api from '../api/index.js'

const auth = useAuthStore()
const device = useDeviceStore()
const activeTab = ref('slots')
const slotGridRef = ref(null)
const selectedSlots = ref([])

const maxSlots = computed(() => {
  const model = (device.status?.model || '').toLowerCase()
  return model.includes('p1') ? 24 : 12
})

function onSelectionChange(slots) {
  selectedSlots.value = slots
}

// 创建容器
const createVisible = ref(false)
const createDefaultSlot = ref(1)
function openCreate() {
  const single = selectedSlots.value.length === 1 ? selectedSlots.value[0] : null
  createDefaultSlot.value = single?.num || 1
  createVisible.value = true
}

// 备份切换
const backupSwitchVisible = ref(false)
const backupSwitchSlot = ref(0)
function openBackupSwitch(slot) {
  backupSwitchSlot.value = slot.num
  backupSwitchVisible.value = true
}

// 设置别名
const aliasVisible = ref(false)
const aliasTarget = ref(null) // 单选时为容器对象，多选时为 null
const aliasBatchTargets = ref([]) // 多选时的容器列表
const aliasInput = ref('')
const aliasSaving = ref(false)
const isBatchAlias = computed(() => aliasBatchTargets.value.length > 1)
const currentAlias = computed(() => {
  if (!aliasTarget.value) return ''
  return device.containerAliases[aliasTarget.value.name] || ''
})
function openAlias(container) {
  if (container) {
    // 单选
    aliasTarget.value = container
    aliasBatchTargets.value = [container]
    aliasInput.value = device.containerAliases[container.name] || ''
  } else {
    // 多选：从选中坑位取每个坑位的代表容器
    const targets = []
    for (const slot of selectedSlots.value) {
      const containers = device.containers.filter(c => c.indexNum === slot.num)
      const running = containers.find(c => c.status === 'running')
      const active = running || containers[0]
      if (active) targets.push(active)
    }
    if (!targets.length) { ElMessage.warning('选中的坑位没有容器'); return }
    aliasTarget.value = targets.length === 1 ? targets[0] : null
    aliasBatchTargets.value = targets
    aliasInput.value = ''
  }
  aliasVisible.value = true
}
async function doSetAlias() {
  const alias = aliasInput.value.trim()
  if (!alias) { aliasVisible.value = false; return }
  aliasSaving.value = true
  try {
    if (aliasBatchTargets.value.length === 1) {
      await device.setAlias(aliasBatchTargets.value[0].name, alias)
    } else {
      // 批量：别名 + 坑位号后缀
      for (const c of aliasBatchTargets.value) {
        await device.setAlias(c.name, `${alias}-${c.indexNum}`)
      }
    }
    ElMessage.success('别名设置成功')
    aliasVisible.value = false
  } catch (e) {
    ElMessage.error(e.message || '设置失败')
  } finally { aliasSaving.value = false }
}
async function doRemoveAlias() {
  aliasSaving.value = true
  try {
    for (const c of aliasBatchTargets.value) {
      if (device.containerAliases[c.name]) {
        await device.removeAlias(c.name)
      }
    }
    ElMessage.success('别名已清除')
    aliasVisible.value = false
  } catch (e) {
    ElMessage.error(e.message || '清除失败')
  } finally { aliasSaving.value = false }
}

// 复制
const copyVisible = ref(false)
const copyTarget = ref(null)
const copySlot = ref(1)
const copyCount = ref(1)
const copying = ref(false)
function openCopy(container) {
  copyTarget.value = container
  copySlot.value = container.indexNum || 1
  copyCount.value = 1
  copyVisible.value = true
}
async function doCopy() {
  copying.value = true
  try {
    await device.request('container:copy', {
      name: copyTarget.value.name,
      indexNum: copySlot.value,
      count: copyCount.value
    })
    ElMessage.success('复制成功')
    copyVisible.value = false
  } catch (e) {
    ElMessage.error(e.message || '复制失败')
  } finally { copying.value = false }
}

// 多投屏
const projections = reactive([])
function openProjection(container) {
  const existing = projections.find(p => p.name === container.name)
  if (existing) { existing.visible = true; existing.container = container; return }
  projections.push({ name: container.name, container, visible: true })
}
function removeProjection(name) {
  const idx = projections.findIndex(p => p.name === name)
  if (idx !== -1) projections.splice(idx, 1)
}
function closeAllProjections() {
  projections.splice(0, projections.length)
}

// 终端
const terminalVisible = ref(false)
const terminalContainer = ref(null)
function openTerminal(container) {
  terminalContainer.value = container
  terminalVisible.value = true
}

// S5 代理
const s5Visible = ref(false)
const s5Container = ref(null)
const s5Status = reactive({ status: 0, statusText: '未启动', addr: '', type: 0 })
const s5Form = reactive({ addr: '', port: '', usr: '', pwd: '', type: '1' })
const s5Setting = ref(false)
const s5Stopping = ref(false)
const s5QuickInput = ref('')

function parseS5Quick() {
  const raw = s5QuickInput.value.trim()
  if (!raw) { ElMessage.warning('请输入 S5 信息'); return }
  const parts = raw.split(':')
  if (parts.length < 2) { ElMessage.warning('格式错误，至少需要 地址:端口'); return }
  s5Form.addr = parts[0]
  s5Form.port = parts[1]
  s5Form.usr = parts[2] || ''
  s5Form.pwd = parts[3] || ''
  ElMessage.success('已解析并填写')
}

function openS5Proxy(container) {
  if (!container) { ElMessage.warning('请选择一个运行中的容器'); return }
  s5Container.value = container
  s5Form.addr = ''; s5Form.port = ''; s5Form.usr = ''; s5Form.pwd = ''; s5Form.type = '1'
  s5QuickInput.value = ''
  Object.assign(s5Status, { status: 0, statusText: '未启动', addr: '', type: 0 })
  s5Visible.value = true
}

async function fetchS5Status() {
  if (!s5Container.value) return
  try {
    const resp = await device.request('proxy:status', { name: s5Container.value.name })
    const d = resp.data?.data || resp.data || {}
    Object.assign(s5Status, { status: d.status || 0, statusText: d.statusText || '未启动', addr: d.addr || '', type: d.type || 0 })
  } catch {}
}

async function doSetS5() {
  if (!s5Form.addr || !s5Form.port) { ElMessage.warning('请填写 IP 和端口'); return }
  s5Setting.value = true
  try {
    await device.request('proxy:set', {
      name: s5Container.value.name,
      addr: s5Form.addr,
      port: s5Form.port,
      usr: s5Form.usr,
      pwd: s5Form.pwd,
      type: s5Form.type
    })
    ElMessage.success('代理设置成功')
    await fetchS5Status()
  } catch (e) { ElMessage.error(e.message || '设置失败') }
  finally { s5Setting.value = false }
}

async function doStopS5() {
  s5Stopping.value = true
  try {
    await device.request('proxy:stop', { name: s5Container.value.name })
    ElMessage.success('代理已停止')
    await fetchS5Status()
  } catch (e) { ElMessage.error(e.message || '停止失败') }
  finally { s5Stopping.value = false }
}

// 切换机型
const switchModelVisible = ref(false)
const switchModelTarget = ref(null)
const switchModelVersion = ref('and16')
const switchModelId = ref('')
const switchModelList = ref([])
const switchModelLoading = ref(false)
const switchModelSaving = ref(false)

const switchModelFiltered = computed(() => {
  const ver = switchModelVersion.value === 'and14' ? '14' : '16'
  return switchModelList.value.filter(m => m.android_version === ver)
})

async function openSwitchModel(container) {
  if (!container) { ElMessage.warning('请选择一个容器'); return }
  switchModelTarget.value = container
  switchModelId.value = ''
  switchModelVisible.value = true
  switchModelLoading.value = true
  try {
    // 并行加载镜像列表和机型列表
    const [mirrorResp, phoneResp] = await Promise.allSettled([
      device.request('device:mirrors'),
      device.request('sdk:getPhoneModels')
    ])
    // 根据容器镜像匹配安卓版本
    let mirrors = []
    if (mirrorResp.status === 'fulfilled') mirrors = mirrorResp.value.data || []
    const image = container.image || ''
    const matchedMirror = mirrors.find(m => m.url === image)
    switchModelVersion.value = matchedMirror?.os_ver === 'and14' ? 'and14' : 'and16'
    // 机型
    if (phoneResp.status === 'fulfilled') {
      const d = phoneResp.value.data
      const pl = d?.data?.list || d?.list || d?.data || d || []
      switchModelList.value = Array.isArray(pl) ? pl : []
    } else {
      switchModelList.value = []
    }
  } catch {
    switchModelList.value = []
  } finally {
    switchModelLoading.value = false
  }
}

async function doSwitchModel() {
  switchModelSaving.value = true
  try {
    let modelId = switchModelId.value
    if (!modelId && switchModelFiltered.value.length) {
      const rand = switchModelFiltered.value[Math.floor(Math.random() * switchModelFiltered.value.length)]
      modelId = rand.id || rand.modelId || ''
    }
    await device.request('sdk:switchModel', {
      name: switchModelTarget.value.name,
      modelId
    }, 120000)
    ElMessage.success('切换机型成功')
    switchModelVisible.value = false
  } catch (e) {
    ElMessage.error(e.message || '切换机型失败')
  } finally {
    switchModelSaving.value = false
  }
}

// 批量上传
const batchUploadVisible = ref(false)
const batchUploadRunning = ref(false)
const batchUploadContainers = ref([])
const batchUploadResult = reactive({})
const batchUploadProgress = reactive({ total: 0, done: 0, fail: 0 })
const batchFileList = ref([])
const batchFileLoading = ref(false)
const batchSelectedFiles = ref([])
const batchFileTableRef = ref(null)

function formatBatchSize(b) {
  if (b < 1024) return b + ' B'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  if (b < 1024 * 1024 * 1024) return (b / (1024 * 1024)).toFixed(1) + ' MB'
  return (b / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

async function loadBatchFiles() {
  batchFileLoading.value = true
  try {
    const { data } = await api.get('/file/list')
    batchFileList.value = data.data || []
  } catch {
    batchFileList.value = []
  } finally {
    batchFileLoading.value = false
  }
}

function onBatchFileSelect(rows) {
  batchSelectedFiles.value = rows
}

function openBatchUpload() {
  const slots = selectedSlots.value
  const containers = []
  for (const slot of slots) {
    const ct = device.containers.find(c => c.indexNum === slot.num)
    if (ct) containers.push(ct)
  }
  if (!containers.length) {
    ElMessage.warning('请选择至少一个容器')
    return
  }
  batchUploadContainers.value = containers
  batchUploadRunning.value = false
  batchSelectedFiles.value = []
  Object.keys(batchUploadResult).forEach(k => delete batchUploadResult[k])
  Object.assign(batchUploadProgress, { total: 0, done: 0, fail: 0 })
  batchUploadVisible.value = true
  loadBatchFiles()
}

async function doBatchUpload() {
  if (!batchSelectedFiles.value.length) { ElMessage.warning('请选择文件'); return }
  batchUploadRunning.value = true
  const files = batchSelectedFiles.value
  const containers = batchUploadContainers.value
  const total = files.length * containers.length
  let done = 0, fail = 0
  Object.assign(batchUploadProgress, { total, done: 0, fail: 0 })
  Object.keys(batchUploadResult).forEach(k => delete batchUploadResult[k])

  for (const ct of containers) {
    batchUploadResult[ct.name] = { uploading: true }
    let containerOk = 0, containerFail = 0
    for (const file of files) {
      try {
        // 服务端直推：文件从服务器本地直接推送到容器，不经浏览器中转（支持大文件）
        await api.post(`/container/${ct.name}/push-upload`, { filename: file.name }, { timeout: 600000 })
        containerOk++
      } catch {
        containerFail++
      }
      done = containerOk + containerFail + (Object.values(batchUploadResult).filter(r => r.success || r.fail).length * files.length)
      batchUploadProgress.done = done
      batchUploadProgress.fail = fail + containerFail
    }
    if (containerFail === 0) {
      batchUploadResult[ct.name] = { success: true }
    } else {
      batchUploadResult[ct.name] = { fail: true }
    }
    fail += containerFail
  }
  batchUploadRunning.value = false
  if (fail === 0) {
    ElMessage.success(`全部上传成功（${containers.length} 个容器 × ${files.length} 个文件）`)
  } else {
    ElMessage.warning(`上传完成：${total - fail} 成功，${fail} 失败`)
  }
}
</script>
