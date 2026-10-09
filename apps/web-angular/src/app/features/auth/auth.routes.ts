import { Routes } from '@angular/router';
import { guestOnly } from '../../core/auth.guard';
import { LoginComponent } from './login.component';
import { RegisterComponent } from './register.component';

export const AUTH_ROUTES: Routes = [
  { path: 'login', component: LoginComponent, canActivate: [guestOnly] },
  { path: 'register', component: RegisterComponent, canActivate: [guestOnly] },
];
