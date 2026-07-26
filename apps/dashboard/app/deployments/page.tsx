import { StatusBadge } from '@/components/StatusBadge'

const deployments = [
  { id: '1', service: 'api-gateway', environment: 'production', image: 'gcr.io/proj/api@sha256:abc...', status: 'running', replicas: '3/3', updated: '2m ago' },
  { id: '2', service: 'auth-service', environment: 'production', image: 'gcr.io/proj/auth@sha256:def...', status: 'deploying', replicas: '2/3', updated: '30s ago' },
  { id: '3', service: 'payment-worker', environment: 'staging', image: 'gcr.io/proj/payment@sha256:ghi...', status: 'failed', replicas: '0/2', updated: '5m ago' },
  { id: '4', service: 'notification-svc', environment: 'production', image: 'gcr.io/proj/notif@sha256:jkl...', status: 'running', replicas: '2/2', updated: '1h ago' },
]

export default function DeploymentsPage() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Deployments</h1>
        <button className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-md transition-colors">
          New Deployment
        </button>
      </div>

      <div className="bg-bg-secondary border border-border rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-border">
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Service</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Environment</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Image</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Status</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Replicas</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Updated</th>
            </tr>
          </thead>
          <tbody>
            {deployments.map((dep) => (
              <tr key={dep.id} className="border-b border-border hover:bg-bg-tertiary transition-colors">
                <td className="px-4 py-3 text-sm font-medium">{dep.service}</td>
                <td className="px-4 py-3 text-sm text-text-secondary">{dep.environment}</td>
                <td className="px-4 py-3 text-sm text-text-secondary font-mono truncate max-w-[200px]">{dep.image}</td>
                <td className="px-4 py-3"><StatusBadge status={dep.status} /></td>
                <td className="px-4 py-3 text-sm text-text-secondary">{dep.replicas}</td>
                <td className="px-4 py-3 text-sm text-text-tertiary">{dep.updated}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
