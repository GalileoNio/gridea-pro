<template>
    <div
        class="memo-input-wrapper bg-card/50 border border-border/50 rounded-xl transition-all duration-200 ring-offset-background focus-within:ring-1 focus-within:ring-primary/10 relative overflow-visible">
        <div class="px-6 py-6 relative">
            <!-- Typewriter placeholder -->
            <div v-if="!content && !isFocused" class="absolute inset-0 px-6 py-6 pointer-events-none text-sm leading-5 tracking-wider text-muted-foreground/50">
                {{ typewriterText }}<span class="animate-blink">|</span>
            </div>
            <textarea
ref="textareaRef" v-model="content"
                class="w-full bg-transparent border-none focus:ring-0 resize-none p-0 min-h-[80px] text-sm leading-5 tracking-wider text-foreground outline-none relative z-10"
                :rows="1" @input="handleInput" @keydown="handleKeydown" @click="handleInput"
                @focus="handleFocus" @blur="handleBlur" />

            <!-- Tag Suggestions Dropdown -->
            <div
v-if="showTagSuggestions && filteredTags.length > 0"
                class="absolute z-500 bg-card text-popover-foreground border border-border rounded-md shadow-md min-w-[120px] max-h-[200px] overflow-y-auto"
                :style="suggestionStyle">
                <div
v-for="(tag, index) in filteredTags" :key="tag.name"
                    class="px-3 py-1.5 text-xs cursor-pointer hover:bg-primary/10 hover:text-primary transition-colors flex items-center justify-between"
                    :class="{ 'bg-primary/10 text-primary text-xs': index === selectedTagIndex }"
                    @click="selectTag(tag.name)">
                    <span># {{ tag.name }}</span>
                    <span class="text-xs text-muted-foreground ml-2 opacity-50">{{ tag.count }}</span>
                </div>
            </div>
        </div>
        <div class="flex items-center justify-between px-4 pb-3 pt-2 border-t border-border/30">
            <!-- 左下角：Markdown 支持提示 + 发布时间设置 -->
            <div class="flex items-center gap-1.5">
                <span class="text-muted-foreground/40" :title="t('memo.markdownSupported')">
                    <MarkdownIcon class="w-[18px] h-[11px]" />
                </span>
                <Popover @update:open="onPickerOpen">
                    <PopoverTrigger as-child>
                        <button
type="button" :title="t('memo.setPublishTime')"
                            class="flex items-center gap-1 h-6 px-1.5 rounded-md text-xs text-muted-foreground/60 hover:text-primary hover:bg-primary/10 transition-colors cursor-pointer">
                            <CalendarIcon class="w-3.5 h-3.5" />
                            <span v-if="publishDateTime" class="tabular-nums">{{ publishDateTime }}</span>
                        </button>
                    </PopoverTrigger>
                    <PopoverContent class="w-auto p-0" align="start">
                        <Calendar
:model-value="(calendarValue as any)" show-week-number
                            @update:model-value="(val: any) => (calendarValue = val)" />
                        <div class="border-t p-3">
                            <div class="relative">
                                <ClockIcon class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground z-10" />
                                <Input
v-model="dateTimeDraft" placeholder="YYYY-MM-DD HH:mm:ss"
                                    class="h-9 pl-9 selection:bg-primary selection:text-primary-foreground"
                                    @blur="commitDateTime" @keyup.enter="commitDateTime" />
                            </div>
                        </div>
                    </PopoverContent>
                </Popover>
                <button
v-if="publishDateTime" type="button" :title="t('common.clear')"
                    class="text-muted-foreground/40 hover:text-destructive transition-colors cursor-pointer"
                    @click="clearPublishDateTime">
                    <XMarkIcon class="w-3.5 h-3.5" />
                </button>
            </div>
            <!-- 右下角：操作按钮 -->
            <div class="flex items-center gap-2">
                <Button
v-if="isEditing" variant="outline" size="sm" class="h-7 px-4 text-xs justify-center rounded-full bg-primary/5 border border-primary/20 text-primary/80 hover:bg-primary/5 hover:text-primary cursor-pointer"
                    @click="handleCancel">
                    {{ t('common.cancel') }}
                </Button>

                <Button
variant="default" size="sm"
                    class="h-7 px-4 rounded-full text-[10px] font-medium transition-all shadow-sm hover:shadow-md"
                    :disabled="!canSubmit" @click="handleSubmit">
                    <PaperAirplaneIcon class="w-3 h-3 mr-1 mb-0.5 -rotate-45" />
                    {{ submitBtnText }}
                </Button>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import { CalendarDate, type DateValue } from '@internationalized/date'
import { Button } from '@/components/ui/button/index'
import { Input } from '@/components/ui/input'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useMemoStore } from '@/stores/memo'
import { PaperAirplaneIcon, CalendarIcon, ClockIcon, XMarkIcon } from '@heroicons/vue/24/outline'

dayjs.extend(customParseFormat)

// Markdown 标志：heroicons 无此图标，内联官方 Markdown mark（纯展示）
const MarkdownIcon = {
  template: `<svg viewBox="0 0 208 128" xmlns="http://www.w3.org/2000/svg"><rect width="198" height="118" x="5" y="5" ry="10" fill="none" stroke="currentColor" stroke-width="10"/><path fill="currentColor" d="M30 98V30h20l20 25 20-25h20v68H90V59L70 84 50 59v39zm125 0l-30-33h20V30h20v35h20z"/></svg>`,
}

interface Props {
    placeholder?: string
    submitText?: string
    isEditing?: boolean
}

const props = withDefaults(defineProps<Props>(), {
    placeholder: '',
    submitText: '',
    isEditing: false,
})

const { t } = useI18n()
const memoStore = useMemoStore()
const content = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const submitBtnText = computed(() => props.submitText || t('memo.publish'))

// ── 发布时间 ──────────────────────────────────────────────
// publishDateTime：已「明确设定」的发布时间。空 = 发布时用当前时间。
//   只有用户真正操作过（在日历选了日期 / 改了输入框）才写入，
//   仅打开面板看一眼不算。决定图标显示与提交的值。
// dateTimeDraft：时间选择面板内的工作副本，打开面板时初始化。
// 内部统一用 'YYYY-MM-DD HH:mm:ss'，提交时转 ISO 给后端。
const publishDateTime = ref('')
const dateTimeDraft = ref('')

// 日历面板与草稿字符串的桥接（与文章设置抽屉同套思路）
const calendarValue = computed<DateValue | undefined>({
  get: () => {
    const d = dateTimeDraft.value ? dayjs(dateTimeDraft.value, 'YYYY-MM-DD HH:mm:ss', true) : null
    return d && d.isValid() ? new CalendarDate(d.year(), d.month() + 1, d.date()) : undefined
  },
  set: (val) => {
    if (!val) return
    // 选日期时保留草稿里已有的时分秒
    const base = dateTimeDraft.value && dayjs(dateTimeDraft.value, 'YYYY-MM-DD HH:mm:ss', true).isValid()
      ? dayjs(dateTimeDraft.value, 'YYYY-MM-DD HH:mm:ss', true)
      : dayjs()
    const next = base.year(val.year).month(val.month - 1).date(val.day).format('YYYY-MM-DD HH:mm:ss')
    dateTimeDraft.value = next
    publishDateTime.value = next // 选了日期 = 明确设定
  },
})

// 输入框：输入中途的非法串只改草稿，失焦/回车才校验。合法即视为明确设定。
function commitDateTime() {
  const parsed = dayjs(dateTimeDraft.value, 'YYYY-MM-DD HH:mm:ss', true)
  if (parsed.isValid()) {
    dateTimeDraft.value = parsed.format('YYYY-MM-DD HH:mm:ss')
    publishDateTime.value = dateTimeDraft.value // 改了输入框 = 明确设定
  } else {
    // 非法：回滚到已设定值，未设定过则回到当前时间起点
    dateTimeDraft.value = publishDateTime.value || dayjs().format('YYYY-MM-DD HH:mm:ss')
  }
}

function clearPublishDateTime() {
  publishDateTime.value = ''
  dateTimeDraft.value = ''
}

// 打开面板时初始化草稿：已设定过就用设定值，否则用当前时间作为起点。
// 起点只是展示用，不写入 publishDateTime——不真正改动就不算「已设定」。
function onPickerOpen(open: boolean) {
  if (open) {
    dateTimeDraft.value = publishDateTime.value || dayjs().format('YYYY-MM-DD HH:mm:ss')
  }
}

// Typewriter placeholder
const placeholderKeys = [
  'memo.placeholder.thinking',
  'memo.placeholder.idea',
  'memo.placeholder.discovery',
  'memo.placeholder.writeDown',
  'memo.placeholder.oneLine',
  'memo.placeholder.tagTip',
  'memo.placeholder.markdownTip',
  'memo.placeholder.imageTip',
]

const typewriterText = ref('')
const isFocused = ref(false)
let typewriterTimer: ReturnType<typeof setTimeout> | null = null
let currentMsgIndex = Math.floor(Math.random() * placeholderKeys.length)
let currentCharIndex = 0
let isDeleting = false

function typewriterTick() {
  if (isFocused.value) return // 聚焦时停止打字

  const msg = t(placeholderKeys[currentMsgIndex])

  if (!isDeleting) {
    currentCharIndex++
    typewriterText.value = msg.slice(0, currentCharIndex)

    if (currentCharIndex >= msg.length) {
      typewriterTimer = setTimeout(() => {
        isDeleting = true
        typewriterTick()
      }, 3500)
      return
    }
    typewriterTimer = setTimeout(typewriterTick, 80 + Math.random() * 40)
  } else {
    currentCharIndex--
    typewriterText.value = msg.slice(0, currentCharIndex)

    if (currentCharIndex <= 0) {
      isDeleting = false
      currentMsgIndex = (currentMsgIndex + 1) % placeholderKeys.length
      typewriterTimer = setTimeout(typewriterTick, 800)
      return
    }
    typewriterTimer = setTimeout(typewriterTick, 30)
  }
}

function startTypewriter() {
  stopTypewriter()
  currentCharIndex = 0
  isDeleting = false
  typewriterTick()
}

function stopTypewriter() {
  if (typewriterTimer) {
    clearTimeout(typewriterTimer)
    typewriterTimer = null
  }
}

function handleFocus() {
  isFocused.value = true
  stopTypewriter()
}

function handleBlur() {
  isFocused.value = false
  if (!content.value) {
    startTypewriter()
  }
}

const isMac = computed(() => navigator.platform.toUpperCase().indexOf('MAC') >= 0)

const canSubmit = computed(() => content.value.trim().length > 0)

function autoResize() {
    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.style.height = 'auto'
            textareaRef.value.style.height = Math.min(textareaRef.value.scrollHeight, 200) + 'px'
        }
    })
}

// Tag Autocomplete
const showTagSuggestions = ref(false)
const suggestionStyle = ref({ top: '0px', left: '0px' })
const currentTagQuery = ref('')
const selectedTagIndex = ref(0)
const cursorPosition = ref(0)
const hashIndex = ref(-1)

const filteredTags = computed(() => {
    if (!currentTagQuery.value) return memoStore.tagStats
    return memoStore.tagStats.filter(tag =>
        tag.name.toLowerCase().includes(currentTagQuery.value.toLowerCase())
    )
})

function handleInput() {
    autoResize()
    checkTagTrigger()
}

// 简单的光标位置计算 (针对textarea)
// 这是一个简化的实现，为了更精确的效果通常需要创建一个隐藏的 div 来模拟
function getCaretCoordinates() {
    if (!textareaRef.value) return { top: 0, left: 0 }

    // 简单估算，实际生产中建议使用专门的库如 textarea-caret
    // 这里我们暂时固定显示在输入框上方或跟随光标的大致位置
    // 由于精确计算 textarea 光标位置比较复杂，我们先用一个简单策略：
    // 显示在 textarea 的顶部，水平位置稍微偏移
    // 或者，我们可以引入一个库，但为了保持无依赖，我们尝试简单定位

    return { top: '40px', left: '20px' }
}



function checkTagTrigger() {
    if (!textareaRef.value) return

    const cursor = textareaRef.value.selectionStart
    const text = content.value

    // 向前查找 #
    // 匹配模式： ... #tag
    // 1. 找到光标前的最后一个 #
    const lastHash = text.lastIndexOf('#', cursor - 1)

    if (lastHash === -1) {
        showTagSuggestions.value = false
        return
    }

    // 检查 # 和光标之间是否有空格（除了 # 后面的那个位置外，通常标签不包含空格）
    // 简单起见，如果 # 和光标之间有换行或空格，则视为结束标签输入
    const textBetween = text.slice(lastHash + 1, cursor)
    if (/\s/.test(textBetween)) {
        showTagSuggestions.value = false
        return
    }

    // 更新状态
    hashIndex.value = lastHash
    currentTagQuery.value = textBetween
    showTagSuggestions.value = true
    selectedTagIndex.value = 0

    // 计算位置 (这里简化处理，显示在输入框左下方，稍微偏移)
    // 理想情况下应该跟随光标
    // 为了简单且不引入大库，我们让它显示在 textarea 内部的左上角，或者跟随文字流
    // 我们暂时先硬编码位置，稍后如果用户需要精确跟随再优化
    // 实际上，为了用户体验，我们应该尽量让它跟随。
    // 这里尝试一种基于 text measurement 的简单方法

    // 临时方案：显示在输入区域下方
    suggestionStyle.value = {
        top: '60px', // 估算值
        left: '24px'
    }
}

function selectTag(tagName: string) {
    if (hashIndex.value === -1 || !textareaRef.value) return

    const before = content.value.slice(0, hashIndex.value)
    const after = content.value.slice(textareaRef.value.selectionStart)

    // 插入标签并加空格
    const newContent = `${before}#${tagName} ${after}`
    content.value = newContent

    showTagSuggestions.value = false

    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.focus()
            // 移动光标到标签后
            const newCursorPos = hashIndex.value + tagName.length + 2 // +2 for # and space
            textareaRef.value.setSelectionRange(newCursorPos, newCursorPos)
            autoResize()
        }
    })
}

function handleKeydown(event: KeyboardEvent) {
    if (showTagSuggestions.value && filteredTags.value.length > 0) {
        if (event.key === 'ArrowDown') {
            event.preventDefault()
            selectedTagIndex.value = (selectedTagIndex.value + 1) % filteredTags.value.length
            return
        }
        if (event.key === 'ArrowUp') {
            event.preventDefault()
            selectedTagIndex.value = (selectedTagIndex.value - 1 + filteredTags.value.length) % filteredTags.value.length
            return
        }
        if (event.key === 'Enter' || event.key === 'Tab') {
            event.preventDefault()
            selectTag(filteredTags.value[selectedTagIndex.value].name)
            return
        }
        if (event.key === 'Escape') {
            showTagSuggestions.value = false
            return
        }
    }

    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
        event.preventDefault()
        handleSubmit()
    }
}

// ... existing handleSubmit, handleCancel, etc. ...

function handleSubmit() {
    if (!canSubmit.value) return

    // 空 = 用当前时间；否则转 ISO（RFC3339）给后端解析
    const createdAt = publishDateTime.value
        ? dayjs(publishDateTime.value, 'YYYY-MM-DD HH:mm:ss').toISOString()
        : ''
    emit('submit', content.value.trim(), createdAt)
    // Keep content if in editing mode, cleaner for parent to handle clear
    if (!props.isEditing) {
        content.value = ''
        publishDateTime.value = ''
        dateTimeDraft.value = ''
        showTagSuggestions.value = false // Reset suggestions
        nextTick(() => {
            if (textareaRef.value) {
                textareaRef.value.style.height = 'auto'
            }
        })
    }
}

function handleCancel() {
    emit('cancel')
    content.value = ''
    showTagSuggestions.value = false // Reset suggestions
    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.style.height = 'auto'
        }
    })
}

const emit = defineEmits<{
    submit: [content: string, createdAt: string]
    cancel: []
}>()

// Expose method to set content
const setContent = (text: string) => {
    content.value = text
    autoResize()
}

// 编辑闪念时回填发布时间（接收 ISO 字符串，内部转 'YYYY-MM-DD HH:mm:ss'）
const setDateTime = (iso: string) => {
    const d = dayjs(iso)
    publishDateTime.value = d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : ''
}

const clearContent = () => {
    content.value = ''
    publishDateTime.value = ''
    dateTimeDraft.value = ''
    showTagSuggestions.value = false
    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.style.height = 'auto'
        }
    })
}

defineExpose({
    setContent,
    setDateTime,
    clearContent
})

onMounted(() => {
    autoResize()
    startTypewriter()
})

onUnmounted(() => {
    stopTypewriter()
})
</script>

<style scoped>
.animate-blink {
    animation: blink 1s step-end infinite;
}
@keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0; }
}
</style>
