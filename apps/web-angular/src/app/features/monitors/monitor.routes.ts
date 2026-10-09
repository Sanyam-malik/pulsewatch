import { Routes } from '@angular/router';
import { requireAuth } from '../../core/auth.guard';
import { MonitorFormComponent } from './monitor-form.component';
import { MonitorListComponent } from './monitor-list.component';
import { MonitorViewComponent } from './monitor-view.component';

export const MONITOR_ROUTES: Routes = [
  {
    path: 'monitors',
    canActivate: [requireAuth],
    children: [
      { path: '', component: MonitorListComponent, pathMatch: 'full' },
      { path: 'new', component: MonitorFormComponent, data: { mode: 'create' } },
      { path: ':id/edit', component: MonitorFormComponent, data: { mode: 'edit' } },
      { path: ':id', component: MonitorViewComponent },
    ],
  },
];
