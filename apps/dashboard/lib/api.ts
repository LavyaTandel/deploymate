const ENGINE_URL = process.env.NEXT_PUBLIC_ENGINE_URL || 'http://localhost:8080'

export interface Deployment {
  id: string
  org_id: string
  project_id: string
  environment: string
  service: string
  image: string
  replicas: number
  resources: {
    cpu: string
    memory: string
  }
  policy_ref: string
  target_ref: string
  created_at: string
  updated_at: string
}

export interface Agent {
  id: string
  name: string
  target_type: string
  status: string
  last_heartbeat: string
}

export interface PolicyBundle {
  id: string
  version: string
  sha256: string
  created_at: string
}

export async function fetchDeployments(): Promise<Deployment[]> {
  const res = await fetch(`${ENGINE_URL}/api/v1/deployments`)
  if (!res.ok) throw new Error('Failed to fetch deployments')
  return res.json()
}

export async function fetchDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${ENGINE_URL}/api/v1/deployments/${id}/desired-state`)
  if (!res.ok) throw new Error('Failed to fetch deployment')
  return res.json()
}

export async function fetchAgents(): Promise<Agent[]> {
  const res = await fetch(`${ENGINE_URL}/api/v1/agents`)
  if (!res.ok) throw new Error('Failed to fetch agents')
  return res.json()
}

export async function rollbackDeployment(id: string): Promise<void> {
  const res = await fetch(`${ENGINE_URL}/api/v1/deployments/${id}/rollback`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error('Failed to rollback deployment')
}

export function connectSSE(
  deploymentId: string,
  onEvent: (event: string, data: unknown) => void
): EventSource {
  const es = new EventSource(
    `${ENGINE_URL}/api/v1/events?deployment_id=${deploymentId}`
  )

  es.onmessage = (e) => {
    onEvent('message', JSON.parse(e.data))
  }

  es.addEventListener('connected', (e) => {
    onEvent('connected', JSON.parse(e.data))
  })

  es.addEventListener('heartbeat', (e) => {
    onEvent('heartbeat', JSON.parse(e.data))
  })

  es.addEventListener('deploy.progress', (e) => {
    onEvent('deploy.progress', JSON.parse(e.data))
  })

  es.addEventListener('deployment.completed', (e) => {
    onEvent('deployment.completed', JSON.parse(e.data))
  })

  es.addEventListener('deployment.rolled_back', (e) => {
    onEvent('deployment.rolled_back', JSON.parse(e.data))
  })

  return es
}
