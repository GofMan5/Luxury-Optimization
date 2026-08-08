export function dialogFocusTarget(activeIndex: number, lastIndex: number, backwards: boolean): number | null {
  if (backwards && activeIndex === 0) return lastIndex
  if (!backwards && activeIndex === lastIndex) return 0
  return null
}
