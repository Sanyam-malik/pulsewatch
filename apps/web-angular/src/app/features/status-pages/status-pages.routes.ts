import { Routes } from '@angular/router';
import { requireAuth } from '../../core/auth.guard';
import { PublicStatusPageComponent } from './public-status-page.component';
import { StatusPageEditorComponent } from './status-page-editor.component';
import { StatusPagesListComponent } from './status-pages-list.component';

export const STATUS_PAGE_ROUTES: Routes = [
  { path: 'status-pages', component: StatusPagesListComponent, canActivate: [requireAuth] },
  { path: 'status-pages/new', component: StatusPageEditorComponent, canActivate: [requireAuth] },
  { path: 'status-pages/:id/edit', component: StatusPageEditorComponent, canActivate: [requireAuth] },
  { path: 'status/:slug', component: PublicStatusPageComponent },
];
