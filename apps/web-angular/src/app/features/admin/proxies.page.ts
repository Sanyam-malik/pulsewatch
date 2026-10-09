import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize, switchMap } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, Proxy } from './admin.types';

const PROTOCOLS = ['http', 'https', 'socks', 'socks5', 'socks5h', 'socks4'] as const;

@Component({
  selector: 'app-proxies-page',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      @if (editing) {
        <a routerLink="/proxies">← Proxies</a>
        <h1>{{ id ? 'Edit proxy' : 'New proxy' }}</h1>
        @if (loading) { <p>Loading proxy…</p> }
        @else {
          <section class="card">
            <form class="fields" [formGroup]="form" (ngSubmit)="save()">
              <label class="field">Protocol<select formControlName="protocol">
                @for (protocol of protocols; track protocol) { <option [value]="protocol">{{ protocol.toUpperCase() }}</option> }
              </select></label>
              <div class="grid">
                <label class="field">Host<input formControlName="host" required></label>
                <label class="field">Port<input type="number" min="1" max="65535" formControlName="port" required></label>
              </div>
              <label><input class="check" type="checkbox" formControlName="auth"> Authentication required</label>
              @if (form.controls.auth.value) {
                <label class="field">Username<input formControlName="username"></label>
                <label class="field">Password<input type="password" formControlName="password"></label>
              }
              @if (error) { <p class="error" role="alert">{{ error }}</p> }
              <div class="actions"><button type="submit" [disabled]="working || form.invalid">{{ working ? 'Saving…' : (id ? 'Update proxy' : 'Create proxy') }}</button></div>
            </form>
          </section>
        }
      } @else {
        <div class="row"><h1>Proxies</h1><a routerLink="/proxies/new"><button type="button">Create proxy</button></a></div>
        <label class="field">Search proxies<input type="search" [value]="search" (input)="search = $any($event.target).value; loadList()" placeholder="Search host or protocol"></label>
        @if (loading) { <p>Loading proxies…</p> }
        @if (error) { <p class="error" role="alert">{{ error }}</p> }
        @for (proxy of proxies; track proxy.id) {
          <section class="card row">
            <a [routerLink]="['/proxies', proxy.id, 'edit']"><strong>{{ proxy.host }}:{{ proxy.port }}</strong><div class="muted">{{ proxy.protocol.toUpperCase() }} {{ proxy.auth ? '(auth)' : '' }}</div></a>
            <button class="danger" type="button" (click)="remove(proxy)">Delete</button>
          </section>
        }
        @if (!loading && !proxies.length) { <section class="card">No proxies found. Create one to get started.</section> }
      }
    </main>`,
})
export class ProxiesPage {
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  readonly protocols = PROTOCOLS;
  readonly form = this.fb.nonNullable.group({
    protocol: ['https' as typeof PROTOCOLS[number], Validators.required],
    host: ['', Validators.required],
    port: [80, [Validators.required, Validators.min(1), Validators.max(65535)]],
    auth: [false],
    username: [''],
    password: [''],
  });
  readonly id = this.route.snapshot.paramMap.get('id');
  readonly editing = this.route.snapshot.routeConfig?.path === 'proxies/new' || !!this.id;
  proxies: Proxy[] = [];
  search = '';
  loading = false;
  working = false;
  error = '';
  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    if (this.id) this.loadProxy();
    else if (!this.editing) this.loadList();
  }
  loadList(): void {
    if (this.searchTimer) clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => {
      this.loading = true;
      const params = new URLSearchParams({ page: '0', limit: '100' });
      if (this.search.trim()) params.set('q', this.search.trim());
      this.api.get<ApiEnvelope<Proxy[]>>(`proxies?${params.toString()}`).pipe(finalize(() => this.loading = false)).subscribe({
        next: ({ data }) => this.proxies = data ?? [], error: () => this.error = 'Could not load proxies.',
      });
    }, 250);
  }
  loadProxy(): void {
    this.loading = true;
    this.api.get<ApiEnvelope<Proxy>>(`proxies/${this.id}`).pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ data }) => this.form.patchValue({
        protocol: data.protocol as typeof PROTOCOLS[number], host: data.host, port: data.port,
        auth: data.auth, username: data.username ?? '', password: data.password ?? '',
      }),
      error: () => this.error = 'Could not load proxy.',
    });
  }
  save(): void {
    if (this.form.invalid || this.working) return;
    this.working = true;
    const value = this.form.getRawValue();
    const body = { ...value, username: value.auth ? value.username : undefined, password: value.auth ? value.password : undefined };
    const request = this.id ? this.api.put(`proxies/${this.id}`, body) : this.api.post('proxies', body);
    request.pipe(finalize(() => this.working = false)).subscribe({
      next: () => void this.router.navigateByUrl('/proxies'),
      error: () => this.error = `Could not ${this.id ? 'update' : 'create'} proxy.`,
    });
  }
  remove(proxy: Proxy): void {
    if (!confirm(`Delete proxy ${proxy.host}:${proxy.port}?`)) return;
    this.api.delete(`proxies/${proxy.id}`).subscribe({
      next: () => this.proxies = this.proxies.filter((item) => item.id !== proxy.id),
      error: () => this.error = 'Could not delete proxy.',
    });
  }
}
