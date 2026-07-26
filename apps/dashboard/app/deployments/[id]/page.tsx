'use client'

import { useEffect, useState } from 'react'
import { StatusBadge } from '@/components/StatusBadge'

interface Event {
  type: string
  data: Record<string, unknown>
  timestamp: string
}

export default function DeploymentDetailPage({ params }: { params: { id: string } }) {
  const [events, setEvents] = useState<Event[]>([])
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    const es = new EventSource(`/api/v1/events?deployment_id=${params.id}`)

    es.addEventListener('connected', () => setConnected(true))
    es.addEventListener('heartbeat', () => {})
    es.addEventListener('deploy.progress', (e) => {
      const data = JSON.parse(e.data)
      setEvents((prev) => [...prev, { type: 'deploy.progress', data, timestamp: new Date().toISOString() }])
    })
    es.addEventListener('deployment.completed', (e) => {
      const data = JSON.parse(e.data)
      setEvents((prev) => [...prev, { type: 'deployment.completed', data, timestamp: new Date().toISOString() }])
    })

    return () => es.close()
  }, [params.id])

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-semibold">Deployment {params.id}</h1>
          <StatusBadge status="running" />
        </div>
        <div className="flex items-center gap-2">
          <button className="px-4 py-2 text-sm font-medium bg-status-failed/20 hover:bg-status-failed/30 text-status-failed rounded-md transition-colors">
            Rollback
          </button>
          <div className="flex items-center gap-1.5 text-xs text-text-tertiary">
            <div className={`w-2 h-2 rounded-full ${connected ? 'bg-status-running animate-pulse' : 'bg-status-pending'}`} />
            {connected ? 'Connected' : 'Connecting...'}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6">
        <div className="bg-bg-secondary border border-border rounded-lg p-4">
          <h2 className="text-sm font-medium text-text-tertiary mb-3">Details</h2>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-text-secondary">Service</dt>
              <dd className="text-text-primary">api-gateway</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-text-secondary">Environment</dt>
              <dd className="text-text-primary">production</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-text-secondary">Image</dt>
              <dd className="text-text-primary font-mono text-xs">gcr.io/proj/api@sha256:abc...</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-text-secondary">Replicas</dt>
              <dd className="text-text-primary">3/3</dd>
            </div>
          </dl>
        </div>

        <div className="bg-bg-secondary border border-border rounded-lg p-4">
          <h2 className="text-sm font-medium text-text-tertiary mb-3">Event Stream</h2>
          <div className="space-y-2 max-h-64 overflow-auto">
            {events.length === 0 ? (
              <p className="text-sm text-text-tertiary">Waiting for events...</p>
            ) : (
              events.map((event, i) => (
                <div key={i} className="flex items-center gap-3 text-sm">
                  <span className="text-text-tertiary font-mono text-xs">
                    {new Date(event.timestamp).toLocaleTimeString()}
                  </span>
                  <span className="text-text-primary">{event.type}</span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
