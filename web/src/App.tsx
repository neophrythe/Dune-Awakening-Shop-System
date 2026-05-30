import { useEffect, useState } from 'react'
import { api } from './api'
import Login from './Login'
import Dashboard from './tabs/Dashboard'
import Items from './tabs/Items'
import Kits from './tabs/Kits'
import Accounts from './tabs/Accounts'
import Transactions from './tabs/Transactions'
import { Button } from './ui'

type Tab = 'dashboard' | 'items' | 'kits' | 'accounts' | 'transactions'

const TABS: { id: Tab; label: string; icon: string }[] = [
  { id: 'dashboard', label: 'Overview', icon: '📊' },
  { id: 'items', label: 'Items', icon: '🛒' },
  { id: 'kits', label: 'Kits', icon: '📦' },
  { id: 'accounts', label: 'Players', icon: '🧑' },
  { id: 'transactions', label: 'Ledger', icon: '🧾' },
]

export default function App() {
  const [authed, setAuthed] = useState<boolean | null>(null)
  const [currency, setCurrency] = useState('Solari')
  const [tab, setTab] = useState<Tab>('dashboard')

  useEffect(() => {
    api.session().then((s) => {
      setAuthed(s.authenticated)
      if (s.currency) setCurrency(s.currency)
    }).catch(() => setAuthed(false))
  }, [])

  if (authed === null) return <div className="p-8 text-sand-300/60">Loading…</div>
  if (!authed) return <Login onLogin={(c) => { setCurrency(c); setAuthed(true) }} />

  const logout = async () => {
    await api.logout().catch(() => {})
    setAuthed(false)
  }

  return (
    <div className="mx-auto max-w-6xl p-4 sm:p-6">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="font-display text-2xl text-sand-200">🏜️ Dune Awakening Shop</h1>
          <p className="text-sm text-sand-300/60">Admin dashboard</p>
        </div>
        <Button variant="ghost" onClick={logout}>Sign out</Button>
      </header>

      <nav className="mb-6 flex flex-wrap gap-2">
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`rounded-lg px-4 py-2 text-sm transition ${
              tab === t.id
                ? 'bg-sand-400 text-night-950'
                : 'border border-sand-800/60 text-sand-200 hover:bg-sand-900/40'
            }`}
          >
            <span className="mr-1">{t.icon}</span>{t.label}
          </button>
        ))}
      </nav>

      {tab === 'dashboard' && <Dashboard currency={currency} />}
      {tab === 'items' && <Items currency={currency} />}
      {tab === 'kits' && <Kits currency={currency} />}
      {tab === 'accounts' && <Accounts currency={currency} />}
      {tab === 'transactions' && <Transactions currency={currency} />}
    </div>
  )
}
