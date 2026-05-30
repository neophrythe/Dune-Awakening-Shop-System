// Small shared UI primitives, styled for the Dune sand/night theme.
import React from 'react'

export function Card({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`rounded-xl border border-sand-800/60 bg-night-900/70 shadow-lg ${className}`}>
      {children}
    </div>
  )
}

export function Stat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <Card className="p-5">
      <div className="text-sm uppercase tracking-wide text-sand-300/70">{label}</div>
      <div className="mt-1 text-3xl font-semibold text-sand-100">{value}</div>
    </Card>
  )
}

export function Button({
  children, onClick, type = 'button', variant = 'primary', disabled,
}: {
  children: React.ReactNode
  onClick?: () => void
  type?: 'button' | 'submit'
  variant?: 'primary' | 'ghost' | 'danger'
  disabled?: boolean
}) {
  const base = 'rounded-lg px-4 py-2 text-sm font-medium transition disabled:opacity-50'
  const styles = {
    primary: 'bg-sand-400 text-night-950 hover:bg-sand-300',
    ghost: 'border border-sand-700 text-sand-200 hover:bg-sand-900/40',
    danger: 'bg-red-700/80 text-white hover:bg-red-600',
  }[variant]
  return (
    <button type={type} onClick={onClick} disabled={disabled} className={`${base} ${styles}`}>
      {children}
    </button>
  )
}

export function Input({
  value, onChange, placeholder, type = 'text',
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  type?: string
}) {
  return (
    <input
      type={type}
      value={value}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-lg border border-sand-800/70 bg-night-950/60 px-3 py-2 text-sand-100 placeholder-sand-300/40 outline-none focus:border-sand-400"
    />
  )
}

export function Pill({ ok, children }: { ok: boolean; children: React.ReactNode }) {
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs ${ok ? 'bg-emerald-900/60 text-emerald-300' : 'bg-night-800 text-sand-300/60'}`}>
      {children}
    </span>
  )
}
