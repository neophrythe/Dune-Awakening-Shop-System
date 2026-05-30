import { useEffect, useState } from 'react'
import { api, type Kit } from '../api'
import { Button, Card, Input, Pill } from '../ui'

export default function Kits({ currency }: { currency: string }) {
  const [kits, setKits] = useState<Kit[]>([])
  const [err, setErr] = useState('')
  const [form, setForm] = useState({ name: '', category: '', price: '', description: '' })
  const [itemForm, setItemForm] = useState<Record<number, { game_item_id: string; quantity: string }>>({})

  const load = () => api.kits().then(setKits).catch((e) => setErr(String(e)))
  useEffect(() => { load() }, [])

  const create = async () => {
    setErr('')
    try {
      await api.createKit({
        name: form.name, category: form.category,
        price: Number(form.price) || 0, description: form.description, enabled: true,
      })
      setForm({ name: '', category: '', price: '', description: '' })
      load()
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    }
  }

  const addItem = async (kitId: number) => {
    const f = itemForm[kitId]
    if (!f?.game_item_id) return
    await api.addKitItem(kitId, { game_item_id: f.game_item_id, quantity: Number(f.quantity) || 1 })
      .catch((e) => setErr(String(e)))
    setItemForm({ ...itemForm, [kitId]: { game_item_id: '', quantity: '1' } })
    load()
  }

  return (
    <div className="space-y-6">
      <Card className="p-5">
        <h2 className="mb-3 font-display text-lg text-sand-200">Create kit</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Input value={form.name} onChange={(v) => setForm({ ...form, name: v })} placeholder="Kit name" />
          <Input value={form.category} onChange={(v) => setForm({ ...form, category: v })} placeholder="Category" />
          <Input value={form.price} onChange={(v) => setForm({ ...form, price: v })} placeholder="Price" type="number" />
          <Input value={form.description} onChange={(v) => setForm({ ...form, description: v })} placeholder="Description" />
        </div>
        <div className="mt-3"><Button onClick={create}>Create kit</Button></div>
        {err && <div className="mt-2 text-sm text-red-400">{err}</div>}
      </Card>

      <div className="grid gap-4 md:grid-cols-2">
        {kits.map((k) => (
          <Card key={k.id} className="p-5">
            <div className="flex items-center justify-between">
              <div>
                <div className="font-display text-lg text-sand-100">{k.name} <span className="text-sand-300/40">#{k.id}</span></div>
                <div className="text-sm text-sand-300/60">{k.price} {currency} · {k.category || 'Packs'}</div>
              </div>
              <Pill ok={k.enabled}>{k.enabled ? 'enabled' : 'disabled'}</Pill>
            </div>
            {k.description && <p className="mt-2 text-sm text-sand-300/70">{k.description}</p>}
            <ul className="mt-3 space-y-1 text-sm">
              {(k.items || []).map((it) => (
                <li key={it.id} className="text-sand-200">• {it.quantity}× {it.name || it.game_item_id}</li>
              ))}
              {(!k.items || k.items.length === 0) && <li className="text-sand-300/40">No items yet</li>}
            </ul>
            <div className="mt-3 flex gap-2">
              <Input
                value={itemForm[k.id]?.game_item_id ?? ''}
                onChange={(v) => setItemForm({ ...itemForm, [k.id]: { ...itemForm[k.id], game_item_id: v, quantity: itemForm[k.id]?.quantity ?? '1' } })}
                placeholder="Game item id"
              />
              <div className="w-20">
                <Input
                  value={itemForm[k.id]?.quantity ?? '1'}
                  onChange={(v) => setItemForm({ ...itemForm, [k.id]: { ...itemForm[k.id], game_item_id: itemForm[k.id]?.game_item_id ?? '', quantity: v } })}
                  placeholder="Qty" type="number"
                />
              </div>
              <Button variant="ghost" onClick={() => addItem(k.id)}>Add</Button>
            </div>
          </Card>
        ))}
        {kits.length === 0 && <div className="text-sand-300/40">No kits yet.</div>}
      </div>
    </div>
  )
}
