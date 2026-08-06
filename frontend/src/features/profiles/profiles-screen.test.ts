import { describe, expect, it } from 'vitest'
import { sortTweakItems } from './profiles-screen'
import type { PlanItem } from '../../shared/contracts/domain'

const item = (id: string, name: string, risk: PlanItem['risk_level'], changed = false, restore = false): PlanItem => ({
  id, name, risk_level: risk, changed, restore_available: restore,
  category: 'Test', current: '0', desired: '1', effect: '', benefit: '', risk: '', reversible: true, manual_available: true,
})

describe('tweak sorting', () => {
  const items = [item('low', 'Zulu', 'low'), item('high', 'Alpha', 'high'), item('changed', 'Beta', 'medium', true), item('restore', 'Gamma', 'medium', false, true)]

  it('sorts by name, risk, and actionable state without mutating the catalog', () => {
    expect(sortTweakItems(items, 'name', 'en').map(({ id }) => id)).toEqual(['high', 'changed', 'restore', 'low'])
    expect(sortTweakItems(items, 'risk', 'en')[0]?.id).toBe('high')
    expect(sortTweakItems(items, 'status', 'en').map(({ id }) => id).slice(0, 2)).toEqual(['changed', 'restore'])
    expect(items.map(({ id }) => id)).toEqual(['low', 'high', 'changed', 'restore'])
  })
})
