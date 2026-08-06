export function StatusDot({ tone = 'success', label }: { tone?: 'success' | 'warning' | 'muted' | 'danger'; label: string }) {
  return <span className={`status status--${tone}`}><span className="status__dot" aria-hidden="true" />{label}</span>
}
