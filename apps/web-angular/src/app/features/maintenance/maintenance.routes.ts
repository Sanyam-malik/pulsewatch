import { Routes } from '@angular/router';
import { requireAuth } from '../../core/auth.guard';
import { MaintenanceEditorComponent } from './maintenance-editor.component';
import { MaintenanceListComponent } from './maintenance-list.component';

export const MAINTENANCE_ROUTES: Routes = [
  { path: 'maintenances', component: MaintenanceListComponent, canActivate: [requireAuth] },
  { path: 'maintenances/new', component: MaintenanceEditorComponent, canActivate: [requireAuth] },
  { path: 'maintenances/:id/edit', component: MaintenanceEditorComponent, canActivate: [requireAuth] },
];
