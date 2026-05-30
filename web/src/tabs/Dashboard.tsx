import { useEffect, useState } from 'react'
import { api, type Stats } from '../api'
import { Stat } from '../ui'

export default function Dashboard({ currency }: { currency: string }) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [err, setErr] = useState('')

  useEffect(() => {
    api.stats().then(setStats).catch((e) => setErr(String(e)))
  }, [])

  if (err) return <div className="text-red-400">{err}</div>
  if (!stats) return <div className="text-sand-300/60">Loading…</div>

  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
      <Stat label="Players" value={stats.linked_accounts} />
      <Stat label="Items" value={stats.catalog_items} />
      <Stat label="Kits" value={stats.kits} />
      <Stat label={`${currency} in circulation`} value={stats.currency_in_circulation.toLocaleString()} />
      <Stat label="Purchases" value={stats.purchases} />
    </div>
  )
}
