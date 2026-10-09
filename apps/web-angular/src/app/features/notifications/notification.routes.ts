import { Routes } from '@angular/router';
import { requireAuth } from '../../core/auth.guard';
import { NotificationChannelEditorComponent } from './notification-channel-editor.component';
import { NotificationChannelsComponent } from './notification-channels.component';

export const NOTIFICATION_ROUTES: Routes = [
  { path: 'notification-channels', component: NotificationChannelsComponent, canActivate: [requireAuth], pathMatch: 'full' },
  { path: 'notification-channels/new', component: NotificationChannelEditorComponent, canActivate: [requireAuth] },
  { path: 'notification-channels/:id/edit', component: NotificationChannelEditorComponent, canActivate: [requireAuth] },
];
