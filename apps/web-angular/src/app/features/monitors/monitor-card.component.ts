import { Component, DestroyRef, OnInit, inject, input, signal } from '@angular/core';
import { HttpParams } from '@angular/common/http';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { HeartbeatModel, MonitorEnvelope, MonitorModel } from './monitor.models';

@Component({
  selector: 'app-monitor-card',
  standalone: true,
  imports: [RouterLink],
  template: `
    <a class="monitor-card" [routerLink]="['/monitors', monitor().id]">
      <div class="card-heading">
        <div>
          <span class="status" [class.up]="status() === 'Up'" [class.down]="status() === 'Down'"
            [class.maintenance]="status() === 'Maintenance'">{{ status() }}</span>
          <h2>{{ monitor().name || 'Unnamed monitor' }}</h2>
          <span class="type">{{ monitor().type }}</span>
        </div>
        <p class="interval">Check every {{ monitor().interval ?? '—' }} seconds</p>
      </div>
      @if (heartbeats().length) {
        <div class="heartbeat-strip" role="img" [attr.aria-label]="'Recent status: ' + status()">
          @for (heartbeat of heartbeats(); track heartbeat.id ?? $index) {
            <span [class.ok]="heartbeat.status === true || heartbeat.status === 1"
              [class.bad]="heartbeat.status === false || heartbeat.status === 0"
              [title]="heartbeat.time || heartbeat.msg || 'Heartbeat'"></span>
          }
        </div>
      }
    </a>
  `,
  styles: [`
    .monitor-card { display:block; color:inherit; background:#fff; border:1px solid #d8deea; border-radius:.65rem; padding:1rem; margin:.65rem 0; text-decoration:none; }
    .monitor-card:hover { border-color:#7892c4; box-shadow:0 2px 8px #25375918; }
    .card-heading { display:flex; justify-content:space-between; align-items:center; gap:1rem; }
    h2 { display:inline; margin:0 .65rem 0 0; font-size:1.05rem; }
    .type { display:inline-block; border:1px solid #c4ccda; border-radius:99px; padding:.1rem .5rem; font-size:.8rem; }
    .status { display:inline-block; margin-right:.55rem; color:#7b5b09; font-weight:700; }
    .status.up { color:#167142; } .status.down { color:#ac2525; } .status.maintenance { color:#3157a4; }
    .interval { margin:0; color:#596579; white-space:nowrap; }
    .heartbeat-strip { display:flex; gap:3px; height:14px; margin-top:.8rem; }
    .heartbeat-strip span { flex:1; border-radius:2px; background:#cbd1db; }
    .heartbeat-strip .ok { background:#38a169; } .heartbeat-strip .bad { background:#d34444; }
    @media (max-width:600px) { .card-heading { align-items:flex-start; flex-direction:column; gap:.55rem; } }
  `],
})
export class MonitorCardComponent implements OnInit {
  readonly monitor = input.required<MonitorModel>();
  readonly heartbeats = signal<HeartbeatModel[]>([]);
  private readonly api = inject(ApiService);
  private readonly destroyRef = inject(DestroyRef);
  status(): string {
    const monitor = this.monitor();
    if (monitor.active === false) return 'Paused';
    const latest = this.heartbeats().at(-1);
    if (latest?.status === true || latest?.status === 1) return 'Up';
    if (latest?.status === false || latest?.status === 0) return 'Down';
    if (latest?.status === 3) return 'Maintenance';
    return 'Unknown';
  }
  ngOnInit(): void {
    const id = this.monitor().id;
    if (!id) return;
    const params = new HttpParams().set('limit', 50).set('reverse', true);
    this.api.get<MonitorEnvelope<HeartbeatModel[]>>(`monitors/${encodeURIComponent(id)}/heartbeats`, params)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (response) => this.heartbeats.set(response.data ?? []) });
  }
}
