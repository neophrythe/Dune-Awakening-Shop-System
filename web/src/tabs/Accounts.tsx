import { useEffect, useState } from 'react'
import { api, type Account } from '../api'
import { Card } from '../ui'

export default function Accounts({ currency }: { currency: string }) {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [err, setErr] = useState('')

  useEffect(() => { api.accounts().then(setAccounts).catch((e) => setErr(String(e))) }, [])

  if (err) return <div className="text-red-400">{err}</div>

  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-night-950/60 text-left text-sand-300/70">
          <tr>
            <th className="p-3">Character</th><th className="p-3">Discord</th>
            <th className="p-3">Balance</th><th className="p-3">Linked</th>
          </tr>
        </thead>
        <tbody>
          {accounts.map((a) => (
            <tr key={a.id} className="border-t border-sand-900/50">
              <td className="p-3 text-sand-100">{a.character_name || '—'}</td>
              <td className="p-3 text-sand-300/70">{a.discord_user_id}</td>
              <td className="p-3">{a.balance.toLocaleString()} {currency}</td>
              <td className="p-3 text-sand-300/50">{new Date(a.linked_at).toLocaleDateString()}</td>
            </tr>
          ))}
          {accounts.length === 0 && <tr><td colSpan={4} className="p-6 text-center text-sand-300/40">No linked players yet.</td></tr>}
        </tbody>
      </table>
    </Card>
  )
}
