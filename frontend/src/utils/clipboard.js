// 复制文本到剪贴板（兼容 HTTP 环境，navigator.clipboard 在非 HTTPS 下为 undefined）
export async function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {}
  }
  return legacyCopy(text)
}

// 兜底方案：临时 textarea + execCommand('copy')
function legacyCopy(text) {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.top = '-9999px'
  ta.style.left = '-9999px'
  ta.setAttribute('readonly', '')
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {}
  document.body.removeChild(ta)
  return ok
}
