import { HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, Monitor, StatusPage, StatusPageFormValue, statusPagePayload } from './status-pages.types';
import { DomainsManagerComponent } from './domains-manager.component';

@Component({
  selector: 'app-status-page-editor',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink, DomainsManagerComponent],
  template: `
    <main class="feature-page">
      <a routerLink="/status-pages">← Back to status pages</a>
      <h1>{{ isEdit ? 'Edit status page' : 'Create status page' }}</h1>
      <p>{{ isEdit ? 'Update your public status page.' : 'Create and configure a public status page.' }}</p>
      @if (loading) { <p>Loading status page…</p> }
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      <form [formGroup]="form" (ngSubmit)="submit()">
        <fieldset [disabled]="saving || loading || associationLoading">
          <section class="panel">
            <h2>Basic information</h2>
            <div class="grid">
              <label>Title<input formControlName="title" placeholder="Status page title" /></label>
              <label>Slug<input formControlName="slug" placeholder="my-status" /></label>
            </div>
            @if (form.controls.title.touched && form.controls.title.invalid) { <small class="error">Title is required.</small> }
            @if (form.controls.slug.touched && form.controls.slug.invalid) { <small class="error">Use lowercase letters, numbers, and hyphens only.</small> }
            <label>Description<textarea rows="3" formControlName="description"></textarea></label>
            <label>Icon URL<input type="url" formControlName="icon" placeholder="https://example.com/icon.png" /></label>

            <h3>Affected monitors</h3>
            <input type="search" [value]="monitorQuery" (input)="searchMonitors($any($event.target).value)" placeholder="Search monitors" aria-label="Search monitors" />
            @if (monitorLoading) { <p>Loading monitors…</p> }
            <div class="monitor-list">
              @for (monitor of monitorOptions; track monitor.id) {
                <label class="monitor-option"><input type="checkbox" [checked]="isSelected(monitor)" (change)="toggleMonitor(monitor, $any($event.target).checked)" />{{ monitor.name || 'Unnamed monitor' }}</label>
              }
            </div>
            @if (selectedMonitors.length) { <p>{{ selectedMonitors.length }} monitor(s) selected</p> }
          </section>
          <section class="panel">
            <h2>Customization</h2>
            <label>Footer text<input formControlName="footer_text" /></label>
            <label>Auto-refresh interval in seconds
              <input type="number" min="0" formControlName="auto_refresh_interval" />
              <small>Set to 0 to use the default 30-second refresh.</small>
            </label>
          </section>
          <section class="panel">
            <h2>Settings</h2>
            <label class="inline"><input type="checkbox" formControlName="published" /> Published (visible to the public)</label>
            <label>Custom domains</label>
            <app-domains-manager [value]="domains" [error]="domainError" [highlightedDomain]="highlightedDomain" (changed)="domains = $event" />
            <p class="hint">Point each custom domain to this service and configure DNS before use.</p>
          </section>
          <div class="actions">
            <button type="button" (click)="router.navigateByUrl('/status-pages')">Cancel</button>
            <button type="submit" [disabled]="saving">{{ saving ? 'Saving…' : (isEdit ? 'Update status page' : 'Create status page') }}</button>
          </div>
        </fieldset>
      </form>
    </main>
  `,
  styles: [`
    .feature-page { max-width: 800px; margin: 2rem auto; padding: 0 1rem; font: 16px/1.5 system-ui,sans-serif; }
    .panel { border: 1px solid #ddd; border-radius: .7rem; padding: 1rem; margin: 1rem 0; display: grid; gap: .8rem; }
    h2, h3 { margin: 0; } .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
    label { display: grid; gap: .3rem; } input:not([type=checkbox]), textarea { width: 100%; box-sizing: border-box; padding: .6rem; border: 1px solid #98a2b3; border-radius: .4rem; }
    .inline { display: flex; align-items: center; } .inline input, .monitor-option input { width: auto; }
    .monitor-list { display: grid; grid-template-columns: repeat(auto-fill,minmax(220px,1fr)); max-height: 14rem; overflow: auto; gap: .35rem; }
    .monitor-option { display: flex; gap: .5rem; align-items: center; } .actions { display: flex; justify-content: flex-end; gap: .6rem; }
    button { padding: .55rem .8rem; border: 1px solid #98a2b3; border-radius: .4rem; background: white; cursor: pointer; }
    button[type=submit] { background: #175cd3; color: white; } button:disabled { opacity: .6; cursor: wait; }
    .error { color: #b42318; } .hint, small { color: #667085; }
    @media(max-width:600px) { .grid { grid-template-columns: 1fr; } }
  `],
})
export class StatusPageEditorComponent {
  private readonly fb = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  readonly router = inject(Router);
  readonly id = this.route.snapshot.paramMap.get('id');
  readonly isEdit = !!this.id;
  readonly form = this.fb.nonNullable.group({
    title: ['', Validators.required],
    slug: ['', [Validators.required, Validators.pattern(/^[a-z0-9-]+$/)]],
    description: [''],
    icon: [''],
    footer_text: [''],
    auto_refresh_interval: [300, [Validators.min(0)]],
    published: [true],
  });
  domains: string[] = [];
  selectedMonitors: Monitor[] = [];
  monitorOptions: Monitor[] = [];
  monitorQuery = '';
  monitorLoading = false;
  associationLoading = false;
  associationFailed = false;
  loading = false;
  saving = false;
  error = '';
  domainError = '';
  highlightedDomain?: string;
  private monitorTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    this.searchMonitors('');
    if (this.id) this.loadStatusPage(this.id);
  }

  loadStatusPage(id: string): void {
    this.loading = true;
    this.api.get<ApiEnvelope<StatusPage>>(`status-pages/${encodeURIComponent(id)}`).subscribe({
      next: ({ data }) => {
        this.form.patchValue({
          title: data.title ?? '', slug: data.slug ?? '', description: data.description ?? '',
          icon: data.icon ?? '', footer_text: data.footer_text ?? '',
          auto_refresh_interval: data.auto_refresh_interval ?? 0, published: !!data.published,
        });
        this.domains = data.domains ?? [];
        if (data.monitor_ids?.length) this.loadSelectedMonitors(data.monitor_ids);
        this.loading = false;
      },
      error: () => { this.error = 'Unable to load this status page.'; this.loading = false; },
    });
  }

  private loadSelectedMonitors(ids: string[]): void {
    this.associationLoading = true;
    this.api.get<ApiEnvelope<Monitor[]>>('monitors/batch', new HttpParams().set('ids', ids.join(','))).subscribe({
      next: ({ data }) => {
        this.selectedMonitors = data ?? [];
        this.mergeMonitorOptions(this.selectedMonitors);
        this.associationLoading = false;
      },
      error: () => { this.error = 'Unable to load the associated monitors.'; this.associationFailed = true; this.associationLoading = false; },
    });
  }

  searchMonitors(query: string): void {
    this.monitorQuery = query;
    if (this.monitorTimer) clearTimeout(this.monitorTimer);
    this.monitorTimer = setTimeout(() => {
      this.monitorLoading = true;
      const params = new HttpParams().set('q', this.monitorQuery).set('limit', 20).set('page', 0);
      this.api.get<ApiEnvelope<Monitor[]>>('monitors', params).subscribe({
        next: ({ data }) => { this.mergeMonitorOptions(data ?? []); this.monitorLoading = false; },
        error: () => { this.monitorLoading = false; },
      });
    }, 250);
  }

  isSelected(monitor: Monitor): boolean {
    return this.selectedMonitors.some((selected) => selected.id === monitor.id);
  }

  toggleMonitor(monitor: Monitor, checked: boolean): void {
    this.selectedMonitors = checked
      ? [...this.selectedMonitors.filter((item) => item.id !== monitor.id), monitor]
      : this.selectedMonitors.filter((item) => item.id !== monitor.id);
  }

  submit(): void {
    if (this.associationFailed) {
      this.error = 'Reload the associated monitors before saving so the page does not lose its monitor selections.';
      return;
    }
    if (this.form.invalid || this.saving) {
      this.form.markAllAsTouched();
      return;
    }
    this.saving = true;
    this.error = '';
    this.domainError = '';
    this.highlightedDomain = undefined;
    const raw = this.form.getRawValue();
    const value: StatusPageFormValue = {
      ...raw,
      auto_refresh_interval: Number(raw.auto_refresh_interval),
      domains: this.domains,
      monitors: this.selectedMonitors.filter((monitor) => monitor.id).map((monitor) => ({
        value: monitor.id!, label: monitor.name ?? '',
      })),
    };
    const request = this.id
      ? this.api.patch<ApiEnvelope<StatusPage>>(`status-pages/${encodeURIComponent(this.id)}`, statusPagePayload(value))
      : this.api.post<ApiEnvelope<StatusPage>>('status-pages', statusPagePayload(value));
    request.subscribe({
      next: () => { void this.router.navigateByUrl('/status-pages'); },
      error: (err: unknown) => {
        const apiError = err instanceof HttpErrorResponse ? err.error?.error : undefined;
        if (apiError?.code === 'DOMAIN_EXISTS') {
          this.highlightedDomain = apiError.domain;
          this.domainError = `This domain is already used: ${apiError.domain}`;
        }
        this.error = this.domainError || 'Unable to save this status page.';
        this.saving = false;
      },
    });
  }

  private mergeMonitorOptions(monitors: Monitor[]): void {
    const byId = new Map<string, Monitor>();
    for (const monitor of [...monitors, ...this.selectedMonitors]) if (monitor.id) byId.set(monitor.id, monitor);
    this.monitorOptions = [...byId.values()];
  }
}
