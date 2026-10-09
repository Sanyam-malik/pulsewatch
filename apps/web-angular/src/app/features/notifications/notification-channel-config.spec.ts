import '@angular/compiler';
import { describe, expect, it } from 'vitest';
import { NOTIFICATION_ROUTES } from './notification.routes';
import {
  createChannelPayload,
  createTestChannelPayload,
  getNotificationProvider,
  NOTIFICATION_PROVIDERS,
  validateChannel,
} from './notification-channel-config';

describe('notification channel configuration', () => {
  it('exposes all provider types implemented by the React channel form', () => {
    expect(NOTIFICATION_PROVIDERS.map((provider) => provider.type)).toEqual([
      'smtp', 'telegram', 'webhook', 'slack', 'ntfy', 'pagerduty', 'opsgenie',
      'google_chat', 'grafana_oncall', 'signal', 'gotify', 'pushover', 'mattermost',
      'matrix', 'discord', 'wecom', 'whatsapp', 'twilio', 'sendgrid', 'pushbullet',
      'pagertree', 'line', 'microsoft_teams', 'whatsapp_whapi', 'callmebot',
      'aliyun_sms', 'dingding', 'clicksend_sms', 'rocket_chat',
    ]);
  });

  it('provides provider-specific defaults and form fields after selection', () => {
    const telegram = getNotificationProvider('telegram');
    const smtp = getNotificationProvider('smtp');

    expect(telegram?.defaults['server_url']).toBe('https://api.telegram.org');
    expect(telegram?.fields.map((field) => field.key)).toContain('bot_token');
    expect(smtp?.fields.map((field) => field.key)).toContain('smtp_secure');
    expect(getNotificationProvider('unknown')).toBeUndefined();
  });

  it('validates provider settings including conditional recipient requirements', () => {
    const callmebot = getNotificationProvider('callmebot');
    expect(callmebot).toBeDefined();
    expect(validateChannel('Alerts', { ...callmebot!.defaults, api_key: 'secret' }, callmebot!)).toMatchObject({
      recipient: 'Recipient is required for WhatsApp and Telegram.',
    });
    expect(validateChannel('Alerts', {
      ...callmebot!.defaults, service: 'facebook', api_key: 'secret',
    }, callmebot!)).toEqual({});
  });

  it('serializes compatible create, edit, and test payloads', () => {
    const config = { name: 'Old name', type: 'smtp', bot_token: 'token', chat_id: '42' };
    expect(createChannelPayload('Alerts', 'telegram', config, false, true)).toEqual({
      name: 'Alerts',
      type: 'telegram',
      config: JSON.stringify({ name: 'Alerts', type: 'telegram', bot_token: 'token', chat_id: '42' }),
      active: false,
      is_default: true,
    });
    expect(createTestChannelPayload('Alerts', 'telegram', config)).toEqual({
      name: 'Alerts',
      type: 'telegram',
      config: JSON.stringify({ type: 'telegram', bot_token: 'token', chat_id: '42' }),
      active: true,
      is_default: false,
    });
  });

  it('exports routes for the legacy list, create, and edit URLs', () => {
    expect(NOTIFICATION_ROUTES.map((route) => route.path)).toEqual([
      'notification-channels',
      'notification-channels/new',
      'notification-channels/:id/edit',
    ]);
  });
});
