import { Component, inject } from '@angular/core';
import { HttpParams } from '@angular/common/http';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, Monitor, MonitorGroup } from './admin.types';

@Component({
  selector: 'app-monitor-groups-page',
  standalone: true,
  imports: [ReactiveFormsModule],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      <h1>Monitor groups</h1>
      <section class="card">
        <p>Organize monitors into reusable sets. These are separate from workspace access groups.</p>
        <form class="grid" [formGroup]="createForm" (ngSubmit)="create()">
          <label class="field">Group name<input formControlName="name" maxlength="255" required></label>
          <label class="field">Description<textarea formControlName="description" maxlength="2000"></textarea></label>
          <button [disabled]="working || createForm.invalid">Create group</button>
        </form>
      </section>
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      @if (loading) { <p>Loading monitor groups…</p> }
      @if (!loading && !groups.length) { <section class="card">No monitor groups yet. Create one to organize related monitors.</section> }
      @if (groups.length) {
        <section class="card">
          <h2>Groups</h2>
          <div class="actions">
            @for (group of groups; track group.id) {
              <button class="secondary" [attr.aria-pressed]="group.id === selected?.id" (click)="select(group)">{{ group.name }} ({{ group.monitor_ids.length }})</button>
            }
          </div>
        </section>
      }
      @if (selected) {
        <section class="card">
          <h2>{{ selected.name }}</h2>
          @if (selected.description) { <p>{{ selected.description }}</p> }
          @if (monitorsLoading) { <p>Loading monitors…</p> }
          @else if (!monitors.length) { <p>Create monitors before assigning them to a group.</p> }
          @else {
            @for (monitor of monitors; track monitor.id) {
              <label class="row">
                <span><input class="check" type="checkbox" [checked]="selectedMonitorIDs.includes(monitor.id)" (change)="toggle(monitor.id)"> {{ monitor.name }}</span>
                <span class="muted">{{ monitor.type }}</span>
              </label>
            }
          }
          <div class="actions">
            <button class="danger" (click)="removeGroup()" [disabled]="working">Delete group</button>
            <button (click)="save()" [disabled]="working || monitorsLoading">Save monitors</button>
          </div>
        </section>
      }
    </main>`,
})
export class MonitorGroupsPage {
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  readonly createForm = this.fb.nonNullable.group({
    name: ['', [Validators.required, Validators.maxLength(255)]],
    description: ['', Validators.maxLength(2000)],
  });
  groups: MonitorGroup[] = [];
  monitors: Monitor[] = [];
  selected: MonitorGroup | undefined;
  selectedMonitorIDs: string[] = [];
  loading = true;
  monitorsLoading = true;
  working = false;
  error = '';

  constructor() {
    this.api.get<ApiEnvelope<MonitorGroup[]>>('monitor-groups').pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ data }) => { this.groups = data ?? []; this.selected = this.groups[0]; this.selectedMonitorIDs = [...(this.selected?.monitor_ids ?? [])]; },
      error: () => this.error = 'Could not load monitor groups.',
    });
    this.api.get<ApiEnvelope<Monitor[]>>('monitors', new HttpParams().set('page', '0').set('limit', '100')).pipe(finalize(() => this.monitorsLoading = false)).subscribe({
      next: ({ data }) => this.monitors = data ?? [],
      error: () => this.error = 'Could not load monitors.',
    });
  }

  create(): void {
    if (this.createForm.invalid || this.working) return;
    this.working = true;
    this.api.post<ApiEnvelope<MonitorGroup>>('monitor-groups', { ...this.createForm.getRawValue(), monitor_ids: [] })
      .pipe(finalize(() => this.working = false))
      .subscribe({ next: ({ data }) => { this.groups = [...this.groups, data]; this.select(data); this.createForm.reset(); }, error: () => this.error = 'Could not create monitor group.' });
  }
  select(group: MonitorGroup): void { this.selected = group; this.selectedMonitorIDs = [...group.monitor_ids]; }
  toggle(id: string): void { this.selectedMonitorIDs = this.selectedMonitorIDs.includes(id) ? this.selectedMonitorIDs.filter((item) => item !== id) : [...this.selectedMonitorIDs, id]; }
  save(): void {
    if (!this.selected || this.working) return;
    this.working = true;
    this.api.put(`monitor-groups/${this.selected.id}/monitors`, { monitor_ids: this.selectedMonitorIDs })
      .pipe(finalize(() => this.working = false))
      .subscribe({ next: () => { this.selected!.monitor_ids = [...this.selectedMonitorIDs]; }, error: () => this.error = 'Could not save monitor membership.' });
  }
  removeGroup(): void {
    if (!this.selected || this.working) return;
    this.working = true;
    const id = this.selected.id;
    this.api.delete(`monitor-groups/${id}`).pipe(finalize(() => this.working = false)).subscribe({
      next: () => { this.groups = this.groups.filter((group) => group.id !== id); this.selected = this.groups[0]; this.selectedMonitorIDs = [...(this.selected?.monitor_ids ?? [])]; },
      error: () => this.error = 'Could not delete monitor group.',
    });
  }
}
