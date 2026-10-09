import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope } from './admin.types';

interface Setting { type?: string; value?: string; }
const TZ_KEY = 'selectedTimezone';
const RETENTION_KEY = 'KEEP_DATA_PERIOD_DAYS';

@Component({
  selector: 'app-settings-page',
  standalone: true,
  imports: [ReactiveFormsModule],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      <h1>Settings</h1>
      <section class="card">
        <h2>Timezone</h2>
        <p>Choose the timezone used to display dates and times.</p>
        <label class="field">Timezone
          <input list="timezone-options" [value]="timezone" (input)="setTimezone($any($event.target).value)" placeholder="Search or enter a timezone">
          <datalist id="timezone-options">@for (zone of timezones; track zone) { <option [value]="zone"></option> }</datalist>
        </label>
        <p class="muted">Current time: {{ currentTime }}</p>
      </section>
      <section class="card">
        <h2>Data retention</h2>
        <p>Set the number of days that monitoring data is retained.</p>
        @if (loading) { <p>Loading…</p> }
        <form class="actions" [formGroup]="form" (ngSubmit)="save()">
          <label class="field">Days<input type="number" min="1" formControlName="value"></label>
          <button [disabled]="loading || saving || form.invalid">{{ saving ? 'Saving…' : 'Save' }}</button>
        </form>
        @if (error) { <p class="error" role="alert">{{ error }}</p> }
        @if (saved) { <p class="success">Setting updated.</p> }
      </section>
    </main>`,
})
export class SettingsPage {
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  readonly form = this.fb.nonNullable.group({ value: [365, [Validators.required, Validators.min(1), Validators.pattern(/^\d+$/)]] });
  readonly timezones = this.getTimezones();
  timezone = localStorage.getItem(TZ_KEY) || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
  loading = true;
  saving = false;
  saved = false;
  error = '';

  constructor() {
    this.api.get<ApiEnvelope<Setting>>(`settings/key/${RETENTION_KEY}`).pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ data }) => { if (data?.value) this.form.patchValue({ value: Number(data.value) }); },
      error: () => this.error = 'Could not load data-retention setting.',
    });
  }
  get currentTime(): string {
    try { return new Intl.DateTimeFormat(undefined, { dateStyle: 'full', timeStyle: 'long', timeZone: this.timezone }).format(new Date()); }
    catch { return new Date().toLocaleString(); }
  }
  setTimezone(value: string): void {
    try {
      new Intl.DateTimeFormat(undefined, { timeZone: value });
      this.timezone = value;
      localStorage.setItem(TZ_KEY, value);
    } catch { /* Ignore incomplete or invalid values while the user is typing. */ }
  }
  save(): void {
    if (this.form.invalid || this.saving) return;
    this.saving = true;
    this.saved = false;
    this.error = '';
    this.api.put(`settings/key/${RETENTION_KEY}`, { type: 'int', value: String(this.form.controls.value.value) })
      .pipe(finalize(() => this.saving = false)).subscribe({
        next: () => this.saved = true, error: () => this.error = 'Could not update data-retention setting.',
      });
  }
  private getTimezones(): string[] {
    const intl = Intl as typeof Intl & { supportedValuesOf?: (key: 'timeZone') => string[] };
    return intl.supportedValuesOf?.('timeZone') ?? ['UTC', 'America/Los_Angeles', 'America/New_York', 'Europe/London', 'Europe/Paris', 'Asia/Tokyo', 'Asia/Kolkata', 'Australia/Sydney'];
  }
}
