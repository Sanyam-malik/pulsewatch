import { Component, DestroyRef, OnInit, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AbstractControl, FormControl, FormRecord, ReactiveFormsModule, ValidationErrors, ValidatorFn, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import {
  MONITOR_FIELDS, MONITOR_TYPES, MonitorField, MonitorModel, MonitorType,
  NotificationChannelModel, ProxyModel, TagModel, parseMonitorConfig, serializeMonitor,
} from './monitor.models';

const DEFAULTS: Record<string, unknown> = {
  name: 'My monitor', interval: 20, timeout: 16, max_retries: 0, retry_interval: 20,
  resend_interval: 0, notification_ids: [], tag_ids: [], proxy_id: '', active: true,
};

@Component({
  selector: 'app-monitor-form',
  standalone: true,
  imports: [ReactiveFormsModule, RouterLink],
  templateUrl: './monitor-form.component.html',
  styleUrl: './monitor-form.component.scss',
})
export class MonitorFormComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  private readonly locationState = typeof history === 'undefined' ? {} : history.state as { cloneData?: MonitorModel };
  readonly types = MONITOR_TYPES;
  readonly form = new FormRecord<FormControl<unknown>>({});
  readonly tags = signal<TagModel[]>([]);
  readonly proxies = signal<ProxyModel[]>([]);
  readonly channels = signal<NotificationChannelModel[]>([]);
  readonly busy = signal(false);
  readonly error = signal('');
  readonly monitor = signal<MonitorModel | null>(null);
  readonly mode = this.route.snapshot.data['mode'] === 'edit' ? 'edit' : 'create';
  fields: MonitorField[] = [];
  private currentType: MonitorType = 'http';

  ngOnInit(): void {
    this.api.get<{ data?: TagModel[] }>('tags', undefined).pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (res) => this.tags.set(res.data ?? []) });
    this.api.get<{ data?: ProxyModel[] }>('proxies').pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (res) => this.proxies.set(res.data ?? []) });
    this.api.get<{ data?: NotificationChannelModel[] }>('notification-channels').pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: (res) => this.channels.set(res.data ?? []) });

    if (this.mode === 'edit') {
      const id = this.route.snapshot.paramMap.get('id');
      if (id) {
        this.busy.set(true);
        this.api.get<{ data: MonitorModel }>(`monitors/${encodeURIComponent(id)}`)
          .pipe(takeUntilDestroyed(this.destroyRef))
          .subscribe({
            next: (response) => {
              this.monitor.set(response.data);
              this.setValues(response.data);
              this.busy.set(false);
            },
            error: () => {
              this.error.set('Unable to load this monitor.');
              this.busy.set(false);
            },
          });
      }
    } else {
      const cloneData = this.locationState.cloneData;
      this.setValues(cloneData ?? undefined);
      if (cloneData) {
        this.form.controls['active'].setValue(true);
        if (cloneData.name) this.form.controls['name'].setValue(`${cloneData.name} Copy`);
      }
    }
  }

  getControl(key: string): FormControl<unknown> {
    return this.form.controls[this.fields.some((field) => field.key === key) ? `config:${key}` : key];
  }

  changeType(value: string): void {
    if (!MONITOR_TYPES.includes(value as MonitorType) || value === this.currentType) return;
    const current = this.form.getRawValue();
    const type = value as MonitorType;
    this.currentType = type;
    this.fields = MONITOR_FIELDS[type];
    this.form.controls['type'].setValue(type);
    for (const key of Object.keys(this.form.controls)) {
      if (key.startsWith('config:')) this.form.removeControl(key);
    }
    for (const field of this.fields) {
      const previous = current[field.key];
      this.form.addControl(`config:${field.key}`, new FormControl<unknown>(
        previous ?? field.defaultValue ?? (field.type === 'checkbox' ? false : ''),
        { validators: this.validatorsFor(field) },
      ));
    }
  }

  onTypeChange(type: string): void {
    const old = this.form.getRawValue();
    const preserve: Record<string, unknown> = {};
    for (const key of Object.keys(DEFAULTS)) preserve[key] = old[key];
    preserve['type'] = type;
    this.changeType(type);
    for (const key of Object.keys(preserve)) {
      if (this.form.controls[key]) this.form.controls[key].setValue(preserve[key]);
    }
    if (type === 'push') this.generatePushToken();
  }

  generatePushToken(): void {
    const alphabet = '0123456789abcdefghijklmnopqrstuvwxyz';
    const random = new Uint8Array(24);
    globalThis.crypto.getRandomValues(random);
    this.getControl('pushToken').setValue(Array.from(random, (value) => alphabet[value % alphabet.length]).join(''));
  }

  addSelection(controlName: 'notification_ids' | 'tag_ids', id: string, checked: boolean): void {
    const selected = (this.getControl(controlName).value as string[] | null) ?? [];
    this.getControl(controlName).setValue(checked
      ? [...new Set([...selected, id])]
      : selected.filter((value) => value !== id));
  }

  isSelected(controlName: 'notification_ids' | 'tag_ids', id: string): boolean {
    return ((this.getControl(controlName).value as string[] | null) ?? []).includes(id);
  }

  currentTypeLabel(): string {
    return (this.getControl('type').value as string ?? 'http').replaceAll('-', ' ');
  }

  hasProxy(): boolean {
    return ['http', 'http-keyword', 'http-json-query', 'docker'].includes(String(this.getControl('type').value));
  }

  removeSelection(controlName: 'notification_ids' | 'tag_ids', id: string): void {
    this.addSelection(controlName, id, false);
  }

  private validatorsFor(field: MonitorField): ValidatorFn[] {
    const validators: ValidatorFn[] = [];
    if (field.required) validators.push(Validators.required);
    if (field.min !== undefined) validators.push(Validators.min(field.min));
    if (field.max !== undefined) validators.push(Validators.max(field.max));
    if (field.key === 'url') validators.push(Validators.pattern(/^https?:\/\/.+/i));
    if (field.key === 'browser_ws_endpoint') validators.push(Validators.pattern(/^wss?:\/\/.+/i));
    if (field.key === 'connection_string') {
      validators.push(Validators.pattern(/^mysql:\/\/([^:@]+)(?::([^@]*))?@([^:/]+)(?::(\d+))?\/([^?]+)(?:\?.*)?$/i));
    }
    if (field.key === 'database_connection_string') {
      if (this.currentType === 'postgres') validators.push(Validators.pattern(/^postgres(?:ql)?:\/\/[^:@\s]+(?::[^@\s]*)?@[^:\s]+(?::\d+)?\/[\w-]+(?:\?.*)?$/i));
      if (this.currentType === 'sqlserver') validators.push(Validators.pattern(/^Server=[^;,]+(?:,\d+)?;Database=[^;]+;User Id=[^;]+;.+/i));
    }
    if (field.key === 'connectionString') {
      validators.push(Validators.pattern(/^mongodb(?:\+srv)?:\/\/(?:[^:@/]+(?::[^@/]*)?@)?[^:/@]+(?::\d+)?\/[^?]+(?:\?.*)?$/i));
    }
    if (field.key === 'databaseConnectionString') {
      validators.push(Validators.pattern(/^rediss?:\/\/(?:[^@]*@)?(?:\[[^\]]+\]|[^:/]+)(?::\d{1,5})?(?:\/[0-9]*)?$/i));
    }
    if (field.key === 'headers' || field.key === 'command') validators.push(this.validJson(false));
    if (field.key === 'conditions') validators.push(this.validJson(true));
    if (field.key === 'accepted_statuscodes') validators.push(Validators.pattern(/^(?:2XX|3XX|4XX|5XX)(?:\s*,\s*(?:2XX|3XX|4XX|5XX))*$/));
    return validators;
  }

  private validJson(arrayOnly: boolean): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
      if (!control.value) return null;
      try {
        const parsed: unknown = JSON.parse(String(control.value));
        if (arrayOnly && !Array.isArray(parsed)) return { json: true };
        return null;
      } catch {
        return { json: true };
      }
    };
  }

  async submit(): Promise<void> {
    this.error.set('');
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.error.set('Check the highlighted fields and try again.');
      return;
    }
    const raw = this.form.getRawValue();
    const values: Record<string, unknown> = { ...raw };
    for (const field of this.fields) values[field.key] = raw[`config:${field.key}`];
    const payload = serializeMonitor(values, Boolean(raw['active']));
    this.busy.set(true);
    const id = this.monitor()?.id;
    const request = this.mode === 'edit' && id
      ? this.api.put<{ data?: MonitorModel }>(`monitors/${encodeURIComponent(id)}`, payload)
      : this.api.post<{ data?: MonitorModel }>('monitors', payload);
    request.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (response) => {
        const monitorId = response.data?.id ?? id;
        if (monitorId) void this.router.navigate(['/monitors', monitorId]);
        else void this.router.navigate(['/monitors']);
        this.busy.set(false);
      },
      error: (response: { error?: { message?: string } }) => {
        this.error.set(response.error?.message ?? 'Unable to save this monitor.');
        this.busy.set(false);
      },
    });
  }

  private setValues(monitor?: MonitorModel): void {
    const type = monitor?.type && MONITOR_TYPES.includes(monitor.type) ? monitor.type : 'http';
    const config = monitor ? parseMonitorConfig(monitor) : {};
    const values: Record<string, unknown> = {
      ...DEFAULTS,
      type,
      ...(monitor ? {
        name: monitor.name ?? DEFAULTS['name'],
        interval: monitor.interval ?? DEFAULTS['interval'],
        timeout: monitor.timeout ?? DEFAULTS['timeout'],
        max_retries: monitor.max_retries ?? DEFAULTS['max_retries'],
        retry_interval: monitor.retry_interval ?? DEFAULTS['retry_interval'],
        resend_interval: monitor.resend_interval ?? DEFAULTS['resend_interval'],
        notification_ids: monitor.notification_ids ?? [],
        tag_ids: monitor.tag_ids ?? [],
        proxy_id: monitor.proxy_id ?? '',
        active: monitor.active ?? true,
      } : {}),
    };
    if (monitor?.type === 'push') values['pushToken'] = config['pushToken'] ?? monitor.push_token ?? '';
    for (const field of MONITOR_FIELDS[type]) {
      let value = config[field.key] ?? field.defaultValue ?? (field.type === 'checkbox' ? false : '');
      if (field.format === 'csv' && Array.isArray(value)) value = value.join(', ');
      if (field.format === 'json' && typeof value !== 'string') value = JSON.stringify(value ?? [], null, 2);
      if (field.format === 'lines' && Array.isArray(value)) value = value.join('\n');
      values[field.key] = value;
    }
    if (monitor) {
      values['name'] = monitor.name ?? 'My monitor';
      values['type'] = type;
      values['notification_ids'] = monitor.notification_ids ?? [];
      values['tag_ids'] = monitor.tag_ids ?? [];
      values['proxy_id'] = monitor.proxy_id ?? '';
      values['active'] = monitor.active ?? true;
      values['interval'] = monitor.interval ?? 60;
      values['timeout'] = monitor.timeout ?? 16;
      values['max_retries'] = monitor.max_retries ?? 3;
      values['retry_interval'] = monitor.retry_interval ?? 60;
      values['resend_interval'] = monitor.resend_interval ?? 10;
    }
    this.currentType = type;
    this.fields = MONITOR_FIELDS[type];
    for (const key of Object.keys(this.form.controls)) this.form.removeControl(key);
    for (const [key, value] of Object.entries(values)) this.form.addControl(key, new FormControl<unknown>(value));
    for (const field of this.fields) {
      this.form.removeControl(field.key);
      this.form.addControl(`config:${field.key}`, new FormControl<unknown>(values[field.key], {
        validators: this.validatorsFor(field),
      }));
    }
    this.form.controls['name'].setValidators([Validators.required]);
    this.form.controls['interval'].setValidators([Validators.required, Validators.min(20)]);
    this.form.controls['timeout'].setValidators([Validators.required, Validators.min(16)]);
    this.form.controls['max_retries'].setValidators([Validators.required, Validators.min(0)]);
    this.form.controls['retry_interval'].setValidators([Validators.required, Validators.min(20)]);
    this.form.controls['resend_interval'].setValidators([Validators.required, Validators.min(0)]);
    for (const key of ['name', 'interval', 'timeout', 'max_retries', 'retry_interval', 'resend_interval']) {
      this.form.controls[key].updateValueAndValidity({ emitEvent: false });
    }
  }
}
