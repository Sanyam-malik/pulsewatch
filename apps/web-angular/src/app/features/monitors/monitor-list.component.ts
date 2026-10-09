import { Component, DestroyRef, OnInit, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HttpParams } from '@angular/common/http';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { debounceTime, distinctUntilChanged, Subscription } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { MonitorEnvelope, MonitorModel, TagModel } from './monitor.models';
import { MonitorCardComponent } from './monitor-card.component';

@Component({
  selector: 'app-monitor-list',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink, MonitorCardComponent],
  templateUrl: './monitor-list.component.html',
  styleUrl: './monitor-list.component.scss',
})
export class MonitorListComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  readonly monitors = signal<MonitorModel[]>([]);
  readonly tags = signal<TagModel[]>([]);
  readonly search = new FormControl('', { nonNullable: true });
  readonly active = new FormControl('all', { nonNullable: true });
  readonly status = new FormControl('all', { nonNullable: true });
  readonly selectedTagIds = signal<string[]>([]);
  readonly loading = signal(false);
  readonly loadingMore = signal(false);
  readonly error = signal('');
  readonly hasMore = signal(false);
  private page = 0;
  private listRequest?: Subscription;

  ngOnInit(): void {
    const query = this.route.snapshot.queryParamMap;
    this.search.setValue(query.get('search') ?? '', { emitEvent: false });
    this.active.setValue(query.get('active') ?? 'all', { emitEvent: false });
    this.status.setValue(query.get('status') ?? 'all', { emitEvent: false });
    this.selectedTagIds.set((query.get('tags') ?? '').split(',').filter(Boolean));

    this.api.get<MonitorEnvelope<TagModel[]>>('tags', new HttpParams().set('limit', 100))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (response) => this.tags.set(response.data ?? []) });
    this.search.valueChanges.pipe(debounceTime(350), distinctUntilChanged(), takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.filtersChanged());
    this.active.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => this.filtersChanged());
    this.status.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => this.filtersChanged());
    this.loadPage(true);
  }

  filtersChanged(): void {
    this.syncUrl();
    this.loadPage(true);
  }

  toggleTag(id: string): void {
    const selected = this.selectedTagIds();
    this.selectedTagIds.set(selected.includes(id) ? selected.filter((tag) => tag !== id) : [...selected, id]);
    this.syncUrl();
    this.loadPage(true);
  }

  clearFilters(): void {
    this.search.setValue('', { emitEvent: false });
    this.active.setValue('all', { emitEvent: false });
    this.status.setValue('all', { emitEvent: false });
    this.selectedTagIds.set([]);
    this.syncUrl();
    this.loadPage(true);
  }

  loadMore(): void {
    if (!this.loadingMore() && this.hasMore()) this.loadPage(false);
  }

  private loadPage(reset: boolean): void {
    if (reset) {
      this.listRequest?.unsubscribe();
      this.page = 0;
      this.monitors.set([]);
      this.loading.set(true);
    } else {
      this.page += 1;
      this.loadingMore.set(true);
    }
    let params = new HttpParams().set('limit', 20).set('page', this.page);
    const query = this.search.value.trim();
    if (query) params = params.set('q', query);
    if (this.active.value !== 'all') params = params.set('active', this.active.value === 'active');
    const statusCode: Record<string, number> = { up: 1, down: 0, maintenance: 3 };
    if (statusCode[this.status.value] !== undefined) params = params.set('status', statusCode[this.status.value]);
    if (this.selectedTagIds().length) params = params.set('tag_ids', this.selectedTagIds().join(','));
    this.error.set('');
    this.listRequest = this.api.get<MonitorEnvelope<MonitorModel[]>>('monitors', params)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          const results = response.data ?? [];
          this.monitors.update((existing) => reset ? results : [...existing, ...results]);
          this.hasMore.set(results.length === 20);
          this.loading.set(false);
          this.loadingMore.set(false);
        },
        error: () => {
          this.error.set('Unable to load monitors. Please try again.');
          this.loading.set(false);
          this.loadingMore.set(false);
        },
      });
  }

  private syncUrl(): void {
    const params: Record<string, string | null> = {
      search: this.search.value.trim() || null,
      active: this.active.value === 'all' ? null : this.active.value,
      status: this.status.value === 'all' ? null : this.status.value,
      tags: this.selectedTagIds().length ? this.selectedTagIds().join(',') : null,
    };
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: params,
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }
}
