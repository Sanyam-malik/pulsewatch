import { Component, inject } from '@angular/core';
import { AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { authErrorMessage } from './auth-error';
import { AuthApiEnvelope, AuthResponse, RegisterCredentials } from './auth.types';

function passwordsMatch(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirmation = control.get('confirmPassword')?.value;
  return password === confirmation ? null : { passwordMismatch: true };
}

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  templateUrl: './register.component.html',
  styleUrl: './register.component.scss',
})
export class RegisterComponent {
  private readonly formBuilder = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly form = this.formBuilder.nonNullable.group(
    {
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', [Validators.required, Validators.minLength(8)]],
    },
    { validators: passwordsMatch },
  );

  loading = false;
  serverError: string | null = null;

  get routeQueryParams(): Record<string, string> {
    const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
    return returnUrl ? { returnUrl } : {};
  }

  submit(): void {
    if (this.form.invalid || this.loading) {
      this.form.markAllAsTouched();
      return;
    }
    this.serverError = null;
    const { email, password } = this.form.getRawValue();
    const credentials: RegisterCredentials = { email, password };
    this.loading = true;
    this.api
      .post<AuthApiEnvelope<AuthResponse>>('auth/register', credentials)
      .pipe(finalize(() => (this.loading = false)))
      .subscribe({
        next: (response) => {
          const result = response.data;
          if (!result?.accessToken || !result.refreshToken) {
            this.serverError = 'Registration succeeded but no authentication tokens were returned.';
            return;
          }
          this.auth.setTokens(result.accessToken, result.refreshToken);
          this.auth.setUser(result.user ?? null);
          void this.router.navigateByUrl(this.getReturnUrl());
        },
        error: (error: unknown) => {
          this.serverError = authErrorMessage(error, 'Unable to create your account. Please try again.');
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
