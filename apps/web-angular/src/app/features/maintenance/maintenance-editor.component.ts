import { Component, inject } from '@angular/core';
import { HttpParams } from '@angular/common/http';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, Monitor } from '../status-pages/status-pages.types';
import { Maintenance, MaintenanceFormValue, MaintenanceStrategy, maintenancePayload, maintenanceValidation } from './maintenance.types';

@Component({
  selector: 'app-maintenance-editor',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  template: `
    <main class="feature-page">
      <a routerLink="/maintenances">← Back to maintenance</a>
      <h1>{{ isEdit ? 'Edit maintenance' : 'Schedule maintenance' }}</h1>
      @if (loading) { <p>Loading maintenance window…</p> }
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      <form [formGroup]="form" (ngSubmit)="submit()">
        <fieldset [disabled]="saving || loading || monitorLoading || associationLoading">
          <section class="panel">
            <label>Title<input formControlName="title" placeholder="Maintenance title" /></label>
            @if (form.controls.title.touched && form.controls.title.invalid) { <small class="error">A title is required.</small> }
            <label>Description<textarea formControlName="description" rows="4" placeholder="Describe the maintenance window"></textarea></label>
            <small>Markdown can be used in descriptions.</small>
            <label class="section-label">Affected monitors</label>
            <input type="search" [value]="monitorQuery" (input)="searchMonitors($any($event.target).value)" placeholder="Search monitors" aria-label="Search monitors" />
            @if (monitorLoading) { <p>Loading monitors…</p> }
            <div class="monitor-list">
              @for (monitor of monitorOptions; track monitor.id) {
                <label class="choice"><input type="checkbox" [checked]="isSelected(monitor)" (change)="toggleMonitor(monitor, $any($event.target).checked)" />{{ monitor.name || 'Unnamed monitor' }}</label>
              }
            </div>
            @if (selectedMonitors.length) { <small>{{ selectedMonitors.length }} monitor(s) selected</small> }
          </section>
          <section class="panel">
            <h2>Date and time</h2>
            <label>Strategy
              <select formControlName="strategy">
                <option value="manual">Manual</option>
                <option value="single">One-time window</option>
                <option value="cron">Cron expression</option>
                <option value="recurring-interval">Recurring interval</option>
                <option value="recurring-weekday">Recurring weekdays</option>
                <option value="recurring-day-of-month">Recurring days of month</option>
              </select>
            </label>
            @if (strategy !== 'manual') {
              @if (strategy === 'cron') {
                <label>Cron expression<input formControlName="cron" placeholder="30 3 * * *" /></label>
                <p>{{ cronDescription }}</p>
                <label>Duration (minutes)<input type="number" min="1" step="1" formControlName="duration" /></label>
                <small>Five-field cron schedule; duration is the maintenance window length.</small>
              }
              @if (strategy === 'recurring-interval') {
                <label>Repeat every (days)<input type="number" min="1" max="3650" formControlName="intervalDay" /></label>
              }
              @if (strategy === 'recurring-weekday') {
                <div><span class="section-label">Days of week</span><div class="choices">
                  @for (day of weekdays; track day.value) {
                    <label class="day"><span>{{ day.label }}</span><input type="checkbox" [checked]="form.controls.weekdays.value.includes(day.value)" (change)="toggleNumber('weekdays', day.value, $any($event.target).checked)" /></label>
                  }
                </div></div>
              }
              @if (strategy === 'recurring-day-of-month') {
                <div><span class="section-label">Days of month</span><div class="days">
                  @for (day of monthDays; track day) {
                    <label class="day"><span>{{ day }}</span><input type="checkbox" [checked]="form.controls.daysOfMonth.value.includes(day)" (change)="toggleNumber('daysOfMonth', day, $any($event.target).checked)" /></label>
                  }
                </div></div>
              }
              @if (strategy === 'recurring-interval' || strategy === 'recurring-weekday' || strategy === 'recurring-day-of-month') {
                <div>
                  <span class="section-label">Maintenance time window</span>
                  <div class="grid">
                    <label>Start time<input type="time" formControlName="startTime" /></label>
                    <label>End time<input type="time" formControlName="endTime" /></label>
                  </div>
                </div>
              }
              <label>Timezone
                <select formControlName="timezone">
                  @for (zone of timezones; track zone.value) { <option [value]="zone.value">{{ zone.label }}</option> }
                </select>
              </label>
              <p class="hint">Scheduled wall-clock times are interpreted in this timezone. “Same as server” uses the server timezone.</p>
              <span class="section-label">Effective date range</span>
              <div class="grid">
                <label>Start date and time<input type="datetime-local" formControlName="startDateTime" /></label>
                <label>End date and time<input type="datetime-local" formControlName="endDateTime" /></label>
              </div>
            }
            @if (strategy === 'manual') { <p>This maintenance window is controlled manually and has no scheduled dates.</p> }
          </section>
          <div class="actions">
            <label class="active"><input type="checkbox" formControlName="active" /> Active</label>
            <button type="button" (click)="router.navigateByUrl('/maintenances')">Cancel</button>
            <button type="submit" [disabled]="saving">{{ saving ? 'Saving…' : 'Save maintenance' }}</button>
          </div>
        </fieldset>
      </form>
    </main>
  `,
  styles: [`
    .feature-page { max-width:850px; margin:2rem auto; padding:0 1rem; font:16px/1.5 system-ui,sans-serif; }
    .panel { border:1px solid #ddd; border-radius:.7rem; padding:1rem; margin:1rem 0; display:grid; gap:.8rem; }
    h2 { margin:0; } label { display:grid; gap:.3rem; } input:not([type=checkbox]), textarea, select { width:100%; box-sizing:border-box; padding:.6rem; border:1px solid #98a2b3; border-radius:.4rem; background:white; }
    .grid { display:grid; grid-template-columns:1fr 1fr; gap:1rem; } .monitor-list { display:grid; grid-template-columns:repeat(auto-fill,minmax(220px,1fr)); max-height:14rem; overflow:auto; gap:.35rem; }
    .choice, .active { display:flex; align-items:center; gap:.5rem; } .choice input, .active input, .day input { width:auto; }
    .section-label { font-weight:600; } .choices { display:flex; flex-wrap:wrap; gap:.7rem; } .days { display:grid; grid-template-columns:repeat(8,minmax(2rem,1fr)); gap:.5rem; }
    .day { display:flex; align-items:center; justify-content:center; gap:.25rem; } .hint, small { color:#667085; } .error { color:#b42318; }
    .actions { display:flex; justify-content:flex-end; align-items:center; gap:.7rem; } .active { margin-right:auto; }
    button { cursor:pointer; padding:.55rem .8rem; border:1px solid #98a2b3; border-radius:.4rem; background:white; } button[type=submit] { background:#175cd3; color:white; }
    @media(max-width:600px) { .grid { grid-template-columns:1fr; } .days { grid-template-columns:repeat(6,minmax(2rem,1fr)); } }
  `],
})
export class MaintenanceEditorComponent {
  private readonly fb = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly router = inject(Router);
  readonly id = this.route.snapshot.paramMap.get('id');
  readonly isEdit = !!this.id;
  readonly weekdays = [
    { value: 0, label: 'Sun' }, { value: 1, label: 'Mon' }, { value: 2, label: 'Tue' },
    { value: 3, label: 'Wed' }, { value: 4, label: 'Thu' }, { value: 5, label: 'Fri' }, { value: 6, label: 'Sat' },
  ];
  readonly monthDays = Array.from({ length: 31 }, (_, index) => index + 1);
  readonly timezones = [
    { value: 'SAME_AS_SERVER', label: 'Same as server' },
    ...['UTC', ...(typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : ['America/Los_Angeles', 'America/New_York', 'Europe/London', 'Asia/Kolkata', 'Asia/Tokyo'])].filter((zone, index, all) => all.indexOf(zone) === index).map((zone) => ({
      value: zone, label: `${zone} (${this.timezoneOffset(zone)})`,
    })),
  ];
  readonly form = this.fb.nonNullable.group({
    title: ['', Validators.required],
    description: [''],
    active: [true],
    strategy: ['single' as MaintenanceStrategy],
    timezone: ['SAME_AS_SERVER'],
    startDateTime: [this.localDateTime(new Date())],
    endDateTime: [this.localDateTime(new Date(Date.now() + 60 * 60 * 1000))],
    startTime: [''],
    endTime: [''],
    cron: [''],
    duration: [60],
    intervalDay: [1],
    weekdays: [[] as number[]],
    daysOfMonth: [[] as Array<number | string>],
  });
  selectedMonitors: Monitor[] = [];
  monitorOptions: Monitor[] = [];
  monitorQuery = '';
  loading = false;
  monitorLoading = false;
  associationLoading = false;
  associationFailed = false;
  saving = false;
  error = '';
  private monitorTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    this.searchMonitors('');
    if (this.id) this.load(this.id);
  }

  get strategy(): MaintenanceStrategy { return this.form.controls.strategy.value; }

  get cronDescription(): string {
    const parts = this.form.controls.cron.value.trim().split(/\s+/);
    if (parts.length !== 5) return 'Enter minute hour day-of-month month day-of-week.';
    if (!/^\d+$/.test(parts[0]) || !/^\d+$/.test(parts[1])) return 'Runs according to the cron schedule and selected timezone.';
    return `Runs at ${parts[1].padStart(2, '0')}:${parts[0].padStart(2, '0')} according to the cron schedule and selected timezone.`;
  }

  submit(): void {
    if (this.saving) return;
    if (this.associationFailed) {
      this.error = 'Reload the associated monitors before saving so the maintenance window does not lose its monitor selections.';
      return;
    }
    this.form.markAllAsTouched();
    const value = this.form.getRawValue() as MaintenanceFormValue;
    this.error = maintenanceValidation(value) ?? '';
    if (this.error) return;
    this.saving = true;
    const payload = maintenancePayload(value);
    const request = this.id
      ? this.api.put<ApiEnvelope<Maintenance>>(`maintenances/${encodeURIComponent(this.id)}`, payload)
      : this.api.post<ApiEnvelope<Maintenance>>('maintenances', payload);
    request.subscribe({
      next: () => { void this.router.navigateByUrl('/maintenances'); },
      error: () => { this.error = 'Unable to save this maintenance window.'; this.saving = false; },
    });
  }

  toggleNumber(field: 'weekdays' | 'daysOfMonth', value: number, checked: boolean): void {
    const control = this.form.controls[field];
    const current = control.value as number[];
    const updated = checked ? [...current.filter((entry) => entry !== value), value] : current.filter((entry) => entry !== value);
    control.setValue(updated);
  }

  toggleMonitor(monitor: Monitor, checked: boolean): void {
    this.selectedMonitors = checked
      ? [...this.selectedMonitors.filter((item) => item.id !== monitor.id), monitor]
      : this.selectedMonitors.filter((item) => item.id !== monitor.id);
  }

  isSelected(monitor: Monitor): boolean { return this.selectedMonitors.some((item) => item.id === monitor.id); }

  searchMonitors(query: string): void {
    this.monitorQuery = query;
    if (this.monitorTimer) clearTimeout(this.monitorTimer);
    this.monitorTimer = setTimeout(() => {
      this.monitorLoading = true;
      const params = new HttpParams().set('q', this.monitorQuery).set('limit', 20).set('page', 0);
      this.api.get<ApiEnvelope<Monitor[]>>('monitors', params).subscribe({
        next: ({ data }) => { this.mergeMonitors(data ?? []); this.monitorLoading = false; },
        error: () => { this.monitorLoading = false; },
      });
    }, 250);
  }

  private load(id: string): void {
    this.loading = true;
    this.api.get<ApiEnvelope<Maintenance>>(`maintenances/${encodeURIComponent(id)}`).subscribe({
      next: ({ data }) => {
        this.form.patchValue({
          title: data.title ?? '', description: data.description ?? '', active: data.active ?? true,
          strategy: (data.strategy ?? 'single') as MaintenanceStrategy, timezone: data.timezone ?? 'SAME_AS_SERVER',
          startDateTime: data.start_date_time ? this.localDateTime(new Date(data.start_date_time)) : '',
          endDateTime: data.end_date_time ? this.localDateTime(new Date(data.end_date_time)) : '',
          startTime: data.start_time ?? '', endTime: data.end_time ?? '', cron: data.cron ?? '',
          duration: data.duration ?? 60, intervalDay: data.interval_day ?? 1,
          weekdays: data.weekdays ?? [], daysOfMonth: data.days_of_month ?? [],
        });
        this.loading = false;
        if (data.monitor_ids?.length) this.loadAssociatedMonitors(data.monitor_ids);
      },
      error: () => { this.error = 'Unable to load this maintenance window.'; this.loading = false; },
    });
  }

  private loadAssociatedMonitors(ids: string[]): void {
    this.associationLoading = true;
    this.api.get<ApiEnvelope<Monitor[]>>('monitors/batch', new HttpParams().set('ids', ids.join(','))).subscribe({
      next: ({ data }) => { this.selectedMonitors = data ?? []; this.mergeMonitors(this.selectedMonitors); this.associationLoading = false; },
      error: () => { this.error = 'Unable to load associated monitors.'; this.associationFailed = true; this.associationLoading = false; },
    });
  }

  private mergeMonitors(monitors: Monitor[]): void {
    const byId = new Map<string, Monitor>();
    for (const monitor of [...monitors, ...this.selectedMonitors]) if (monitor.id) byId.set(monitor.id, monitor);
    this.monitorOptions = [...byId.values()];
  }

  private localDateTime(date: Date): string {
    const pad = (value: number) => String(value).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  private timezoneOffset(zone: string): string {
    if (zone === 'UTC') return 'UTC';
    return new Intl.DateTimeFormat('en', { timeZone: zone, timeZoneName: 'short' })
      .formatToParts(new Date()).find((part) => part.type === 'timeZoneName')?.value ?? zone;
  }
}
