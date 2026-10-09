import { HttpErrorResponse } from '@angular/common/http';
import { Component, DestroyRef, OnInit, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../core/api.service';
import {
  ChannelField,
  ChannelValue,
  createChannelPayload,
  createTestChannelPayload,
  getNotificationProvider,
  NOTIFICATION_PROVIDERS,
  validateChannel,
} from './notification-channel-config';
import { NotificationChannel } from './notification-channels.component';

interface ApiEnvelope<T> {
  data: T;
  message?: string;
}

@Component({
  selector: 'app-notification-channel-editor',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './notification-channel-editor.component.html',
  styleUrl: './notification-channel-editor.component.scss',
})
export class NotificationChannelEditorComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly destroyRef = inject(DestroyRef);

  readonly providers = NOTIFICATION_PROVIDERS;
  type = 'smtp';
  channelId: string | null = null;
  name = '';
  config: Record<string, ChannelValue> = { ...NOTIFICATION_PROVIDERS[0].defaults };
  active = true;
  isDefault = false;
  loading = false;
  saving = false;
  testing = false;
  error = '';
  notice = '';
  fieldErrors: Record<string, string> = {};

  get providerInfo() {
    return getNotificationProvider(this.type);
  }

  get title(): string {
    return this.channelId ? 'Edit notification channel' : 'Create notification channel';
  }

  ngOnInit(): void {
    this.channelId = this.route.snapshot.paramMap.get('id');
    if (this.channelId) this.loadChannel(this.channelId);
  }

  selectProvider(type: string): void {
    const selected = getNotificationProvider(type);
    if (!selected) return;
    this.type = selected.type;
    this.config = { ...selected.defaults };
    this.fieldErrors = {};
    this.error = '';
    this.notice = '';
  }

  isFieldVisible(field: ChannelField): boolean {
    return !field.showWhen || this.config[field.showWhen.key] === field.showWhen.value;
  }

  fieldValue(key: string): string {
    const value = this.config[key];
    return typeof value === 'boolean' || value === undefined ? '' : String(value);
  }

  setFieldValue(field: ChannelField, event: Event): void {
    const target = event.target;
    if (!(target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement)) return;
    const rawValue = target instanceof HTMLInputElement && target.type === 'checkbox'
      ? target.checked
      : target.value;
    const value = field.kind === 'number' && typeof rawValue === 'string' && rawValue !== ''
      ? Number(rawValue)
      : rawValue;
    this.config = { ...this.config, [field.key]: value };
    delete this.fieldErrors[field.key];
    this.error = '';
    this.notice = '';
  }

  submit(): void {
    if (this.saving || this.loading) return;
    if (!this.validateForm()) return;
    const payload = createChannelPayload(this.name, this.type, this.config, this.active, this.isDefault);
    this.saving = true;
    const request = this.channelId
      ? this.api.put<ApiEnvelope<NotificationChannel>>(`notification-channels/${encodeURIComponent(this.channelId)}`, payload)
      : this.api.post<ApiEnvelope<NotificationChannel>>('notification-channels', payload);
    request.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: () => {
        this.saving = false;
        void this.router.navigateByUrl('/notification-channels');
      },
      error: (error: unknown) => {
        this.error = this.errorMessage(error, 'Unable to save notification channel.');
        this.saving = false;
      },
    });
  }

  testChannel(): void {
    if (this.testing || this.loading) return;
    if (!this.validateForm()) return;
    this.testing = true;
    this.notice = '';
    this.api.post<unknown>(
      'notification-channels/test',
      createTestChannelPayload(this.name, this.type, this.config),
    ).pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: () => {
        this.notice = 'Test notification sent successfully.';
        this.testing = false;
      },
      error: (error: unknown) => {
        this.error = this.errorMessage(error, 'Unable to send test notification.');
        this.testing = false;
      },
    });
  }

  private loadChannel(id: string): void {
    this.loading = true;
    this.api.get<ApiEnvelope<NotificationChannel>>(`notification-channels/${encodeURIComponent(id)}`)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          const channel = response?.data;
          if (!channel) {
            this.error = 'Notification channel was not found.';
            this.loading = false;
            return;
          }
          const type = channel.type ?? '';
          const providerInfo = getNotificationProvider(type);
          if (!providerInfo) {
            this.error = `Unsupported notification provider: ${type || 'unknown'}.`;
            this.loading = false;
            return;
          }
          this.type = type;
          this.name = channel.name ?? '';
          this.active = channel.active ?? true;
          this.isDefault = channel.is_default ?? false;
          let parsed: Record<string, unknown> = {};
          if (channel.config) {
            try {
              const value: unknown = JSON.parse(channel.config);
              if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                parsed = value as Record<string, unknown>;
              }
            } catch {
              this.error = 'Channel configuration could not be read; default values were loaded.';
            }
          }
          this.config = {
            ...providerInfo.defaults,
            ...parsed,
          } as Record<string, ChannelValue>;
          this.loading = false;
        },
        error: (error: unknown) => {
          this.error = this.errorMessage(error, 'Unable to load notification channel.');
          this.loading = false;
        },
      });
  }

  private validateForm(): boolean {
    const providerInfo = this.providerInfo;
    if (!providerInfo) {
      this.error = 'Select a supported notification provider.';
      return false;
    }
    this.fieldErrors = validateChannel(this.name, this.config, providerInfo);
    this.error = '';
    if (Object.keys(this.fieldErrors).length > 0) {
      this.error = 'Check the highlighted fields and try again.';
      return false;
    }
    return true;
  }

  private errorMessage(error: unknown, fallback: string): string {
    if (error instanceof HttpErrorResponse) {
      const body = error.error as { message?: unknown } | null;
      if (typeof body?.message === 'string' && body.message) return body.message;
    }
    return fallback;
  }
}
