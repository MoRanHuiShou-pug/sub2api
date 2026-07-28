/**
 * Admin Upstream Sessions API endpoints
 * Manages upstream Sub2API / NewAPI instance connections
 */

import { apiClient } from '../client'

export interface UpstreamGroup {
  name: string
  platform?: string
  rate_multiplier: number
  description?: string
}

export interface Upstream {
  id: string
  platform: 'sub2api' | 'newapi'
  name: string
  base_url: string
  email: string
  health: boolean
  health_msg?: string
  balance: number
  groups: UpstreamGroup[]
  last_synced_at?: string
  status: string
  created_at: string
  updated_at: string
}

export interface CreateUpstreamRequest {
  platform: 'sub2api' | 'newapi'
  name: string
  base_url: string
  email: string
  password: string
}

export interface UpdateUpstreamRequest {
  name?: string
  base_url?: string
  email?: string
  password?: string
  status?: string
}

/**
 * List all upstream instances
 */
export async function list(): Promise<Upstream[]> {
  const { data } = await apiClient.get<Upstream[]>('/admin/upstreams')
  return data
}

/**
 * Create a new upstream instance
 */
export async function create(payload: CreateUpstreamRequest): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>('/admin/upstreams', payload)
  return data
}

/**
 * Get a single upstream instance by ID
 */
export async function getById(id: string): Promise<Upstream> {
  const { data } = await apiClient.get<Upstream>(`/admin/upstreams/${id}`)
  return data
}

/**
 * Update an upstream instance
 */
export async function update(id: string, payload: UpdateUpstreamRequest): Promise<Upstream> {
  const { data } = await apiClient.put<Upstream>(`/admin/upstreams/${id}`, payload)
  return data
}

/**
 * Delete an upstream instance
 */
export async function deleteUpstream(id: string): Promise<void> {
  await apiClient.delete(`/admin/upstreams/${id}`)
}

/**
 * Trigger an immediate sync for an upstream instance
 */
export async function sync(id: string): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>(`/admin/upstreams/${id}/sync`)
  return data
}

export default { list, create, getById, update, deleteUpstream, sync }
