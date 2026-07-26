# DeployMate Dashboard — Design Taste

## Product Context

DeployMate is a **developer/DevOps tool** — a GitOps deployment engine dashboard. Users are platform engineers, SREs, and developers who:
- Monitor deployments in real-time
- Trigger rollbacks under pressure
- Scan cost breakdowns
- Configure policies

The design must be **functional first**, beautiful second. Speed of comprehension > aesthetics.

---

## Design Direction: "Terminal Clarity"

A clean, dark-first interface inspired by terminal aesthetics and modern DevOps dashboards (Vercel, Linear, Raycast). High information density with clear visual hierarchy. No decorative elements — every pixel serves a purpose.

### Core Principles

1. **Dark-first** — Dark background as default (devs prefer it). Light mode as option.
2. **Data-dense, not cluttered** — Maximize information per screen without chaos.
3. **Real-time feel** — Subtle animations for status changes, SSE updates feel alive.
4. **Error-forward** — Failures are visually prominent. Success is quiet.
5. **Keyboard-first** — Every action reachable via keyboard. Command palette (Cmd+K).

---

## Color System

### Dark Mode (Primary)

| Token | Value | Usage |
|-------|-------|-------|
| `--bg-primary` | `#0a0a0b` | Page background |
| `--bg-secondary` | `#141416` | Card/panel backgrounds |
| `--bg-tertiary` | `#1c1c1f` | Hover states, subtle elevation |
| `--border` | `#27272a` | Dividers, card borders |
| `--text-primary` | `#fafafa` | Headings, primary text |
| `--text-secondary` | `#a1a1aa` | Descriptions, labels |
| `--text-tertiary` | `#71717a` | Timestamps, metadata |

### Status Colors

| Status | Color | Hex | Usage |
|--------|-------|-----|-------|
| Running | Green | `#22c55e` | Healthy deployments |
| Deploying | Blue | `#3b82f6` | In-progress operations |
| Failed | Red | `#ef4444` | Errors, failures |
| Warning | Amber | `#f59e0b` | Degraded state, rollback limits |
| Pending | Gray | `#71717a` | Queued, waiting |

### Accent

| Token | Value | Usage |
|-------|-------|-------|
| `--accent` | `#6366f1` | Primary actions, links, focus rings |
| `--accent-hover` | `#818cf8` | Hover states on accent |

---

## Typography

| Element | Font | Size | Weight | Line Height |
|---------|------|------|--------|-------------|
| Page title | Inter | 24px | 600 | 1.2 |
| Section header | Inter | 16px | 600 | 1.4 |
| Body text | Inter | 14px | 400 | 1.5 |
| Code/mono | JetBrains Mono | 13px | 400 | 1.5 |
| Label | Inter | 12px | 500 | 1.4 |
| Timestamp | Inter | 11px | 400 | 1.4 |

**Why Inter:** Excellent readability at small sizes, designed for screens. Standard in dev tools.

**Why JetBrains Mono:** Distinct `0O`, `1lI` differentiation. Ligatures for code. Developer affinity.

---

## Layout

### Dashboard Grid

```
┌─────────────────────────────────────────────────────┐
│  Sidebar (240px)  │  Main Content Area              │
│  ───────────────  │  ─────────────────────────────  │
│  Logo             │  Header: Page title + actions    │
│  Nav items        │  ─────────────────────────────  │
│  ───────────────  │  Content cards / data tables    │
│  Quick actions    │                                  │
│  ───────────────  │                                  │
│  Org selector     │                                  │
└───────────────────┴─────────────────────────────────┘
```

### Card System

- **Border radius:** 8px (consistent, modern)
- **Padding:** 16px internal, 12px for compact cards
- **Shadow:** None in dark mode (borders provide separation)
- **Border:** 1px solid `var(--border)`

### Spacing Scale

4px base unit: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64

---

## Components

### Status Badge

```jsx
// Pill shape, 20px height, dot + text
<div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium">
  <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
  Running
</div>
```

### Deployment Card

```
┌──────────────────────────────────────────┐
│  ● api-service          production  v42  │
│  ────────────────────────────────────── │
│  Image: gcr.io/proj/api@sha256:abc...   │
│  Replicas: 3/3  CPU: 500m  Mem: 512Mi  │
│  ────────────────────────────────────── │
│  Last deploy: 2m ago   Cost: $12.40/mo  │
│                          [Rollback] [Log]│
└──────────────────────────────────────────┘
```

### SSE Event Stream

```
┌──────────────────────────────────────────┐
│  Events                          [Clear] │
│  ────────────────────────────────────── │
│  02:10:15  ● deployment.started          │
│  02:10:18  ● policy.evaluated  ✓ passed  │
│  02:10:22  ● build.completed  45s        │
│  02:10:25  ● deploy.progress  60%        │
│  02:10:30  ● health.check     ✓ 3/3      │
│  02:10:32  ● deployment.completed ✓      │
└──────────────────────────────────────────┘
```

### Data Table

- Row height: 44px
- Sticky header
- Subtle hover: `var(--bg-tertiary)`
- Sortable columns: arrow indicator
- Zebra striping: NO (cleaner without)

---

## Interactions

### Transitions

| Property | Duration | Easing |
|----------|----------|--------|
| Background color | 150ms | ease-in-out |
| Border color | 150ms | ease-in-out |
| Transform (scale) | 200ms | ease-out |
| Opacity | 150ms | ease-in-out |

### Micro-interactions

- **Status change:** Brief pulse animation (scale 1 → 1.02 → 1)
- **New event:** Slide in from right, fade in
- **Rollback button:** Red glow on hover, confirmation modal
- **SSE connected:** Subtle green dot pulsing in header

### Command Palette (Cmd+K)

- Modal overlay with backdrop blur
- Search input at top
- Results grouped by category
- Keyboard navigation (↑↓ arrows, Enter to select)

---

## Responsive Breakpoints

| Breakpoint | Width | Layout |
|------------|-------|--------|
| Mobile | < 768px | Single column, sidebar collapses |
| Tablet | 768-1024px | Two columns, sidebar collapsible |
| Desktop | > 1024px | Full layout with sidebar |

---

## Icons

Use **Lucide React** — consistent 1.5px stroke, clean geometric style.

Key icons:
- Deployment: `Rocket`, `ArrowUpCircle`
- Status: `CheckCircle`, `XCircle`, `AlertTriangle`
- Navigation: `LayoutDashboard`, `GitBranch`, `Shield`
- Actions: `RotateCcw` (rollback), `Trash2` (destroy), `RefreshCw`

---

## Anti-patterns to Avoid

1. **No gradients** — Flat colors only. Gradients feel dated.
2. **No drop shadows in dark mode** — Use borders instead.
3. **No center-aligned text** — Left-aligned for scannability.
4. **No placeholder text in tables** — Empty state with illustration.
5. **No colored backgrounds for cards** — Use borders and subtle bg shifts.

---

## Reference Implementations

- **Vercel Dashboard** — Clean data density, real-time deployment view
- **Linear** — Keyboard-first, command palette, smooth transitions
- **Raycast** — Dark mode execution, minimal chrome
- **Grafana** — Data visualization, alert states, time-series

---

## Tech Stack (Dashboard)

- **Framework:** Next.js 14 (App Router)
- **Styling:** Tailwind CSS v4
- **Components:** shadcn/ui (Radix primitives)
- **Icons:** Lucide React
- **State:** React hooks + SSE EventSource
- **Fonts:** Inter (UI) + JetBrains Mono (code)

---

*This document defines the visual taste for DeployMate's dashboard. Apply consistently across all frontend components.*
