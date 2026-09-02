import { ref, onMounted, onUnmounted } from 'vue'

export function useFocusTrap(containerRef) {
  const isActive = ref(false)

  function getFocusableElements() {
    if (!containerRef.value) return []
    return Array.from(
      containerRef.value.querySelectorAll(
        'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      )
    )
  }

  function handleKeydown(e) {
    if (!isActive.value || e.key !== 'Tab') return

    const focusable = getFocusableElements()
    if (focusable.length === 0) return

    const first = focusable[0]
    const last = focusable[focusable.length - 1]

    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault()
        last.focus()
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
  }

  function handleEscape(e) {
    if (!isActive.value || e.key !== 'Escape') return
    deactivate()
  }

  function activate() {
    isActive.value = true
    document.addEventListener('keydown', handleKeydown)
    document.addEventListener('keydown', handleEscape)
    const focusable = getFocusableElements()
    if (focusable.length > 0) {
      focusable[0].focus()
    }
  }

  function deactivate() {
    isActive.value = false
    document.removeEventListener('keydown', handleKeydown)
    document.removeEventListener('keydown', handleEscape)
  }

  onUnmounted(() => {
    deactivate()
  })

  return { isActive, activate, deactivate }
}
