import { StatusBadge } from '@/components/StatusBadge'

const policies = [
  { id: '1', name: 'cpu-limit', version: 'v1.2.0', status: 'active', evaluations: 1247, violations: 23, lastUpdated: '1h ago' },
  { id: '2', name: 'prod-approval', version: 'v1.1.0', status: 'active', evaluations: 892, violations: 5, lastUpdated: '3h ago' },
  { id: '3', name: 'image-provenance', version: 'v1.0.0', status: 'active', evaluations: 2100, violations: 0, lastUpdated: '1d ago' },
]

export default function PoliciesPage() {
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Policies</h1>
        <button className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-md transition-colors">
          Upload Bundle
        </button>
      </div>

      <div className="bg-bg-secondary border border-border rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-border">
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Name</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Version</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Status</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Evaluations</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Violations</th>
              <th className="text-left text-xs font-medium text-text-tertiary uppercase tracking-wider px-4 py-3">Updated</th>
            </tr>
          </thead>
          <tbody>
            {policies.map((policy) => (
              <tr key={policy.id} className="border-b border-border hover:bg-bg-tertiary transition-colors">
                <td className="px-4 py-3 text-sm font-medium">{policy.name}</td>
                <td className="px-4 py-3 text-sm text-text-secondary font-mono">{policy.version}</td>
                <td className="px-4 py-3"><StatusBadge status="running" /></td>
                <td className="px-4 py-3 text-sm text-text-secondary">{policy.evaluations.toLocaleString()}</td>
                <td className="px-4 py-3 text-sm text-text-secondary">{policy.violations}</td>
                <td className="px-4 py-3 text-sm text-text-tertiary">{policy.lastUpdated}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
