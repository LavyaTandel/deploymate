import { StatusBadge } from './StatusBadge'

interface DeploymentCardProps {
  id: string
  service: string
  environment: string
  image: string
  status: string
  replicas: string
  lastDeploy: string
}

export function DeploymentCard({
  id,
  service,
  environment,
  image,
  status,
  replicas,
  lastDeploy,
}: DeploymentCardProps) {
  return (
    <div className="bg-bg-secondary border border-border rounded-lg p-4 hover:bg-bg-tertiary transition-colors">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <h3 className="font-medium">{service}</h3>
          <StatusBadge status={status} />
        </div>
        <span className="text-xs text-text-tertiary">{environment}</span>
      </div>
      <div className="text-sm text-text-secondary space-y-1">
        <p className="truncate">Image: {image}</p>
        <p>Replicas: {replicas}</p>
        <p>Last deploy: {lastDeploy}</p>
      </div>
      <div className="mt-4 flex gap-2">
        <button className="px-3 py-1.5 text-xs font-medium bg-bg-tertiary hover:bg-accent text-text-primary rounded-md transition-colors">
          Rollback
        </button>
        <button className="px-3 py-1.5 text-xs font-medium bg-bg-tertiary hover:bg-bg-secondary text-text-secondary rounded-md transition-colors">
          Logs
        </button>
      </div>
    </div>
  )
}
