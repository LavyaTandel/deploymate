interface StatusBadgeProps {
  status: string
}

const statusConfig: Record<string, { color: string; label: string }> = {
  running: { color: 'bg-status-running', label: 'Running' },
  deploying: { color: 'bg-status-deploying', label: 'Deploying' },
  failed: { color: 'bg-status-failed', label: 'Failed' },
  warning: { color: 'bg-status-warning', label: 'Warning' },
  pending: { color: 'bg-status-pending', label: 'Pending' },
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.pending

  return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium bg-bg-tertiary">
      <span className={`w-1.5 h-1.5 rounded-full ${config.color}`} />
      {config.label}
    </span>
  )
}
