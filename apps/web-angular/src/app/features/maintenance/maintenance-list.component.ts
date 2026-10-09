import { HttpParams } from '@angular/common/http';
import { DatePipe } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope } from '../status-pages/status-pages.types';
import { Maintenance } from './maintenance.types';

@Component({
  selector: 'app-maintenance-list',
  standalone: true,
  imports: [FormsModule, DatePipe],
  template: `
    <main class="feature-page">
      <header class="page-header">
        <div><h1>Maintenance windows</h1><p>Schedule and manage one-time or recurring maintenance.</p></div>
        <button type="button" (click)="router.navigateByUrl('/maintenances/new')">Schedule maintenance</button>
      </header>
      <label class="search">Search maintenance windows
        <input type="search" [(ngModel)]="query" (ngModelChange)="searchChanged()" placeholder="Search by title" />
      </label>
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      @if (loading && items.length === 0) { <p>Loading maintenance windows…</p> }
      @for (item of items; track item.id) {
        <article class="card" [class.ended]="isEnded(item)">
          <button class="summary" type="button" (click)="edit(item)">
            <strong>{{ item.title }}</strong>
            <span class="badges"><span class="badge" [class.inactive]="!item.active">{{ statusText(item) }}</span><span class="badge outline">{{ strategyLabel(item.strategy) }}</span></span>
            @if (item.description) { <span class="description">{{ item.description }}</span> }
            <span class="metadata">
              @if (item.start_date_time) { <span>◷ Starts {{ item.start_date_time | date:'medium' }}</span> }
              @if (item.end_date_time) { <span>◷ Ends {{ item.end_date_time | date:'medium' }}</span> }
              @if (item.duration) { <span>◴ {{ item.duration }} minutes</span> }
              @if (item.timezone) { <span>Timezone: {{ item.timezone }}</span> }
            </span>
          </button>
          <div class="actions">
            <button type="button" [disabled]="pendingId === item.id" (click)="toggle(item)">{{ item.active ? 'Pause' : 'Resume' }}</button>
            <button type="button" class="danger" (click)="remove(item)">Delete</button>
          </div>
        </article>
      }
      @if (!loading && items.length === 0) {
        <section class="empty"><h2>No maintenance windows found</h2><p>Schedule a maintenance window for your monitors.</p>
          <button type="button" (click)="router.navigateByUrl('/maintenances/new')">Schedule maintenance</button>
        </section>
      }
      @if (hasMore) { <button type="button" [disabled]="loading" (click)="load()">Load more</button> }
    </main>
  `,
  styles: [`
    .feature-page { max-width:1000px; margin:2rem auto; padding:0 1rem; font:16px/1.5 system-ui,sans-serif; }
    .page-header, .card, .summary, .actions, .badges, .metadata { display:flex; gap:1rem; }
    .page-header, .card { justify-content:space-between; align-items:center; } h1 { margin-bottom:.25rem; }
    .card { border:1px solid #ddd; border-radius:.75rem; padding:.9rem; margin:.6rem 0; }
    .card.ended { color:#667085; } .summary { border:0; background:none; color:inherit; flex:1; text-align:left; flex-direction:column; gap:.35rem; cursor:pointer; }
    .badges, .metadata { flex-wrap:wrap; align-items:center; gap:.5rem; } .badge { border-radius:1rem; padding:.1rem .55rem; background:#d1fadf; color:#067647; font-size:.8rem; }
    .badge.inactive { background:#eaecf0; color:#344054; } .badge.outline { border:1px solid #d0d5dd; background:white; color:inherit; }
    .description, .metadata { color:#667085; font-size:.85rem; } .search { display:grid; gap:.3rem; max-width:28rem; margin:1rem 0; }
    input { padding:.6rem; border:1px solid #98a2b3; border-radius:.4rem; } button { cursor:pointer; padding:.45rem .75rem; border:1px solid #98a2b3; border-radius:.4rem; background:white; color:inherit; }
    .danger, .error { color:#b42318; } .empty { text-align:center; padding:3rem 1rem; border:1px dashed #98a2b3; border-radius:.75rem; }
    @media(max-width:650px) { .page-header, .card { align-items:flex-start; flex-direction:column; } .actions { align-self:flex-end; } }
  `],
})
export class MaintenanceListComponent {
  readonly api = inject(ApiService);
  readonly router = inject(Router);
  items: Maintenance[] = [];
  query = '';
  page = 0;
  loading = false;
  hasMore = false;
  error = '';
  pendingId?: string;
  private searchTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    this.query = new URLSearchParams(location.search).get('q') ?? '';
    this.load(true);
  }

  searchChanged(): void {
    if (this.searchTimer) clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => {
      const params = new URLSearchParams(location.search);
      this.query.trim() ? params.set('q', this.query.trim()) : params.delete('q');
      history.replaceState(null, '', `${location.pathname}${params.size ? `?${params}` : ''}`);
      this.load(true);
    }, 300);
  }

  load(reset = false): void {
    if (this.loading) return;
    if (reset) { this.page = 0; this.items = []; this.hasMore = false; }
    this.loading = true;
    const params = new HttpParams().set('page', this.page).set('limit', 20).set('q', this.query.trim());
    this.api.get<ApiEnvelope<Maintenance[]>>('maintenances', params).subscribe({
      next: ({ data }) => {
        const pageItems = data ?? [];
        this.items = [...this.items, ...pageItems];
        this.hasMore = pageItems.length === 20;
        this.page += 1;
        this.loading = false;
      },
      error: () => { this.error = 'Unable to load maintenance windows.'; this.loading = false; },
    });
  }

  edit(item: Maintenance): void {
    if (item.id) void this.router.navigate(['/maintenances', item.id, 'edit']);
  }

  remove(item: Maintenance): void {
    if (!item.id || !confirm(`Delete “${item.title ?? 'this maintenance window'}”?`)) return;
    this.api.delete<void>(`maintenances/${encodeURIComponent(item.id)}`).subscribe({
      next: () => { this.items = this.items.filter((entry) => entry.id !== item.id); },
      error: () => { this.error = 'Unable to delete this maintenance window.'; },
    });
  }

  toggle(item: Maintenance): void {
    if (!item.id || this.pendingId) return;
    const wasActive = !!item.active;
    if (!confirm(`${wasActive ? 'Pause' : 'Resume'} this maintenance window?`)) return;
    const id = item.id;
    item.active = !wasActive;
    this.pendingId = id;
    this.api.patch<ApiEnvelope<Maintenance>>(`maintenances/${encodeURIComponent(id)}/${wasActive ? 'pause' : 'resume'}`, {}).subscribe({
      next: () => { this.pendingId = undefined; },
      error: () => {
        item.active = wasActive;
        this.error = `Unable to ${wasActive ? 'pause' : 'resume'} this maintenance window.`;
        this.pendingId = undefined;
      },
    });
  }

  isEnded(item: Maintenance): boolean {
    return !!item.end_date_time && new Date(item.end_date_time).getTime() < Date.now();
  }

  statusText(item: Maintenance): string {
    return this.isEnded(item) ? 'Ended' : item.active ? 'Active' : 'Inactive';
  }

  strategyLabel(strategy?: string): string {
    const labels: Record<string, string> = {
      manual: 'Manual', single: 'One time', cron: 'Cron', 'recurring-interval': 'Interval',
      'recurring-weekday': 'Weekdays', 'recurring-day-of-month': 'Days of month',
    };
    return strategy ? labels[strategy] ?? strategy : 'Unknown';
  }
}
