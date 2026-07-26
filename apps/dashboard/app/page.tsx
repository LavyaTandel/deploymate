import { DeploymentCard } from '@/components/DeploymentCard'

const mockDeployments = [
  {
    id: '1',
    service: 'api-gateway',
    environment: 'production',
    image: 'gcr.io/proj/api@sha256:abc123',
    status: 'running',
    replicas: '3/3',
    lastDeploy: '2m ago',
  },
  {
    id: '2',
    service: 'auth-service',
    environment: 'production',
    image: 'gcr.io/proj/auth@sha256:def456',
    status: 'deploying',
    replicas: '2/3',
    lastDeploy: '30s ago',
  },
  {
    id: '3',
    service: 'payment-worker',
    environment: 'staging',
    image: 'gcr.io/proj/payment@sha256:ghi789',
    status: 'failed',
    replicas: '0/2',
    lastDeploy: '5m ago',
  },
]

export default function DashboardPage() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <div className="flex items-center gap-2">
          <span className="text-xs text-text-tertiary">Last updated: just now</span>
          <div className="w-2 h-2 rounded-full bg-status-running animate-pulse" />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {mockDeployments.map((dep) => (
          <DeploymentCard key={dep.id} {...dep} />
        ))}
      </div>

      <div className="mt-8">
        <h2 className="text-lg font-medium mb-4">Recent Events</h2>
        <div className="bg-bg-secondary border border-border rounded-lg p-4 space-y-2">
          {[
            { time: '02:10:15', event: 'deployment.started', service: 'api-gateway' },
            { time: '02:10:18', event: 'policy.evaluated', service: 'api-gateway', detail: '✓ passed' },
            { time: '02:10:25', event: 'deploy.progress', service: 'auth-service', detail: '60%' },
            { time: '02:10:30', event: 'health.check', service: 'api-gateway', detail: '✓ 3/3' },
          ].map((e, i) => (
            <div key={i} className="flex items-center gap-4 text-sm">
              <span className="text-text-tertiary font-mono text-xs">{e.time}</span>
              <span className="text-text-secondary">{e.service}</span>
              <span className="text-text-primary">{e.event}</span>
              {e.detail && <span className="text-text-tertiary">{e.detail}</span>}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
