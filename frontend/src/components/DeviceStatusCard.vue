<template>
  <el-card class="device-status-card" :body-style="{ padding: '16px' }">
    <template #header>
      <span style="font-weight: bold">{{ title }}</span>
    </template>
    <div style="text-align: center">
      <div style="font-size: 28px; font-weight: bold; color: #409eff">{{ displayValue }}</div>
      <div style="font-size: 12px; color: #a0a0a0; margin-top: 4px">{{ subtitle }}</div>
      <el-progress v-if="showProgress" :percentage="percentage" :color="progressColor" :show-text="false"
        style="margin-top: 8px" />
    </div>
  </el-card>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: String,
  value: [String, Number],
  subtitle: { type: String, default: '' },
  unit: { type: String, default: '' },
  max: { type: Number, default: 0 },
  showProgress: { type: Boolean, default: false },
  warningThreshold: { type: Number, default: 70 },
  dangerThreshold: { type: Number, default: 90 }
})

const displayValue = computed(() => props.value + props.unit)

const percentage = computed(() => {
  if (!props.max) return 0
  return Math.min(Math.round((Number(props.value) / props.max) * 100), 100)
})

const progressColor = computed(() => {
  if (percentage.value >= props.dangerThreshold) return '#f56c6c'
  if (percentage.value >= props.warningThreshold) return '#e6a23c'
  return '#67c23a'
})
</script>

<style scoped>
.device-status-card {
  border-color: var(--border-color);
}
</style>
