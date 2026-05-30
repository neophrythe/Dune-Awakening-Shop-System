import { useEffect, useState } from 'react'
import { api, type Txn } from '../api'
import { Card } from '../ui'

const KIND_COLOR: Record<string, string> = {
  earn: 'text-emerald-300',
  spend: 'text-sand-300',
  adjust: 'text-sky-300',
}

export default function Transactions({ currency }: { currency: string }) {
  const [txns, setTxns] = useState<Txn[]>([])
  const [err, setErr] = useState('')

  useEffect(() => { api.transactions().then(setTxns).catch((e) => setErr(String(e))) }, [])

  if (err) return <div className="text-red-400">{err}</div>

  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-night-950/60 text-left text-sand-300/70">
          <tr>
            <th className="p-3">When</th><th className="p-3">Player</th><th className="p-3">Kind</th>
            <th className="p-3">Amount</th><th className="p-3">Delivery</th><th className="p-3">Note</th>
          </tr>
        </thead>
        <tbody>
          {txns.map((t) => (
            <tr key={t.id} className="border-t border-sand-900/50">
              <td className="p-3 text-sand-300/50">{new Date(t.created_at).toLocaleString()}</td>
              <td className="p-3 text-sand-100">{t.character_name || '—'}</td>
              <td className={`p-3 ${KIND_COLOR[t.kind] ?? ''}`}>{t.kind}</td>
              <td className={`p-3 ${t.amount < 0 ? 'text-red-300' : 'text-emerald-300'}`}>
                {t.amount > 0 ? '+' : ''}{t.amount} {currency}
              </td>
              <td className="p-3 text-sand-300/60">{t.delivery || '—'}</td>
              <td className="p-3 text-sand-300/70">{t.note}</td>
            </tr>
          ))}
          {txns.length === 0 && <tr><td colSpan={6} className="p-6 text-center text-sand-300/40">No transactions yet.</td></tr>}
        </tbody>
      </table>
    </Card>
  )
}
