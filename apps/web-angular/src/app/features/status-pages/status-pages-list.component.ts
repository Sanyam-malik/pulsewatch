import { HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, StatusPage } from './status-pages.types';

@Component({
  selector: 'app-status-pages-list',
  standalone: true,
  imports: [FormsModule],
  template: `
    <main class="feature-page">
      <header class="page-header">
        <div><h1>Status pages</h1><p>Manage public status pages, custom domains, and monitored services.</p></div>
        <button type="button" (click)="router.navigateByUrl('/status-pages/new')">Create status page</button>
      </header>
      <label class="search">Search status pages
        <input type="search" [(ngModel)]="query" (ngModelChange)="searchChanged()" placeholder="Search by title or slug" />
      </label>
      @if (error) { <p class="error" role="alert">{{ error }}</p> }
      @if (loading && pages.length === 0) { <p aria-live="polite">Loading status pages…</p> }
      @for (page of pages; track page.id) {
        <article class="page-card">
          <button class="page-summary" type="button" (click)="edit(page)">
            <span class="badge" [class.draft]="!page.published">{{ page.published ? 'Published' : 'Draft' }}</span>
            <span><strong>{{ page.title }}</strong><small>/status/{{ page.slug }}</small></span>
          </button>
          <div class="actions">
            <a [href]="'/status/' + page.slug" target="_blank" rel="noopener">View</a>
            <button type="button" (click)="edit(page)">Edit</button>
            <button type="button" class="danger" (click)="remove(page)">Delete</button>
          </div>
        </article>
      }
      @if (!loading && pages.length === 0) {
        <section class="empty"><h2>No status pages found</h2><p>Create your first public status page to get started.</p>
          <button type="button" (click)="router.navigateByUrl('/status-pages/new')">Create status page</button>
        </section>
      }
      @if (hasMore) { <button type="button" [disabled]="loading" (click)="loadNext()">Load more</button> }
    </main>
  `,
  styles: [`
    .feature-page { max-width: 960px; margin: 2rem auto; padding: 0 1rem; font: 16px/1.5 system-ui,sans-serif; }
    .page-header, .page-card, .page-summary, .actions { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
    .page-header { margin-bottom: 1.5rem; } h1 { margin-bottom: .25rem; }
    .page-card { border: 1px solid #ddd; border-radius: .75rem; padding: .9rem; margin: .6rem 0; }
    .page-summary { border: 0; background: none; text-align: left; flex: 1; justify-content: flex-start; cursor: pointer; }
    .page-summary span:last-child { display: grid; } small { color: #667085; }
    .badge { padding: .15rem .55rem; border-radius: 1rem; background: #d1fadf; color: #067647; font-size: .8rem; }
    .badge.draft { background: #eaecf0; color: #344054; }
    button, .actions a { cursor: pointer; padding: .45rem .75rem; border: 1px solid #98a2b3; border-radius: .4rem; background: white; color: inherit; text-decoration: none; }
    .actions .danger, .error { color: #b42318; } .search { display: grid; gap: .3rem; max-width: 28rem; margin: 1rem 0; }
    input { padding: .6rem; border: 1px solid #98a2b3; border-radius: .4rem; } .empty { text-align: center; padding: 3rem 1rem; border: 1px dashed #98a2b3; border-radius: .75rem; }
    @media(max-width:600px) { .page-header, .page-card { align-items: flex-start; flex-direction: column; } }
  `],
})
export class StatusPagesListComponent {
  readonly api = inject(ApiService);
  readonly router = inject(Router);
  pages: StatusPage[] = [];
  query = '';
  page = 0;
  loading = false;
  hasMore = false;
  error = '';
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
    }, 350);
  }

  load(reset = false): void {
    if (this.loading) return;
    if (reset) {
      this.page = 0;
      this.pages = [];
      this.hasMore = false;
    }
    this.loading = true;
    this.error = '';
    const params = new HttpParams().set('page', this.page).set('limit', 20)
      .set('q', this.query.trim());
    this.api.get<ApiEnvelope<StatusPage[]>>('status-pages', params).subscribe({
      next: (response) => {
        const items = response.data ?? [];
        this.pages = [...this.pages, ...items];
        this.hasMore = items.length === 20;
        this.page += 1;
        this.loading = false;
      },
      error: (err: unknown) => {
        this.error = err instanceof HttpErrorResponse ? err.error?.message ?? 'Unable to load status pages.' : 'Unable to load status pages.';
        this.loading = false;
      },
    });
  }

  loadNext(): void { this.load(); }

  edit(page: StatusPage): void {
    if (page.id) void this.router.navigate(['/status-pages', page.id, 'edit']);
  }

  remove(page: StatusPage): void {
    if (!page.id || !confirm(`Delete “${page.title ?? 'this status page'}”?`)) return;
    this.api.delete<void>(`status-pages/${encodeURIComponent(page.id)}`).subscribe({
      next: () => { this.pages = this.pages.filter((item) => item.id !== page.id); },
      error: () => { this.error = 'Unable to delete this status page.'; },
    });
  }
}
