import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, PublicIncident, PublicMonitor, StatusPage, heartbeatClass, overallStatus } from './status-pages.types';

@Component({
  selector: 'app-public-status-page',
  standalone: true,
  imports: [CommonModule],
  template: `
    <main class="public-page" [class.dark-theme]="darkTheme">
      @if (loading) {
        <section class="container"><div class="skeleton title"></div><div class="skeleton short"></div>
          @for (_ of [1,2,3]; track $index) { <div class="skeleton monitor"></div> }
        </section>
      } @else if (error || !statusPage?.published) {
        <section class="container"><p class="alert" role="alert">This status page could not be found.</p></section>
      } @else {
        <section class="container">
          <header class="header">
            @if (page.icon) { <img class="icon" [src]="page.icon" [alt]="page.title + ' icon'" (error)="hideIcon($event)" /> }
            <h1>{{ page.title }}</h1>
            @if (page.description) { <p class="muted">{{ page.description }}</p> }
            <button class="theme-toggle" type="button" (click)="darkTheme = !darkTheme" [attr.aria-pressed]="darkTheme">
              {{ darkTheme ? 'Use light theme' : 'Use dark theme' }}
            </button>
            <p class="overall"><span class="dot" [class]="statusClass(overall)"></span>{{ overallText }}</p>
          </header>
          @if (incidents.length) {
            <section class="incidents" aria-label="Incidents">
              <h2>Incidents</h2>
              @for (incident of incidents; track incident.id) {
                <article class="card">
                  <header class="incident-header"><h3>{{ incident.title }}</h3><span class="badge" [class.resolved]="incident.status === 'resolved'">{{ incident.status }}</span></header>
                  <p class="muted">Updated {{ incident.updated_at | date:'medium' }}</p>
                  <ol>
                    @for (update of incident.updates; track update.id) {
                      <li><strong>{{ update.status }}</strong><p>{{ update.message }}</p><time>{{ update.created_at | date:'medium' }}</time></li>
                    }
                  </ol>
                </article>
              }
            </section>
          }
          <section class="monitors" aria-label="Monitors">
            @if (monitorsLoading) { @for (_ of [1,2,3]; track $index) { <div class="skeleton monitor"></div> } }
            @else if (!monitors.length) { <article class="card empty">No monitors configured.</article> }
            @else {
              @for (monitor of monitors; track monitor.id) {
                <article class="card monitor-card">
                  <div class="monitor-summary">
                    <span class="dot" [class]="statusClass(lastStatus(monitor))"></span>
                    <h2>{{ monitor.name }}</h2>
                    <span class="uptime">{{ monitor.uptime_24h !== undefined ? '24h ' + monitor.uptime_24h.toFixed(2) + '%' : '' }}</span>
                    <span class="badge" [class]="statusClass(lastStatus(monitor))">{{ statusText(lastStatus(monitor)) }}</span>
                  </div>
                  @if (monitor.heartbeats?.length) {
                    <div class="bars" role="img" [attr.aria-label]="monitor.name + ' recent status history'">
                      @for (beat of monitor.heartbeats; track beat.id ?? $index) {
                        <span class="bar" [class]="statusClass(beat.status)" [title]="statusText(beat.status) + (beat.time ? ' · ' + (beat.time | date:'medium') : '')"></span>
                      }
                    </div>
                  }
                </article>
              }
            }
          </section>
          @if (page.footer_text) { <footer>{{ page.footer_text }}</footer> }
          <div class="refresh">
            <span>Last updated: {{ lastUpdated | date:'yyyy-MM-dd HH:mm:ss' }}</span>
            <span>Refresh in: {{ countdownText }}</span>
            <button type="button" (click)="refreshNow()">↻ Refresh now</button>
          </div>
          <p class="powered">Powered by <a href="https://github.com/Sanyam-malik/pulsewatch">Pulsewatch</a></p>
        </section>
      }
    </main>
  `,
  styles: [`
    :host { display:block; min-height:100vh; background:#f8fafc; color:#101828; }
    .public-page { min-height:100vh; padding:2rem 1rem; }
    .container { max-width:900px; margin:0 auto; }
    .header { text-align:center; margin:1rem 0 2rem; } h1 { font-size:2rem; margin:.5rem; }
    .muted, .powered { color:#667085; } .icon { width:4rem; height:4rem; object-fit:contain; }
    .overall, .monitor-summary, .incident-header, .refresh { display:flex; align-items:center; justify-content:center; gap:.65rem; }
    .overall { font-weight:600; font-size:1.1rem; }
    .card { padding:1rem; border:1px solid #e4e7ec; border-radius:.7rem; background:white; margin:.75rem 0; }
    .incident-header { justify-content:space-between; } h2, h3 { margin:.2rem 0; } .incident-header h3 { font-size:1.1rem; }
    ol { border-left:1px solid #d0d5dd; padding-left:1.2rem; } li { padding:.25rem 0 .65rem; }
    li p { margin:.2rem 0; } time { color:#667085; font-size:.8rem; }
    .monitor-summary { justify-content:flex-start; flex-wrap:wrap; } .monitor-summary h2 { flex:1; font-size:1rem; }
    .badge { border-radius:1rem; background:#eaecf0; padding:.15rem .65rem; font-size:.8rem; text-transform:capitalize; }
    .badge.up { background:#d1fadf; color:#067647; } .badge.down { background:#fee4e2; color:#b42318; }
    .badge.maintenance { background:#dbeafe; color:#175cd3; } .dot { width:.65rem; height:.65rem; border-radius:50%; background:#98a2b3; flex:none; }
    .dot.up, .bar.up { background:#12b76a; } .dot.down, .bar.down { background:#f04438; }
    .dot.maintenance, .bar.maintenance { background:#2e90fa; } .bar.unknown, .dot.unknown { background:#98a2b3; }
    .bars { display:flex; gap:2px; margin-top:.8rem; overflow:hidden; } .bar { min-width:6px; height:16px; border-radius:2px; background:#98a2b3; flex:1; }
    .uptime { color:#475467; font-size:.9rem; } footer { border-top:1px solid #eaecf0; margin-top:2rem; padding:1.2rem; text-align:center; color:#667085; }
    .refresh { flex-wrap:wrap; font-size:.85rem; color:#667085; margin:1rem 0; } button { cursor:pointer; border:0; background:none; color:#175cd3; }
    .powered { text-align:center; font-size:.8rem; } .alert { background:#fee4e2; padding:1rem; color:#b42318; }
    .theme-toggle { border:1px solid #98a2b3; border-radius:.4rem; background:transparent; padding:.35rem .65rem; }
    .dark-theme { background:#101828; color:#f9fafb; }
    .dark-theme .card { background:#1d2939; border-color:#344054; }
    .dark-theme .muted, .dark-theme .powered, .dark-theme time, .dark-theme .uptime { color:#d0d5dd; }
    .dark-theme footer { border-color:#344054; color:#d0d5dd; }
    .skeleton { border-radius:.5rem; background:#eaecf0; animation:pulse 1.3s infinite alternate; } .title { height:3rem; width:40%; margin:2rem auto; }
    .short { height:1.5rem; width:60%; margin:1rem auto 2rem; } .monitor { height:6rem; margin:.8rem 0; }
    @keyframes pulse { to { opacity:.5; } }
  `],
})
export class PublicStatusPageComponent implements OnInit, OnDestroy {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  statusPage?: StatusPage;
  monitors: PublicMonitor[] = [];
  incidents: PublicIncident[] = [];
  loading = true;
  monitorsLoading = true;
  error = false;
  overall = 1;
  overallText = 'All systems operational';
  darkTheme = false;
  lastUpdated = new Date();
  countdown = 30;
  refreshInterval = 30;
  private timer?: ReturnType<typeof setInterval>;
  private slug = '';

  get countdownText(): string {
    return `${String(Math.floor(this.countdown / 60)).padStart(2, '0')}:${String(this.countdown % 60).padStart(2, '0')}`;
  }
  get page(): StatusPage { return this.statusPage ?? {}; }

  ngOnInit(): void {
    this.slug = this.route.snapshot.paramMap.get('slug') ?? '';
    this.loadPage();
  }

  ngOnDestroy(): void {
    if (this.timer) clearInterval(this.timer);
  }

  refreshNow(): void {
    this.loadPage(true);
    if (this.statusPage?.published) this.loadPublicData();
    this.lastUpdated = new Date();
    this.countdown = this.refreshInterval;
  }

  statusClass(status?: number): string { return heartbeatClass(status); }
  lastStatus(monitor: PublicMonitor): number | undefined { return monitor.heartbeats?.at(-1)?.status; }
  statusText(status?: number): string {
    switch (status) {
      case 1: return 'Operational';
      case 0:
      case 2: return 'Down';
      case 3: return 'Maintenance';
      default: return 'Unknown';
    }
  }
  hideIcon(event: Event): void { (event.target as HTMLImageElement).style.display = 'none'; }

  private loadPage(refresh = false): void {
    if (!this.slug) { this.error = true; this.loading = false; return; }
    this.api.get<ApiEnvelope<StatusPage>>(`status-pages/slug/${encodeURIComponent(this.slug)}`).subscribe({
      next: ({ data }) => {
        this.statusPage = data;
        this.loading = false;
        if (!data?.published) { this.error = true; return; }
        const configured = Number(data.auto_refresh_interval);
        if (configured > 0) this.refreshInterval = configured;
        this.countdown = this.refreshInterval;
        this.overall = overallStatus(this.monitors);
        this.overallText = this.getOverallText(this.overall);
        if (!refresh) this.loadPublicData();
        this.startRefreshTimer();
      },
      error: () => { this.error = true; this.loading = false; },
    });
  }

  private loadPublicData(): void {
    this.api.get<ApiEnvelope<PublicMonitor[]>>(`status-pages/slug/${encodeURIComponent(this.slug)}/monitors`).subscribe({
      next: ({ data }) => {
        this.monitors = data ?? [];
        this.monitorsLoading = false;
        this.overall = overallStatus(this.monitors);
        this.overallText = this.getOverallText(this.overall);
      },
      error: () => { this.monitorsLoading = false; },
    });
    this.api.get<ApiEnvelope<PublicIncident[]>>(`status-pages/slug/${encodeURIComponent(this.slug)}/incidents`).subscribe({
      next: ({ data }) => { this.incidents = data ?? []; },
      error: () => { this.incidents = []; },
    });
  }

  private startRefreshTimer(): void {
    if (this.timer) return;
    this.timer = setInterval(() => {
      if (this.countdown <= 1) {
        this.refreshNow();
      } else {
        this.countdown -= 1;
      }
    }, 1000);
  }

  private getOverallText(status: number): string {
    switch (status) {
      case 0: return 'Partial system outage';
      case 3: return 'Under maintenance';
      default: return 'All systems operational';
    }
  }
}
