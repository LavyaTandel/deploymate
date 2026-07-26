'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Rocket, Shield, Globe, Settings } from 'lucide-react'

const navItems = [
  { href: '/', label: 'Overview', icon: Rocket },
  { href: '/deployments', label: 'Deployments', icon: Rocket },
  { href: '/agents', label: 'Agents', icon: Globe },
  { href: '/policies', label: 'Policies', icon: Shield },
  { href: '/settings', label: 'Settings', icon: Settings },
]

export function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="w-60 bg-bg-secondary border-r border-border flex flex-col">
      <div className="p-4 border-b border-border">
        <h1 className="text-lg font-semibold flex items-center gap-2">
          <Rocket className="w-5 h-5 text-accent" />
          DeployMate
        </h1>
      </div>
      <nav className="flex-1 p-2">
        {navItems.map((item) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                isActive
                  ? 'bg-bg-tertiary text-text-primary'
                  : 'text-text-secondary hover:text-text-primary hover:bg-bg-tertiary'
              }`}
            >
              <item.icon className="w-4 h-4" />
              {item.label}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
