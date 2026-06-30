import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { AppShell } from '@/components/AppShell'
import { ProtectedRoute } from '@/features/auth/ProtectedRoute'
import { LoginPage } from '@/features/auth/LoginPage'
import { AcceptInvitePage } from '@/features/auth/AcceptInvitePage'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { BrandsPage } from '@/features/brands/BrandsPage'
import { ListsPage } from '@/features/lists/ListsPage'
import { ListDetailPage } from '@/features/subscribers/ListDetailPage'
import { CampaignsPage } from '@/features/campaigns/CampaignsPage'
import { ComposePage } from '@/features/campaigns/ComposePage'
import { ReportPage } from '@/features/campaigns/ReportPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { AnalyticsPage } from '@/features/analytics/AnalyticsPage'
import { SegmentsPage } from '@/features/segments/SegmentsPage'
import { SignupFormsPage } from '@/features/signup-forms/SignupFormsPage'
import { ABTestsPage } from '@/features/ab-tests/ABTestsPage'
import { AutomationsPage } from '@/features/automations/AutomationsPage'
import { TemplatesPage } from '@/features/templates/TemplatesPage'
import { DeliverabilityPage } from '@/features/deliverability/DeliverabilityPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/accept/:token', element: <AcceptInvitePage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        path: '/',
        element: <AppShell />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'settings', element: <SettingsPage /> },
          { path: 'brands', element: <BrandsPage /> },
          { path: 'lists', element: <ListsPage /> },
          { path: 'lists/:id', element: <ListDetailPage /> },
          { path: 'campaigns', element: <CampaignsPage /> },
          { path: 'campaigns/:id', element: <ComposePage /> },
          { path: 'campaigns/:id/report', element: <ReportPage /> },
          { path: 'analytics', element: <AnalyticsPage /> },
          { path: 'segments', element: <SegmentsPage /> },
          { path: 'signup-forms', element: <SignupFormsPage /> },
          { path: 'automations', element: <AutomationsPage /> },
          { path: 'ab-tests', element: <ABTestsPage /> },
          { path: 'templates', element: <TemplatesPage /> },
          { path: 'deliverability', element: <DeliverabilityPage /> },
        ],
      },
    ],
  },
])

export function App() {
  return <RouterProvider router={router} />
}
