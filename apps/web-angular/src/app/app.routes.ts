import { Routes } from '@angular/router';
import { ADMIN_ROUTES } from './features/admin/admin.routes';
import { AUTH_ROUTES } from './features/auth/auth.routes';
import { MAINTENANCE_ROUTES } from './features/maintenance/maintenance.routes';
import { MONITOR_ROUTES } from './features/monitors/monitor.routes';
import { NOTIFICATION_ROUTES } from './features/notifications/notification.routes';
import { STATUS_PAGE_ROUTES } from './features/status-pages/status-pages.routes';

export const routes: Routes = [
  ...AUTH_ROUTES,
  ...MONITOR_ROUTES,
  ...STATUS_PAGE_ROUTES,
  ...MAINTENANCE_ROUTES,
  ...NOTIFICATION_ROUTES,
  ...ADMIN_ROUTES,
  { path: '', redirectTo: 'monitors', pathMatch: 'full' },
  { path: '**', redirectTo: 'monitors' },
];
