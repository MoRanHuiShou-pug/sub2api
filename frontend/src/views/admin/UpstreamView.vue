<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('admin.upstream.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.upstream.description') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadUpstreams">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            {{ t('admin.upstream.refresh') }}
          </button>
          <button class="btn btn-primary" @click="openAddDialog">
            <Icon name="plus" size="sm" />
            {{ t('admin.upstream.addUpstream') }}
          </button>
        </div>
      </div>

      <!-- Error state -->
      <div v-if="error" class="card p-6 text-center text-red-500">
        {{ t('admin.upstream.loadFailed') }}
      </div>

      <!-- Empty state -->
      <div
        v-else-if="!loading && upstreams.length === 0"
        class="card flex flex-col items-center justify-center gap-4 py-16"
      >
        <Icon name="server" size="xl" class="text-gray-300 dark:text-gray-600" />
        <p class="text-gray-500 dark:text-gray-400">{{ t('admin.upstream.empty') }}</p>
        <button class="btn btn-primary" @click="openAddDialog">
          {{ t('admin.upstream.addUpstream') }}
        </button>
      </div>

      <!-- Cards grid -->
      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <!-- Skeleton while loading -->
        <template v-if="loading && upstreams.length === 0">
          <div v-for="n in 3" :key="n" class="card animate-pulse space-y-3 p-5">
            <div class="h-4 w-1/2 rounded bg-gray-200 dark:bg-gray-700" />
            <div class="h-3 w-3/4 rounded bg-gray-200 dark:bg-gray-700" />
            <div class="h-3 w-1/3 rounded bg-gray-200 dark:bg-gray-700" />
          </div>
        </template>

        <!-- Upstream cards -->
        <div v-for="item in upstreams" :key="item.id" class="card overflow-hidden">
          <!-- Card header -->
          <div
            class="flex items-start justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-800"
          >
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span
                  class="inline-block h-2 w-2 flex-shrink-0 rounded-full"
                  :class="item.health ? 'bg-green-500' : 'bg-red-500'"
                  :title="
                    item.health
                      ? t('admin.upstream.health.healthy')
                      : item.health_msg || t('admin.upstream.health.unhealthy')
                  "
                />
                <span class="truncate font-medium text-gray-900 dark:text-white">
                  {{ item.name }}
                </span>
              </div>
              <div class="mt-1 flex items-center gap-2">
                <span
                  class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="
                    item.platform === 'sub2api'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                      : 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300'
                  "
                >
                  {{ t(`admin.upstream.platform.${item.platform}`) }}
                </span>
                <span class="truncate text-xs text-gray-400">{{ item.base_url }}</span>
              </div>
            </div>
          </div>

          <!-- Card body -->
          <div class="space-y-3 px-5 py-4">
            <div class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-gray-400">{{
                t('admin.upstream.card.balance')
              }}</span>
              <span class="font-medium text-gray-900 dark:text-white">
                ${{ item.balance.toFixed(4) }}
              </span>
            </div>

            <div class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-gray-400">{{
                t('admin.upstream.card.groups')
              }}</span>
              <span class="font-medium text-gray-900 dark:text-white">
                {{ item.groups?.length ?? 0 }}
              </span>
            </div>

            <div v-if="item.groups && item.groups.length > 0" class="flex flex-wrap gap-1">
              <span
                v-for="g in item.groups.slice(0, 6)"
                :key="g.name"
                :title="`${g.name}: ${g.rate_multiplier}x`"
                class="inline-flex items-center gap-1 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-300"
              >
                {{ g.name }}
                <span class="text-gray-400">×{{ g.rate_multiplier }}</span>
              </span>
              <span
                v-if="item.groups.length > 6"
                class="inline-flex items-center rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-400 dark:bg-gray-800"
              >
                +{{ item.groups.length - 6 }}
              </span>
            </div>

            <div class="text-xs text-gray-400">
              {{ t('admin.upstream.card.lastSynced') }}:
              {{
                item.last_synced_at
                  ? formatRelative(item.last_synced_at)
                  : t('admin.upstream.card.never')
              }}
            </div>
          </div>

          <!-- Card actions -->
          <div
            class="flex items-center gap-1 border-t border-gray-100 px-5 py-3 dark:border-gray-800"
          >
            <button
              class="btn btn-xs btn-secondary flex-1"
              :disabled="syncingIds.has(item.id)"
              @click="triggerSync(item)"
            >
              <Icon
                name="refresh"
                size="xs"
                :class="{ 'animate-spin': syncingIds.has(item.id) }"
              />
              {{ t('admin.upstream.card.sync') }}
            </button>
            <button class="btn btn-xs btn-secondary" @click="openEditDialog(item)">
              <Icon name="edit" size="xs" />
              {{ t('admin.upstream.card.edit') }}
            </button>
            <button class="btn btn-xs btn-danger-outline" @click="openDeleteDialog(item)">
              <Icon name="trash" size="xs" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Add / Edit dialog -->
    <BaseDialog
      :show="showFormDialog"
      :title="
        editTarget ? t('admin.upstream.dialog.editTitle') : t('admin.upstream.dialog.addTitle')
      "
      @close="showFormDialog = false"
    >
      <form class="space-y-4" @submit.prevent="submitForm">
        <!-- Platform (only on create) -->
        <div v-if="!editTarget">
          <label class="input-label">{{ t('admin.upstream.fields.platform') }}</label>
          <Select v-model="form.platform" :options="platformOptions" />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstream.fields.name') }}</label>
          <input
            v-model.trim="form.name"
            type="text"
            class="input"
            :placeholder="t('admin.upstream.placeholders.name')"
            required
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstream.fields.baseUrl') }}</label>
          <input
            v-model.trim="form.base_url"
            type="url"
            class="input"
            :placeholder="t('admin.upstream.placeholders.baseUrl')"
            required
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstream.fields.email') }}</label>
          <input
            v-model.trim="form.email"
            type="email"
            class="input"
            :placeholder="t('admin.upstream.placeholders.email')"
            required
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstream.fields.password') }}</label>
          <input
            v-model="form.password"
            type="password"
            class="input"
            :placeholder="t('admin.upstream.placeholders.password')"
            :required="!editTarget"
            autocomplete="new-password"
          />
          <p v-if="editTarget" class="mt-1 text-xs text-gray-400">留空则保持原密码不变</p>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button type="button" class="btn btn-secondary" @click="showFormDialog = false">
            {{ t('admin.upstream.dialog.cancel') }}
          </button>
          <button type="submit" class="btn btn-primary" :disabled="formSubmitting">
            {{ editTarget ? t('admin.upstream.dialog.save') : t('admin.upstream.dialog.add') }}
          </button>
        </div>
      </form>
    </BaseDialog>

    <!-- Delete confirm dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.upstream.dialog.deleteTitle')"
      :message="t('admin.upstream.dialog.deleteMessage', { name: deleteTarget?.name ?? '' })"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import { adminAPI } from '@/api/admin'
import type { Upstream, CreateUpstreamRequest, UpdateUpstreamRequest } from '@/api/admin/upstream'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

// ── State ─────────────────────────────────────────────────────────────────────
const upstreams = ref<Upstream[]>([])
const loading = ref(false)
const error = ref(false)
const syncingIds = ref(new Set<string>())

// Form dialog
const showFormDialog = ref(false)
const editTarget = ref<Upstream | null>(null)
const formSubmitting = ref(false)
const form = reactive<{
  platform: 'sub2api' | 'newapi'
  name: string
  base_url: string
  email: string
  password: string
}>({
  platform: 'sub2api',
  name: '',
  base_url: '',
  email: '',
  password: '',
})

// Delete dialog
const showDeleteDialog = ref(false)
const deleteTarget = ref<Upstream | null>(null)

// ── Options ───────────────────────────────────────────────────────────────────
const platformOptions = [
  { value: 'sub2api', label: 'Sub2API' },
  { value: 'newapi', label: 'NewAPI' },
]

// ── Data loading ──────────────────────────────────────────────────────────────
async function loadUpstreams() {
  loading.value = true
  error.value = false
  try {
    upstreams.value = await adminAPI.upstreams.list()
  } catch (err: any) {
    error.value = true
    appStore.showError(err?.message || t('admin.upstream.loadFailed'))
  } finally {
    loading.value = false
  }
}

// ── Sync ──────────────────────────────────────────────────────────────────────
async function triggerSync(upstream: Upstream) {
  syncingIds.value = new Set(syncingIds.value).add(upstream.id)
  try {
    const updated = await adminAPI.upstreams.sync(upstream.id)
    const idx = upstreams.value.findIndex((u) => u.id === upstream.id)
    if (idx !== -1) upstreams.value[idx] = updated
    appStore.showSuccess(t('admin.upstream.syncSuccess'))
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.upstream.syncFailed'))
  } finally {
    const next = new Set(syncingIds.value)
    next.delete(upstream.id)
    syncingIds.value = next
  }
}

// ── Add / Edit ────────────────────────────────────────────────────────────────
function openAddDialog() {
  editTarget.value = null
  Object.assign(form, { platform: 'sub2api', name: '', base_url: '', email: '', password: '' })
  showFormDialog.value = true
}

function openEditDialog(upstream: Upstream) {
  editTarget.value = upstream
  Object.assign(form, {
    platform: upstream.platform,
    name: upstream.name,
    base_url: upstream.base_url,
    email: upstream.email,
    password: '',
  })
  showFormDialog.value = true
}

async function submitForm() {
  formSubmitting.value = true
  try {
    if (editTarget.value) {
      const payload: UpdateUpstreamRequest = {
        name: form.name,
        base_url: form.base_url,
        email: form.email,
      }
      if (form.password) payload.password = form.password
      const updated = await adminAPI.upstreams.update(editTarget.value.id, payload)
      const idx = upstreams.value.findIndex((u) => u.id === editTarget.value!.id)
      if (idx !== -1) upstreams.value[idx] = updated
      appStore.showSuccess(t('admin.upstream.updateSuccess'))
    } else {
      const created = await adminAPI.upstreams.create(form as CreateUpstreamRequest)
      upstreams.value.unshift(created)
      appStore.showSuccess(t('admin.upstream.createSuccess'))
    }
    showFormDialog.value = false
  } catch (err: any) {
    appStore.showError(
      err?.message ||
        (editTarget.value ? t('admin.upstream.updateFailed') : t('admin.upstream.createFailed')),
    )
  } finally {
    formSubmitting.value = false
  }
}

// ── Delete ────────────────────────────────────────────────────────────────────
function openDeleteDialog(upstream: Upstream) {
  deleteTarget.value = upstream
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  try {
    await adminAPI.upstreams.deleteUpstream(deleteTarget.value.id)
    upstreams.value = upstreams.value.filter((u) => u.id !== deleteTarget.value!.id)
    appStore.showSuccess(t('admin.upstream.deleteSuccess'))
    showDeleteDialog.value = false
  } catch (err: any) {
    appStore.showError(err?.message || t('admin.upstream.deleteFailed'))
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function formatRelative(iso: string): string {
  const d = new Date(iso)
  const diff = Date.now() - d.getTime()
  const mins = Math.floor(diff / 60_000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

// ── Init ──────────────────────────────────────────────────────────────────────
onMounted(loadUpstreams)
</script>
