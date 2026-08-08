import { lazy, Suspense, startTransition, useState } from 'react'
import type { RouteID } from './routes'
import { AppShell } from '../widgets/app-shell/app-shell'
import { LoadingState } from '../shared/ui/feedback'

const OverviewScreen = lazy(() => import('../features/overview/overview-screen'))
const ProfilesScreen = lazy(() => import('../features/profiles/profiles-screen'))
const GamesScreen = lazy(() => import('../features/games/games-screen'))
const MeasurementsScreen = lazy(() => import('../features/measurements/measurements-screen'))
const SystemScreen = lazy(() => import('../features/system/system-screen'))
const RestoreScreen = lazy(() => import('../features/restore/restore-screen'))

export default function App() {
  const [route, setRoute] = useState<RouteID>('overview')
  const navigate = (next: RouteID) => startTransition(() => setRoute(next))
  let screen
  switch (route) {
    case 'overview': screen = <OverviewScreen onNavigate={navigate} />; break
    case 'profiles': screen = <ProfilesScreen />; break
    case 'games': screen = <GamesScreen onNavigate={navigate} />; break
    case 'benchmarks': screen = <MeasurementsScreen />; break
    case 'system': screen = <SystemScreen />; break
    case 'restore': screen = <RestoreScreen />; break
  }
  return <AppShell route={route} onNavigate={navigate}><Suspense fallback={<LoadingState />}>{screen}</Suspense></AppShell>
}
