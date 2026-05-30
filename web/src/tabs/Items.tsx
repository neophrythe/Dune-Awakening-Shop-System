import { useEffect, useState } from 'react'
import { api, type Item } from '../api'
import { Button, Card, Input, Pill } from '../ui'

export default function Items({ currency }: { currency: string }) {
  const [items, setItems] = useState<Item[]>([])
  const [err, setErr] = useState('')
  const [form, setForm] = useState({ game_item_id: '', name: '', category: '', price: '', quantity: '1' })

  const load = () => api.items().then(setItems).catch((e) => setErr(String(e)))
  useEffect(() => { load() }, [])

  const add = async () => {
    setErr('')
    try {
      await api.upsertItem({
        game_item_id: form.game_item_id,
        name: form.name,
        category: form.category,
        price: Number(form.price) || 0,
        quantity: Number(form.quantity) || 1,
        enabled: true,
      })
      setForm({ game_item_id: '', name: '', category: '', price: '', quantity: '1' })
      load()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    }
  }

  const toggle = async (it: Item) => {
    await api.setItemEnabled(it.id, !it.enabled).catch((e) => setErr(String(e)))
    load()
  }

  return (
    <div className="space-y-6">
      <Card className="p-5">
        <h2 className="mb-3 font-display text-lg text-sand-200">Add item</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <Input value={form.name} onChange={(v) => setForm({ ...form, name: v })} placeholder="Name" />
          <Input value={form.game_item_id} onChange={(v) => setForm({ ...form, game_item_id: v })} placeholder="Game item id" />
          <Input value={form.category} onChange={(v) => setForm({ ...form, category: v })} placeholder="Category" />
          <Input value={form.price} onChange={(v) => setForm({ ...form, price: v })} placeholder="Price" type="number" />
          <Input value={form.quantity} onChange={(v) => setForm({ ...form, quantity: v })} placeholder="Qty" type="number" />
        </div>
        <div className="mt-3"><Button onClick={add}>Add item</Button></div>
        {err && <div className="mt-2 text-sm text-red-400">{err}</div>}
      </Card>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-night-950/60 text-left text-sand-300/70">
            <tr>
              <th className="p-3">#</th><th className="p-3">Name</th><th className="p-3">Category</th>
              <th className="p-3">Price</th><th className="p-3">Qty</th><th className="p-3">Status</th><th className="p-3"></th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id} className="border-t border-sand-900/50">
                <td className="p-3 text-sand-300/50">{it.id}</td>
                <td className="p-3 text-sand-100">{it.name}<div className="text-xs text-sand-300/40">{it.game_item_id}</div></td>
                <td className="p-3">{it.category || '—'}</td>
                <td className="p-3">{it.price} {currency}</td>
                <td className="p-3">{it.quantity}</td>
                <td className="p-3"><Pill ok={it.enabled}>{it.enabled ? 'enabled' : 'disabled'}</Pill></td>
                <td className="p-3 text-right">
                  <Button variant="ghost" onClick={() => toggle(it)}>{it.enabled ? 'Disable' : 'Enable'}</Button>
                </td>
              </tr>
            ))}
            {items.length === 0 && <tr><td colSpan={7} className="p-6 text-center text-sand-300/40">No items yet.</td></tr>}
          </tbody>
        </table>
      </Card>
    </div>
  )
}
