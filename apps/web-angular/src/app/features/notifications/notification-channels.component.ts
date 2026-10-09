import { HttpParams } from '@angular/common/http';
import { Component, DestroyRef, inject, OnInit } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, debounceTime, distinctUntilChanged } from 'rxjs';
import { ApiService } from '../../core/api.service';

export interface NotificationChannel {
  id?: string;
  name?: string;
  type?: string;
  config?: string;
  active?: boolean;
  is_default?: boolean;
}

interface ApiEnvelope<T> {
  data: T;
  message?: string;
}

@Component({
  selector: 'app-notification-channels',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './notification-channels.component.html',
  styleUrl: './notification-channels.component.scss',
})
export class NotificationChannelsComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly destroyRef = inject(DestroyRef);
  private readonly searchInput = new Subject<string>();
  private readonly pageSize = 20;

  channels: NotificationChannel[] = [];
  search = '';
  loading = false;
  loadingMore = false;
  deleting = false;
  error = '';
  channelToDelete: NotificationChannel | null = null;
  private page = 0;
  hasMore = false;

  ngOnInit(): void {
    this.search = this.route.snapshot.queryParamMap.get('q') ?? '';
    this.searchInput.pipe(debounceTime(350), distinctUntilChanged(), takeUntilDestroyed(this.destroyRef))
      .subscribe((value) => {
        this.search = value;
        void this.router.navigate([], {
          relativeTo: this.route,
          queryParams: { q: value || null },
          queryParamsHandling: 'merge',
          replaceUrl: true,
        });
        this.loadChannels();
      });
    this.loadChannels();
  }

  updateSearch(value: string): void {
    this.searchInput.next(value);
  }

  loadMore(): void {
    if (this.loadingMore || !this.hasMore) return;
    this.page += 1;
    this.fetchPage(this.page, true);
  }

  confirmDelete(channel: NotificationChannel, event: Event): void {
    event.stopPropagation();
    this.channelToDelete = channel;
  }

  cancelDelete(): void {
    if (!this.deleting) this.channelToDelete = null;
  }

  deleteChannel(): void {
    const id = this.channelToDelete?.id;
    if (!id || this.deleting) return;
    this.deleting = true;
    this.api.delete<unknown>(`notification-channels/${encodeURIComponent(id)}`)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.channelToDelete = null;
          this.deleting = false;
          this.loadChannels();
        },
        error: (error: unknown) => {
          this.error = this.errorMessage(error, 'Unable to delete notification channel.');
          this.deleting = false;
          this.channelToDelete = null;
        },
      });
  }

  private loadChannels(): void {
    this.page = 0;
    this.channels = [];
    this.error = '';
    this.loading = true;
    this.fetchPage(0, false);
  }

  private fetchPage(page: number, append: boolean): void {
    const params = new HttpParams()
      .set('page', page)
      .set('limit', this.pageSize)
      .set('q', this.search);
    if (append) this.loadingMore = true;
    this.api.get<ApiEnvelope<NotificationChannel[]>>('notification-channels', params)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          const data = Array.isArray(response?.data) ? response.data : [];
          this.channels = append ? [...this.channels, ...data] : data;
          this.hasMore = data.length === this.pageSize;
          this.loading = false;
          this.loadingMore = false;
        },
        error: (error: unknown) => {
          this.error = this.errorMessage(error, 'Unable to load notification channels.');
          this.loading = false;
          this.loadingMore = false;
        },
      });
  }

  private errorMessage(error: unknown, fallback: string): string {
    if (typeof error === 'object' && error !== null && 'error' in error) {
      const body = (error as { error?: { message?: unknown } }).error;
      if (typeof body?.message === 'string' && body.message) return body.message;
    }
    return fallback;
  }
}
