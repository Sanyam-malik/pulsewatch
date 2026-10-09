import { DatePipe } from '@angular/common';
import { HttpParams } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { ApiEnvelope } from './admin.types';

interface StatusPage { id: string; title: string; slug: string; }
interface IncidentUpdate { id: string; status: Incident['status']; message: string; created_at: string; }
interface Incident {
  id: string; status_page_id: string; status_page_title: string; status_page_slug: string;
  title: string; status: 'investigating' | 'identified' | 'monitoring' | 'resolved';
  created_at: string; updated_at: string; resolved_at?: string; updates: IncidentUpdate[];
}
const INCIDENT_STATUSES: Incident['status'][] = ['investigating', 'identified', 'monitoring', 'resolved'];

@Component({
  selector: 'app-incidents-page',
  standalone: true,
  imports: [DatePipe, ReactiveFormsModule],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      <h1>Incidents</h1>
      @if (!readOnly) {
        <section class="card">
          <h2>Report an incident</h2>
          <p>New incidents start as investigating and appear on the selected published status page.</p>
          <form class="fields" [formGroup]="createForm" (ngSubmit)="create()">
            <label class="field">Status page
              <select formControlName="status_page_id" required><option value="">Select a status page</option>
                @for (page of pages; track page.id) { <option [value]="page.id">{{ page.title }} ({{ page.slug }})</option> }
              </select>
            </label>
            <label class="field">Incident title<input formControlName="title" maxlength="255" required></label>
            <label class="field">Description<textarea formControlName="message" maxlength="4000" required></textarea></label>
            <button [disabled]="working || createForm.invalid">Create incident</button>
          </form>
        </section>
      }
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      @if (loading) { <p>Loading incidents…</p> }
      @else if (!incidents.length) { <section class="card">No incidents have been reported.</section> }
      @for (incident of incidents; track incident.id) {
        <section class="card">
          <div class="row"><div><h2>{{ incident.title }}</h2><p>{{ incident.status_page_title }} · {{ incident.status_page_slug }} · {{ incident.status }}</p></div><time>{{ incident.updated_at | date:'medium' }}</time></div>
          <ol>
            @for (update of incident.updates; track update.id) {
              <li><strong>{{ update.status }}</strong><p>{{ update.message }}</p><time class="muted">{{ update.created_at | date:'medium' }}</time></li>
            }
          </ol>
          @if (!readOnly && incident.status !== 'resolved') {
            <form class="fields" [formGroup]="updateForms[incident.id]" (ngSubmit)="postUpdate(incident)">
              <label class="field">Status<select formControlName="status">@for (status of statuses; track status) { <option [value]="status">{{ status }}</option> }</select></label>
              <label class="field">Update<textarea formControlName="message" maxlength="4000" required placeholder="Post an update"></textarea></label>
              <button [disabled]="working || updateForms[incident.id].invalid">Post update</button>
            </form>
          }
        </section>
      }
    </main>`,
})
export class IncidentsPage {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder);
  readonly statuses = INCIDENT_STATUSES;
  readonly readOnly = this.auth.user()?.role === 'viewer';
  readonly createForm = this.fb.nonNullable.group({
    status_page_id: ['', Validators.required], title: ['', [Validators.required, Validators.maxLength(255)]],
    message: ['', [Validators.required, Validators.maxLength(4000)]],
  });
  pages: StatusPage[] = [];
  incidents: Incident[] = [];
  updateForms: Record<string, FormGroup<{ status: FormControl<Incident['status']>; message: FormControl<string> }>> = {};
  loading = true;
  working = false;
  error = '';

  constructor() {
    this.api.get<ApiEnvelope<StatusPage[]>>('status-pages', new HttpParams().set('page', '0').set('limit', '100')).subscribe({ next: ({ data }) => this.pages = data ?? [], error: () => this.error = 'Could not load status pages.' });
    this.load();
  }
  load(): void {
    this.loading = true;
    this.api.get<ApiEnvelope<Incident[]>>('incidents').pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ data }) => {
        this.incidents = data ?? [];
        this.updateForms = {};
        for (const incident of this.incidents) this.updateForms[incident.id] = new FormGroup({
          status: new FormControl(incident.status, { nonNullable: true }),
          message: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.maxLength(4000)] }),
        });
      },
      error: () => this.error = 'Could not load incidents.',
    });
  }
  create(): void {
    if (this.readOnly || this.createForm.invalid || this.working) return;
    this.working = true;
    this.api.post('incidents', this.createForm.getRawValue()).pipe(finalize(() => this.working = false)).subscribe({
      next: () => { this.createForm.reset(); this.load(); }, error: () => this.error = 'Could not create incident.',
    });
  }
  postUpdate(incident: Incident): void {
    const form = this.updateForms[incident.id];
    if (!form || form.invalid || this.working) return;
    this.working = true;
    this.api.post(`incidents/${incident.id}/updates`, form.getRawValue()).pipe(finalize(() => this.working = false)).subscribe({
      next: () => this.load(), error: () => this.error = 'Could not post incident update.',
    });
  }
}
