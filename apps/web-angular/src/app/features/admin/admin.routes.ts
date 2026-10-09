import { Routes } from '@angular/router';
import { requireAuth } from '../../core/auth.guard';
import { GroupsPage } from './groups.page';
import { IncidentsPage } from './incidents.page';
import { MonitorGroupsPage } from './monitor-groups.page';
import { ProxiesPage } from './proxies.page';
import { SecurityPage } from './security.page';
import { SettingsPage } from './settings.page';
import { TagsPage } from './tags.page';

export const ADMIN_ROUTES: Routes = [
  { path: 'groups', component: GroupsPage, canActivate: [requireAuth] },
  { path: 'monitor-groups', component: MonitorGroupsPage, canActivate: [requireAuth] },
  { path: 'incidents', component: IncidentsPage, canActivate: [requireAuth] },
  { path: 'proxies/new', component: ProxiesPage, canActivate: [requireAuth] },
  { path: 'proxies/:id/edit', component: ProxiesPage, canActivate: [requireAuth] },
  { path: 'proxies', component: ProxiesPage, canActivate: [requireAuth], pathMatch: 'full' },
  { path: 'tags/new', component: TagsPage, canActivate: [requireAuth] },
  { path: 'tags/:id/edit', component: TagsPage, canActivate: [requireAuth] },
  { path: 'tags', component: TagsPage, canActivate: [requireAuth], pathMatch: 'full' },
  { path: 'security', component: SecurityPage, canActivate: [requireAuth] },
  { path: 'settings', component: SettingsPage, canActivate: [requireAuth] },
];
