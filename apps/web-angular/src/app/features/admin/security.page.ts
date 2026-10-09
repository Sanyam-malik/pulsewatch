import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { ApiEnvelope } from './admin.types';

interface ApiKey {
  id: string; name: string; display_key: string; usage_count: number;
  max_usage_count?: number; last_used?: string; expires_at?: string;
}
interface NewApiKeyResponse { data: ApiKey & { token: string }; }
interface TwoFaSetupResponse { secret?: string; twofa_secret?: string; provisioningUri?: string; qr?: string; qr_code?: string; }

@Component({
  selector: 'app-security-page',
  standalone: true,
  imports: [ReactiveFormsModule],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      <h1>Security</h1>
      <section class="card">
        <h2>Update password</h2>
        <form class="fields" [formGroup]="passwordForm" (ngSubmit)="updatePassword()">
          <label class="field">Current password<input type="password" autocomplete="current-password" formControlName="currentPassword" required></label>
          <label class="field">New password<input type="password" autocomplete="new-password" formControlName="newPassword" required></label>
          <label class="field">Confirm new password<input type="password" autocomplete="new-password" formControlName="confirmPassword" required></label>
          @if (passwordMismatch) { <p class="error">Passwords do not match.</p> }
          <button [disabled]="working || passwordForm.invalid">Update password</button>
        </form>
      </section>
      <section class="card">
        <h2>Two-factor authentication</h2>
        @if (auth.user()?.twofa_status) {
          <p>Two-factor authentication is enabled for your account.</p>
          @if (showDisable) {
            <form class="fields" [formGroup]="disableForm" (ngSubmit)="disable2fa()">
              <label class="field">Password to disable 2FA<input type="password" formControlName="password" required></label>
              <div class="actions"><button class="danger" [disabled]="working || disableForm.invalid">Disable 2FA</button><button type="button" class="secondary" (click)="showDisable = false">Cancel</button></div>
            </form>
          } @else { <button class="danger" (click)="showDisable = true">Disable 2FA</button> }
        } @else if (step === 'password') {
          <p>Verify your password to set up an authenticator app.</p>
          <form class="fields" [formGroup]="setupPasswordForm" (ngSubmit)="start2fa()">
            <label class="field">Current password<input type="password" formControlName="password" required></label>
            <button [disabled]="working || setupPasswordForm.invalid">Continue</button>
          </form>
        } @else {
          <p>Enter the secret or provisioning URI in your authenticator app. The QR image is not shown in this Angular version.</p>
          @if (provisioningUri) { <p><code>{{ provisioningUri }}</code></p> }
          @if (secret) { <p>Secret: <code>{{ secret }}</code></p> }
          <form class="fields" [formGroup]="verifyForm" (ngSubmit)="verify2fa()">
            <label class="field">Authenticator code<input autocomplete="one-time-code" formControlName="code" required></label>
            <div class="actions"><button [disabled]="working || verifyForm.invalid">Enable 2FA</button><button type="button" class="secondary" (click)="cancel2fa()">Cancel</button></div>
          </form>
        }
      </section>
      <section class="card">
        <div class="row"><h2>API keys</h2><button (click)="showCreateKey = !showCreateKey">Create API key</button></div>
        @if (showCreateKey) {
          <form class="grid" [formGroup]="keyForm" (ngSubmit)="createKey()">
            <label class="field">Name<input formControlName="name" maxlength="255" required></label>
            <label class="field">Expires at<input type="datetime-local" formControlName="expiresAt"></label>
            <label class="field">Maximum usage<input type="number" min="1" formControlName="maxUsageCount"></label>
            <button [disabled]="working || keyForm.invalid">Create key</button>
          </form>
        }
        @if (createdToken) {
          <p class="success">API key created. Copy it now; the full token will not be shown again.</p>
          <div class="row"><code>{{ createdToken }}</code><button class="secondary" (click)="copyToken()">Copy</button><button class="secondary" (click)="createdToken = ''">Dismiss</button></div>
        }
        @if (keysLoading) { <p>Loading API keys…</p> }
        @if (error) { <p class="error" role="alert">{{ error }}</p> }
        @for (key of keys; track key.id) {
          <div class="row">
            <div><strong>{{ key.name }}</strong><div><code>{{ key.display_key }}</code></div><span class="muted">{{ key.usage_count }}{{ key.max_usage_count ? ' / ' + key.max_usage_count : '' }} uses · Last used: {{ dateTime(key.last_used) }} · Expires: {{ dateTime(key.expires_at) }}</span></div>
            <button class="danger" (click)="deleteKey(key)">Delete</button>
          </div>
        }
        @if (!keysLoading && !keys.length) { <p>No API keys yet.</p> }
      </section>
    </main>`,
})
export class SecurityPage {
  readonly auth = inject(AuthService);
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  readonly passwordForm = this.fb.nonNullable.group({
    currentPassword: ['', Validators.required],
    newPassword: ['', [Validators.required, Validators.minLength(8)]],
    confirmPassword: ['', Validators.required],
  });
  readonly setupPasswordForm = this.fb.nonNullable.group({ password: ['', Validators.required] });
  readonly disableForm = this.fb.nonNullable.group({ password: ['', Validators.required] });
  readonly verifyForm = this.fb.nonNullable.group({ code: ['', [Validators.required, Validators.minLength(6)]] });
  readonly keyForm = this.fb.nonNullable.group({
    name: ['', [Validators.required, Validators.maxLength(255)]],
    expiresAt: [''],
    maxUsageCount: ['', Validators.pattern(/^[1-9]\d*$/)],
  });
  keys: ApiKey[] = [];
  keysLoading = true;
  working = false;
  error = '';
  showDisable = false;
  showCreateKey = false;
  step: 'password' | 'verify' = 'password';
  secret = '';
  provisioningUri = '';
  createdToken = '';

  constructor() { this.loadKeys(); }
  get passwordMismatch(): boolean {
    const form = this.passwordForm;
    return form.controls.confirmPassword.touched && form.controls.newPassword.value !== form.controls.confirmPassword.value;
  }
  loadKeys(): void {
    this.api.get<ApiEnvelope<ApiKey[]>>('api-keys').pipe(finalize(() => this.keysLoading = false)).subscribe({
      next: ({ data }) => this.keys = data ?? [], error: () => this.error = 'Could not load API keys.',
    });
  }
  updatePassword(): void {
    if (this.passwordForm.invalid || this.passwordMismatch || this.working) return;
    const { currentPassword, newPassword } = this.passwordForm.getRawValue();
    this.working = true;
    this.api.put('auth/password', { currentPassword, newPassword }).pipe(finalize(() => this.working = false)).subscribe({
      next: () => this.passwordForm.reset(), error: () => this.error = 'Could not update password.',
    });
  }
  start2fa(): void {
    const email = this.auth.user()?.email;
    if (!email || this.setupPasswordForm.invalid || this.working) return;
    this.working = true;
    this.api.post<TwoFaSetupResponse>('auth/2fa/setup', { email, password: this.setupPasswordForm.controls.password.value })
      .pipe(finalize(() => this.working = false)).subscribe({
        next: (data) => {
          this.secret = data.secret ?? data.twofa_secret ?? '';
          this.provisioningUri = data.provisioningUri ?? data.qr ?? data.qr_code ?? '';
          this.step = 'verify';
        },
        error: () => this.error = 'Could not verify password or start 2FA setup.',
      });
  }
  verify2fa(): void {
    const email = this.auth.user()?.email;
    if (!email || this.verifyForm.invalid || this.working) return;
    this.working = true;
    this.api.post('auth/2fa/verify', { email, code: this.verifyForm.controls.code.value })
      .pipe(finalize(() => this.working = false)).subscribe({
        next: () => { this.auth.setUser({ ...this.auth.user(), twofa_status: true }); this.cancel2fa(); },
        error: () => this.error = 'Could not verify authenticator code.',
      });
  }
  disable2fa(): void {
    const email = this.auth.user()?.email;
    if (!email || this.disableForm.invalid || this.working) return;
    this.working = true;
    this.api.post('auth/2fa/disable', { email, password: this.disableForm.controls.password.value })
      .pipe(finalize(() => this.working = false)).subscribe({
        next: () => { this.auth.setUser({ ...this.auth.user(), twofa_status: false }); this.disableForm.reset(); this.showDisable = false; },
        error: () => this.error = 'Could not disable 2FA. Check your password.',
      });
  }
  cancel2fa(): void { this.step = 'password'; this.secret = ''; this.provisioningUri = ''; this.verifyForm.reset(); this.setupPasswordForm.reset(); }
  createKey(): void {
    if (this.keyForm.invalid || this.working) return;
    const value = this.keyForm.getRawValue();
    let expires_at: string | undefined;
    if (value.expiresAt) {
      const timezone = localStorage.getItem('selectedTimezone') || Intl.DateTimeFormat().resolvedOptions().timeZone;
      const parsed = localDateTimeInTimezoneToUtc(value.expiresAt, timezone);
      if (Number.isNaN(parsed.getTime()) || parsed <= new Date()) { this.error = 'Expiration date must be in the future.'; return; }
      expires_at = parsed.toISOString();
    }
    this.working = true;
    this.api.post<NewApiKeyResponse>('api-keys', {
      name: value.name, expires_at,
      max_usage_count: value.maxUsageCount ? Number(value.maxUsageCount) : undefined,
    }).pipe(finalize(() => this.working = false)).subscribe({
      next: (response) => {
        this.createdToken = response.data.token;
        this.showCreateKey = false;
        this.keyForm.reset();
        this.keysLoading = true;
        this.loadKeys();
      },
      error: () => this.error = 'Could not create API key.',
    });
  }
  deleteKey(key: ApiKey): void {
    if (!confirm(`Delete API key “${key.name}”?`)) return;
    this.api.delete(`api-keys/${key.id}`).subscribe({
      next: () => this.keys = this.keys.filter((item) => item.id !== key.id),
      error: () => this.error = 'Could not delete API key.',
    });
  }
  copyToken(): void { void navigator.clipboard?.writeText(this.createdToken); }
  dateTime(value?: string): string {
    if (!value) return 'Never';
    const timezone = localStorage.getItem('selectedTimezone') || Intl.DateTimeFormat().resolvedOptions().timeZone;
    try { return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short', timeZone: timezone }).format(new Date(value)); }
    catch { return new Date(value).toLocaleString(); }
  }
}

function localDateTimeInTimezoneToUtc(value: string, timezone: string): Date {
  const [datePart, timePart] = value.split('T');
  const [year, month, day] = (datePart ?? '').split('-').map(Number);
  const [hour, minute] = (timePart ?? '').split(':').map(Number);
  const targetUtc = Date.UTC(year ?? 0, (month ?? 1) - 1, day ?? 1, hour ?? 0, minute ?? 0);
  let candidate = targetUtc;
  for (let iteration = 0; iteration < 3; iteration += 1) {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
    }).formatToParts(new Date(candidate));
    const fields = Object.fromEntries(parts.map((part) => [part.type, Number(part.value)]));
    const renderedUtc = Date.UTC(fields['year'] ?? 0, (fields['month'] ?? 1) - 1, fields['day'] ?? 1, fields['hour'] ?? 0, fields['minute'] ?? 0);
    candidate += targetUtc - renderedUtc;
  }
  return new Date(candidate);
}
