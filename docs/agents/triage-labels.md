# Triage Label Mapping

## Canonical Roles → Label Strings

| Canonical Role | Label String | Description |
|----------------|--------------|-------------|
| needs-triage | `needs-triage` | Maintainer needs to evaluate |
| needs-info | `needs-info` | Waiting on reporter for more information |
| ready-for-agent | `ready-for-agent` | Fully specified, ready for an AFK agent |
| ready-for-human | `ready-for-human` | Needs human implementation |
| wontfix | `wontfix` | Will not be actioned |

## Category Labels (also applied)

| Category | Label String |
|----------|--------------|
| bug | `bug` |
| enhancement | `enhancement` |

## Usage

Every triaged issue must carry **exactly one** state label (from the five above) and **exactly one** category label (`bug` or `enhancement`).

State transitions:
```
unlabeled → needs-triage → needs-info → needs-triage → ready-for-agent | ready-for-human | wontfix
```

Maintainer may override at any time — flag unusual transitions before proceeding.