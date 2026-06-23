// Root layout: TopNav + Outlet + Footer. Scroll-to-top on route change.
import { Outlet, useLocation } from 'react-router-dom'
import { useEffect } from 'react'
import { TopNav } from './TopNav'
import { Footer } from './Footer'

export function AppShell() {
  const location = useLocation()
  useEffect(() => {
    window.scrollTo(0, 0)
  }, [location.pathname])

  // Hide the economics ticker on live /check (busy enough already) and /settings/*.
  const hideTicker =
    location.pathname === '/check' ||
    location.pathname.startsWith('/check/') ||
    location.pathname.startsWith('/settings/')

  return (
    <div className="min-h-screen flex flex-col">
      <TopNav />
      <main id="main" className="flex-1">
        <Outlet />
      </main>
      <Footer showTicker={!hideTicker} />
    </div>
  )
}
