// Typed API client for the dashboard. All calls are same-origin and rely on the
// HttpOnly session cookie set by /api/login.

export interface Stats {
  linked_accounts: number
  catalog_items: number
  kits: number
  currency_in_circulation: number
  purchases: number
}

export interface Item {
  id: number
  game_item_id: string
  name: string
  description: string
  category: string
  price: number
  quantity: number
  stock: number | null
  enabled: boolean
}

export interface KitItem {
  id: number
  kit_id: number
  game_item_id: string
  name: string
  quantity: number
}

export interface Kit {
  id: number
  name: string
  description: string
  category: string
  price: number
  stock: number | null
  enabled: boolean
  items: KitItem[]
}

export interface Account {
  id: number
  discord_user_id: string
  game_account_id: string
  character_name: string
  linked_at: string
  balance: number
}

export interface Txn {
  id: number
  linked_account_id: number
  kind: string
  amount: number
  delivery: string
  note: string
  created_at: string
  character_name: string
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'same-origin',
  })
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const j = await res.json()
      if (j.error) msg = j.error
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  session: () => req<{ authenticated: boolean; currency: string }>('GET', '/api/session'),
  login: (user: string, password: string) =>
    req<{ ok: boolean; currency: string }>('POST', '/api/login', { user, password }),
  logout: () => req<{ ok: boolean }>('POST', '/api/logout'),
  stats: () => req<Stats>('GET', '/api/stats'),
  items: () => req<Item[]>('GET', '/api/items'),
  upsertItem: (it: Partial<Item>) => req<{ id: number }>('POST', '/api/items', it),
  setItemEnabled: (id: number, enabled: boolean) =>
    req<{ ok: boolean }>('POST', `/api/items/${id}/enabled`, { enabled }),
  kits: () => req<Kit[]>('GET', '/api/kits'),
  createKit: (k: Partial<Kit>) => req<{ id: number }>('POST', '/api/kits', k),
  addKitItem: (id: number, it: Partial<KitItem>) =>
    req<{ ok: boolean }>('POST', `/api/kits/${id}/items`, it),
  setKitEnabled: (id: number, enabled: boolean) =>
    req<{ ok: boolean }>('POST', `/api/kits/${id}/enabled`, { enabled }),
  accounts: () => req<Account[]>('GET', '/api/accounts'),
  transactions: () => req<Txn[]>('GET', '/api/transactions'),
}
