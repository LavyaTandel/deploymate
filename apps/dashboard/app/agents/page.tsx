import { StatusBadge } from '@/components/StatusBadge'

const agents = [
  { id: '1', name: 'prod-cluster-1', type: 'kubernetes', status: 'online', lastHeartbeat: '5s ago', deployments: 12 },
  { id: '2', name: 'staging-cluster', type: 'kubernetes', status: 'online', lastHeartbeat: '3s ago', deployments: 8 },
  { id: '3', name: 'cloudrun-us-central', type: 'cloudrun', status: 'online', lastHeartbeat: '10s ago', deployments: 5 },
  { id: '4', name: 'legacy-vm-01', type: 'vm', status: 'offline', lastHeartbeat: '2h ago', deployments: 1 },
]

export default function AgentsPage() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Agents</h1>
        <div className="flex items-center gap-2 text-sm text-text-secondary">
          <span>{agents.filter((a) => a.status === 'online').length} online</span>
          <span>·</span>
          <span>{agents.length} total</span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {agents.map((agent) => (
          <div key={agent.id} className="bg-bg-secondary border border-border rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <h3 className="font-medium">{agent.name}</h3>
                <StatusBadge status={agent.status === 'online' ? 'running' : 'failed'} />
              </div>
              <span className="text-xs text-text-tertiary px-2 py-1 bg-bg-tertiary rounded">{agent.type}</span>
            </div>
            <div className="text-sm text-text-secondary space-y-1">
              <p>Last heartbeat: {agent.lastHeartbeat}</p>
              <p>Active deployments: {agent.deployments}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
