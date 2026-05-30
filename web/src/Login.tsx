import { useState } from 'react'
import { api } from './api'
import { Button, Card, Input } from './ui'

export default function Login({ onLogin }: { onLogin: (currency: string) => void }) {
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setErr('')
    try {
      const r = await api.login(user, password)
      onLogin(r.currency)
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm p-8">
        <h1 className="font-display text-2xl text-sand-200">🏜️ Dune Shop</h1>
        <p className="mt-1 text-sm text-sand-300/60">Admin dashboard</p>
        <form onSubmit={submit} className="mt-6 space-y-3">
          <Input value={user} onChange={setUser} placeholder="Username" />
          <Input value={password} onChange={setPassword} placeholder="Password" type="password" />
          {err && <div className="text-sm text-red-400">{err}</div>}
          <Button type="submit" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</Button>
        </form>
      </Card>
    </div>
  )
}
