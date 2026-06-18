<template>
  <div>
    <div
      class="flex gap-1 border-b border-hairline -mb-px"
      role="tablist"
    >
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px cursor-pointer"
        :class="selected === tab.id
          ? 'border-ink text-ink'
          : 'border-transparent text-mute hover:text-body hover:border-ash'"
        :aria-selected="selected === tab.id"
        role="tab"
        @click="selected = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="pt-4 pb-4">
      <slot :name="selected" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  tabs: { type: Array, required: true },
  modelValue: { type: String, default: undefined },
})

const emit = defineEmits(['update:modelValue'])

const selected = ref(props.modelValue || props.tabs[0]?.id)

watch(() => props.modelValue, v => { if (v !== undefined) selected.value = v })
watch(selected, v => emit('update:modelValue', v))
</script>
