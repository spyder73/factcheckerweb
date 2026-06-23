// Route table. Pages will live in src/pages/<Name>.tsx. The lazy() imports
// keep the initial JS bundle small — the landing + check pages are the
// only paths most visitors load.

import { createBrowserRouter, Navigate, type RouteObject } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import { AppShell } from './components/AppShell'

const lazyImport = <T extends { default: React.ComponentType }>(loader: () => Promise<T>) => {
  const C = lazy(loader)
  return () => (
    <Suspense fallback={<div className="p-12 text-center text-fg-muted">Loading…</div>}>
      <C />
    </Suspense>
  )
}

const Landing = lazyImport(() => import('./pages/Landing'))
const Check = lazyImport(() => import('./pages/Check'))
const CheckDetail = lazyImport(() => import('./pages/CheckDetail'))
const History = lazyImport(() => import('./pages/History'))
const HowItWorks = lazyImport(() => import('./pages/HowItWorks'))
const Sources = lazyImport(() => import('./pages/Sources'))
const Pricing = lazyImport(() => import('./pages/Pricing'))
const Economics = lazyImport(() => import('./pages/Economics'))
const Login = lazyImport(() => import('./pages/Login'))
const Signup = lazyImport(() => import('./pages/Signup'))
const Forgot = lazyImport(() => import('./pages/Forgot'))
const Reset = lazyImport(() => import('./pages/Reset'))
const SettingsKeys = lazyImport(() => import('./pages/SettingsKeys'))
const SettingsJournalist = lazyImport(() => import('./pages/SettingsJournalist'))
const NotFound = lazyImport(() => import('./pages/NotFound'))

const routes: RouteObject[] = [
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <Landing /> },
      { path: 'check', element: <Check /> },
      { path: 'check/:id', element: <CheckDetail /> },
      { path: 'history', element: <History /> },
      { path: 'how-it-works', element: <HowItWorks /> },
      { path: 'sources', element: <Sources /> },
      { path: 'pricing', element: <Pricing /> },
      { path: 'economics', element: <Economics /> },
      { path: 'login', element: <Login /> },
      { path: 'signup', element: <Signup /> },
      { path: 'forgot', element: <Forgot /> },
      { path: 'reset', element: <Reset /> },
      { path: 'settings', element: <Navigate to="/settings/keys" replace /> },
      { path: 'settings/keys', element: <SettingsKeys /> },
      { path: 'settings/journalist', element: <SettingsJournalist /> },
      { path: '*', element: <NotFound /> },
    ],
  },
]

export const router = createBrowserRouter(routes)
