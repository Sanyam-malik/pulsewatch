import { Component, DestroyRef, OnInit, inject, signal } from '@angular/core';
import { HttpParams } from '@angular/common/http';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { HeartbeatModel, MonitorEnvelope, MonitorModel, parseMonitorConfig } from './monitor.models';

@Component({
  selector: 'app-monitor-view',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './monitor-view.component.html',
  styleUrl: './monitor-view.component.scss',
})
export class MonitorViewComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  readonly monitor = signal<MonitorModel | null>(null);
  readonly heartbeats = signal<HeartbeatModel[]>([]);
  readonly config = signal<Record<string, unknown>>({});
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal('');
  private id = '';

  ngOnInit(): void {
    this.id = this.route.snapshot.paramMap.get('id') ?? '';
    if (!this.id) {
      this.error.set('Monitor ID is missing.');
      this.loading.set(false);
      return;
    }
    this.api.get<MonitorEnvelope<MonitorModel>>(`monitors/${encodeURIComponent(this.id)}`)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          this.monitor.set(response.data);
          this.config.set(parseMonitorConfig(response.data));
          this.loading.set(false);
        },
        error: () => {
          this.error.set('Unable to load this monitor.');
          this.loading.set(false);
        },
      });
    const params = new HttpParams().set('limit', 150).set('reverse', true);
    this.api.get<MonitorEnvelope<HeartbeatModel[]>>(
      `monitors/${encodeURIComponent(this.id)}/heartbeats`, params,
    ).pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (response) => this.heartbeats.set(response.data ?? []) });
  }

  toggleActive(): void {
    const monitor = this.monitor();
    if (!monitor) return;
    this.mutate('patch', { active: !monitor.active }, () => {
      this.monitor.update((current) => current ? { ...current, active: !current.active } : current);
    });
  }

  delete(): void {
    if (!window.confirm(`Delete "${this.monitor()?.name ?? 'this monitor'}"? This cannot be undone.`)) return;
    this.busy.set(true);
    this.api.delete<unknown>(`monitors/${encodeURIComponent(this.id)}`)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => void this.router.navigate(['/monitors']),
        error: () => { this.error.set('Unable to delete this monitor.'); this.busy.set(false); },
      });
  }

  reset(): void {
    if (!window.confirm('Reset this monitor? Its current status will be cleared.')) return;
    this.mutate('post', undefined, () => this.heartbeats.set([]), '/reset');
  }

  clone(): void {
    const monitor = this.monitor();
    if (monitor) void this.router.navigate(['/monitors/new'], { state: { cloneData: monitor } });
  }

  statusLabel(status: unknown): string {
    if (status === 1 || status === true) return 'Up';
    if (status === 0 || status === false) return 'Down';
    if (status === 3) return 'Maintenance';
    return 'Unknown';
  }

  configEntries(): Array<{ key: string; value: unknown }> {
    return Object.entries(this.config()).map(([key, value]) => ({ key, value }));
  }

  formatValue(key: string, value: unknown): string {
    if (/password|secret|tlskey|clientkey/i.test(key)) return '••••••••';
    return typeof value === 'string' ? value : JSON.stringify(value);
  }

  private mutate(method: 'patch' | 'post', body: unknown, success: () => void, suffix = ''): void {
    this.busy.set(true);
    this.error.set('');
    const path = `monitors/${encodeURIComponent(this.id)}${suffix}`;
    const request = method === 'patch'
      ? this.api.patch<unknown>(path, body)
      : this.api.post<unknown>(path, body);
    request.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: () => { success(); this.busy.set(false); },
      error: () => { this.error.set('Unable to update this monitor.'); this.busy.set(false); },
    });
  }
}
