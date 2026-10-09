import { HttpErrorResponse } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { authErrorMessage } from './auth-error';
import { AuthApiEnvelope, AuthResponse, LoginCredentials } from './auth.types';
import { TwoFactorComponent } from './two-factor.component';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink, TwoFactorComponent],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent {
  private readonly formBuilder = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly form = this.formBuilder.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', Validators.required],
  });

  loading = false;
  showTwoFactor = false;
  serverError: string | null = null;
  private pendingCredentials: LoginCredentials | null = null;

  get routeQueryParams(): Record<string, string> {
    const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
    return returnUrl ? { returnUrl } : {};
  }

  submit(): void {
    if (this.form.invalid || this.loading) {
      this.form.markAllAsTouched();
      return;
    }
    this.pendingCredentials = this.form.getRawValue();
    this.serverError = null;
    this.sendLogin(this.pendingCredentials);
  }

  verifyTwoFactor(token: string): void {
    if (!this.pendingCredentials || this.loading) {
      return;
    }
    this.serverError = null;
    this.sendLogin({ ...this.pendingCredentials, token });
  }

  backToCredentials(): void {
    this.showTwoFactor = false;
    this.serverError = null;
  }

  private sendLogin(credentials: LoginCredentials): void {
    this.loading = true;
    this.api
      .post<AuthApiEnvelope<AuthResponse>>('auth/login', credentials)
      .pipe(finalize(() => (this.loading = false)))
      .subscribe({
        next: (response) => {
          const result = response.data;
          if (!result?.accessToken || !result.refreshToken) {
            this.serverError = 'Sign in succeeded but no authentication tokens were returned.';
            return;
          }
          this.auth.setTokens(result.accessToken, result.refreshToken);
          this.auth.setUser(result.user ?? null);
          this.showTwoFactor = false;
          this.pendingCredentials = null;
          void this.router.navigateByUrl(this.getReturnUrl());
        },
        error: (error: unknown) => {
          const message = authErrorMessage(error, 'Unable to sign in. Please try again.');
          if (error instanceof HttpErrorResponse && message === '2FA token required') {
            this.showTwoFactor = true;
            this.serverError = null;
            return;
          }
          this.serverError = message;
        },
      });
  }

  private getReturnUrl(): string {
    const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
    if (returnUrl?.startsWith('/') && !returnUrl.startsWith('//')) {
      return returnUrl;
    }
    return '/';
  }
}
