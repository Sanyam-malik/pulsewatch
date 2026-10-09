export type ChannelValue = string | number | boolean;
export type ChannelFieldKind = 'text' | 'password' | 'url' | 'email' | 'number' | 'textarea' | 'checkbox' | 'select';

export interface ChannelField {
  key: string;
  label: string;
  kind?: ChannelFieldKind;
  required?: boolean;
  min?: number;
  max?: number;
  options?: readonly { label: string; value: string }[];
  showWhen?: { key: string; value: ChannelValue };
  validate?: 'json' | 'pagertree-url' | 'e164';
}

export interface ChannelProvider {
  type: string;
  label: string;
  defaults: Record<string, ChannelValue>;
  fields: readonly ChannelField[];
}

const choices = (...values: string[]): { label: string; value: string }[] =>
  values.map((value) => ({ label: value, value }));

const provider = (
  type: string,
  label: string,
  defaults: Record<string, ChannelValue>,
  fields: readonly ChannelField[],
): ChannelProvider => ({ type, label, defaults, fields });

export const NOTIFICATION_PROVIDERS: readonly ChannelProvider[] = [
  provider('smtp', 'Email (SMTP)', {
    smtp_secure: false, smtp_host: 'example.com', smtp_port: 587, username: 'username',
    password: 'password', from: 'sender@example.com', to: 'recipient@example.com',
    cc: 'cc@example.com', bcc: 'bcc@example.com', custom_subject: '{{ msg }}', custom_body: '{{ msg }}',
  }, [
    { key: 'smtp_secure', label: 'Use secure connection (TLS)', kind: 'checkbox' },
    { key: 'smtp_host', label: 'SMTP host', required: true },
    { key: 'smtp_port', label: 'SMTP port', kind: 'number', required: true, min: 1 },
    { key: 'username', label: 'Username', required: true },
    { key: 'password', label: 'Password', kind: 'password', required: true },
    { key: 'from', label: 'Sender email', kind: 'email', required: true },
    { key: 'to', label: 'Recipient(s)', required: true },
    { key: 'cc', label: 'CC' }, { key: 'bcc', label: 'BCC' },
    { key: 'custom_subject', label: 'Custom subject' }, { key: 'custom_body', label: 'Custom body', kind: 'textarea' },
  ]),
  provider('telegram', 'Telegram', {
    bot_token: '', chat_id: '', message_thread_id: '', server_url: 'https://api.telegram.org', use_template: false,
    template_parse_mode: 'plain', template: 'Pulsewatch Alert - {{ monitor.name }}\n\n{{ msg }}',
    send_silently: false, protect_content: false,
  }, [
    { key: 'bot_token', label: 'Bot token', kind: 'password', required: true }, { key: 'chat_id', label: 'Chat ID', required: true },
    { key: 'message_thread_id', label: 'Message thread ID' }, { key: 'server_url', label: 'Telegram API server URL', required: true },
    { key: 'use_template', label: 'Use custom template', kind: 'checkbox' },
    { key: 'template_parse_mode', label: 'Template parse mode', kind: 'select', options: choices('plain', 'HTML', 'MarkdownV2') },
    { key: 'template', label: 'Message template', kind: 'textarea' }, { key: 'send_silently', label: 'Send silently', kind: 'checkbox' },
    { key: 'protect_content', label: 'Protect content', kind: 'checkbox' },
  ]),
  provider('webhook', 'Webhook', {
    webhook_url: 'https://example.com/webhook', webhook_content_type: 'json',
    webhook_custom_body: '{\n  "Title": "Uptime Alert - {{ monitor.name }}",\n  "Body": "{{ msg }}"\n}', webhook_additional_headers: '',
  }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true },
    { key: 'webhook_content_type', label: 'Content type', kind: 'select', options: choices('json', 'form-data', 'custom') },
    { key: 'webhook_custom_body', label: 'Custom body', kind: 'textarea', required: true, showWhen: { key: 'webhook_content_type', value: 'custom' } },
    { key: 'webhook_additional_headers', label: 'Additional headers', kind: 'textarea' },
  ]),
  provider('slack', 'Slack', {
    slack_webhook_url: '', slack_username: '', slack_icon_emoji: '', slack_channel: '',
    slack_rich_message: false, slack_channel_notify: false,
  }, [
    { key: 'slack_webhook_url', label: 'Slack webhook URL', kind: 'url', required: true },
    { key: 'slack_username', label: 'Username' }, { key: 'slack_icon_emoji', label: 'Icon emoji' },
    { key: 'slack_channel', label: 'Channel' }, { key: 'slack_rich_message', label: 'Use rich messages', kind: 'checkbox' },
    { key: 'slack_channel_notify', label: 'Notify channel', kind: 'checkbox' },
  ]),
  provider('ntfy', 'NTFY', {
    server_url: 'https://ntfy.sh', topic: 'pulsewatch', authentication_type: 'none', username: '', password: '', token: '',
    priority: 3, tags: 'pulsewatch,monitoring', title: 'Pulsewatch Alert - {{ name }}', custom_message: '{{ msg }}',
  }, [
    { key: 'server_url', label: 'Server URL', kind: 'url', required: true }, { key: 'topic', label: 'Topic', required: true },
    { key: 'authentication_type', label: 'Authentication type', kind: 'select', options: choices('none', 'basic', 'token') },
    { key: 'username', label: 'Username', showWhen: { key: 'authentication_type', value: 'basic' } },
    { key: 'password', label: 'Password', kind: 'password', showWhen: { key: 'authentication_type', value: 'basic' } },
    { key: 'token', label: 'Access token', kind: 'password', showWhen: { key: 'authentication_type', value: 'token' } },
    { key: 'priority', label: 'Priority (1–5)', kind: 'number', required: true, min: 1, max: 5 },
    { key: 'tags', label: 'Tags' }, { key: 'title', label: 'Title' }, { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('pagerduty', 'PagerDuty', {
    pagerduty_integration_key: '', pagerduty_integration_url: 'https://events.pagerduty.com/v2/enqueue',
    pagerduty_priority: 'warning', pagerduty_auto_resolve: '0',
  }, [
    { key: 'pagerduty_integration_key', label: 'Integration key', kind: 'password', required: true },
    { key: 'pagerduty_integration_url', label: 'Integration URL', kind: 'url', required: true },
    { key: 'pagerduty_priority', label: 'Priority' }, { key: 'pagerduty_auto_resolve', label: 'Auto-resolve' },
  ]),
  provider('opsgenie', 'Opsgenie', { region: 'us', api_key: '', priority: 3 }, [
    { key: 'region', label: 'Region', kind: 'select', required: true, options: choices('us', 'eu') },
    { key: 'api_key', label: 'API key', kind: 'password', required: true },
    { key: 'priority', label: 'Priority (1–5)', kind: 'number', min: 1, max: 5 },
  ]),
  provider('google_chat', 'Google Chat', { webhook_url: '' }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true },
  ]),
  provider('grafana_oncall', 'Grafana OnCall', { grafana_oncall_url: '' }, [
    { key: 'grafana_oncall_url', label: 'Grafana OnCall URL', kind: 'url', required: true },
  ]),
  provider('signal', 'Signal', { signal_url: '', signal_number: '', signal_recipients: '', custom_message: '{{ msg }}' }, [
    { key: 'signal_url', label: 'Signal API URL', kind: 'url', required: true }, { key: 'signal_number', label: 'Phone number', required: true },
    { key: 'signal_recipients', label: 'Recipients', required: true }, { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('gotify', 'Gotify', { server_url: '', application_token: '', priority: 8, title: '', custom_message: '' }, [
    { key: 'server_url', label: 'Server URL', kind: 'url', required: true },
    { key: 'application_token', label: 'Application token', kind: 'password', required: true },
    { key: 'priority', label: 'Priority (0–10)', kind: 'number', min: 0, max: 10 },
    { key: 'title', label: 'Title' }, { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('pushover', 'Pushover', {
    pushover_user_key: '', pushover_app_token: '', pushover_device: '', pushover_title: '', pushover_priority: 0,
    pushover_sounds: 'pushover', pushover_sounds_up: 'pushover', pushover_ttl: 0,
  }, [
    { key: 'pushover_user_key', label: 'User key', kind: 'password', required: true },
    { key: 'pushover_app_token', label: 'Application token', kind: 'password', required: true },
    { key: 'pushover_device', label: 'Device' }, { key: 'pushover_title', label: 'Title' },
    { key: 'pushover_priority', label: 'Priority (−2–2)', kind: 'number', min: -2, max: 2 },
    { key: 'pushover_sounds', label: 'Sound' }, { key: 'pushover_sounds_up', label: 'Recovery sound' },
    { key: 'pushover_ttl', label: 'Time to live', kind: 'number', min: 0 },
  ]),
  provider('mattermost', 'Mattermost', {
    webhook_url: '', username: 'Pulsewatch', channel: '', icon_url: '', icon_emoji: '', use_template: false, template: '',
  }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'username', label: 'Username' },
    { key: 'channel', label: 'Channel' }, { key: 'icon_url', label: 'Icon URL', kind: 'url' }, { key: 'icon_emoji', label: 'Icon emoji' },
    { key: 'use_template', label: 'Use custom template', kind: 'checkbox' }, { key: 'template', label: 'Template', kind: 'textarea' },
  ]),
  provider('matrix', 'Matrix', { homeserver_url: '', internal_room_id: '', access_token: '', custom_message: '{{ msg }}' }, [
    { key: 'homeserver_url', label: 'Homeserver URL', kind: 'url', required: true },
    { key: 'internal_room_id', label: 'Internal room ID', required: true },
    { key: 'access_token', label: 'Access token', kind: 'password', required: true },
    { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('discord', 'Discord', {
    webhook_url: '', bot_display_name: 'Pulsewatch', custom_message_prefix: '', message_type: 'send_to_channel', thread_name: '', thread_id: '',
  }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'bot_display_name', label: 'Bot display name', required: true },
    { key: 'custom_message_prefix', label: 'Custom message prefix' },
    { key: 'message_type', label: 'Message type', kind: 'select', required: true, options: [
      { label: 'Send to channel', value: 'send_to_channel' }, { label: 'New forum post', value: 'send_to_new_forum_post' }, { label: 'Send to thread', value: 'send_to_thread' },
    ] },
    { key: 'thread_name', label: 'Thread name', required: true, showWhen: { key: 'message_type', value: 'send_to_new_forum_post' } },
    { key: 'thread_id', label: 'Thread ID', required: true, showWhen: { key: 'message_type', value: 'send_to_thread' } },
  ]),
  provider('wecom', 'WeCom (企业微信)', { webhook_url: '', custom_message: '{{ msg }}' }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('whatsapp', 'WhatsApp (WAHA)', {
    server_url: 'http://localhost:3000', api_key: '', phone_number: '', session: '', use_template: false,
    template: '🚨 Pulsewatch Alert\n\nMonitor: {{ monitor.name }}\nStatus: {{ status }}\nMessage: {{ msg }}\n\nTime: {{ heartbeat.created_at }}',
    custom_message: '',
  }, [
    { key: 'server_url', label: 'Server URL', kind: 'url', required: true }, { key: 'api_key', label: 'API key', kind: 'password' },
    { key: 'phone_number', label: 'Phone number', required: true }, { key: 'session', label: 'Session', required: true },
    { key: 'use_template', label: 'Use custom template', kind: 'checkbox' }, { key: 'template', label: 'Template', kind: 'textarea' },
    { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('twilio', 'Twilio', {
    twilio_account_sid: '', twilio_api_key: '', twilio_auth_token: '', twilio_from_number: '', twilio_to_number: '',
  }, [
    { key: 'twilio_account_sid', label: 'Account SID', required: true }, { key: 'twilio_api_key', label: 'API key' },
    { key: 'twilio_auth_token', label: 'Auth token', kind: 'password', required: true },
    { key: 'twilio_from_number', label: 'From number (E.164)', required: true, validate: 'e164' },
    { key: 'twilio_to_number', label: 'To number (E.164)', required: true, validate: 'e164' },
  ]),
  provider('sendgrid', 'SendGrid', {
    api_key: '', from_email: 'noreply@example.com', to_email: 'recipient@example.com', cc_email: '', bcc_email: '', subject: '{{ name }} - {{ status }}',
  }, [
    { key: 'api_key', label: 'API key', kind: 'password', required: true }, { key: 'from_email', label: 'Sender email', kind: 'email', required: true },
    { key: 'to_email', label: 'Recipient email(s)', required: true }, { key: 'cc_email', label: 'CC email(s)' },
    { key: 'bcc_email', label: 'BCC email(s)' }, { key: 'subject', label: 'Subject' },
  ]),
  provider('pushbullet', 'Pushbullet', {
    pushbullet_access_token: '', pushbullet_device_id: '', pushbullet_channel_tag: '',
    pushbullet_custom_template: '{{ msg }}\n\nMonitor: {{ name }}\nStatus: {{ status }}\nTime: {{ heartbeat.time }}',
  }, [
    { key: 'pushbullet_access_token', label: 'Access token', kind: 'password', required: true },
    { key: 'pushbullet_device_id', label: 'Device ID' }, { key: 'pushbullet_channel_tag', label: 'Channel tag' },
    { key: 'pushbullet_custom_template', label: 'Custom template', kind: 'textarea' },
  ]),
  provider('pagertree', 'PagerTree', { integrationUrl: '', urgency: 'medium', autoResolve: true, authToken: '' }, [
    { key: 'integrationUrl', label: 'Integration URL', kind: 'url', required: true, validate: 'pagertree-url' },
    { key: 'urgency', label: 'Urgency', kind: 'select', required: true, options: choices('silent', 'low', 'medium', 'high', 'critical') },
    { key: 'autoResolve', label: 'Auto-resolve alerts', kind: 'checkbox' }, { key: 'authToken', label: 'Authentication token', kind: 'password' },
  ]),
  provider('line', 'LINE messaging', {
    channel_access_token: '', user_id: '', template: 'Pulsewatch Alert - {{ monitor.name }}\n\nStatus: {{ status }}\n{{ msg }}',
  }, [
    { key: 'channel_access_token', label: 'Channel access token', kind: 'password', required: true },
    { key: 'user_id', label: 'User ID', required: true }, { key: 'template', label: 'Message template', kind: 'textarea' },
  ]),
  provider('microsoft_teams', 'Microsoft Teams', { webhook_url: '', title: '' }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'title', label: 'Title' },
  ]),
  provider('whatsapp_whapi', 'WhatsApp (Whapi)', { server_url: 'https://gate.whapi.cloud', api_key: '', recipient: '' }, [
    { key: 'server_url', label: 'Server URL', kind: 'url', required: true }, { key: 'api_key', label: 'API key', kind: 'password', required: true },
    { key: 'recipient', label: 'Recipient', required: true },
  ]),
  provider('callmebot', 'CallMeBot', { service: 'whatsapp', recipient: '', api_key: '', language: '', api_url: '' }, [
    { key: 'service', label: 'Service', kind: 'select', required: true, options: choices('whatsapp', 'telegram', 'facebook') },
    { key: 'recipient', label: 'Recipient' },
    { key: 'api_key', label: 'API key', kind: 'password', required: true }, { key: 'language', label: 'Language' },
    { key: 'api_url', label: 'API URL', kind: 'url' },
  ]),
  provider('aliyun_sms', 'Aliyun SMS', {
    access_key_id: '', access_key_secret: '', region_id: 'cn-hangzhou', sign_name: '', template_code: '',
    phone_numbers: '', template_param: '{}', endpoint: '',
  }, [
    { key: 'access_key_id', label: 'Access key ID', required: true }, { key: 'access_key_secret', label: 'Access key secret', kind: 'password', required: true },
    { key: 'region_id', label: 'Region ID' }, { key: 'sign_name', label: 'Sign name', required: true },
    { key: 'template_code', label: 'Template code', required: true }, { key: 'phone_numbers', label: 'Phone numbers', required: true },
    { key: 'template_param', label: 'Template parameters (JSON)', kind: 'textarea', validate: 'json' },
    { key: 'endpoint', label: 'Endpoint URL', kind: 'url' },
  ]),
  provider('dingding', 'DingDing (钉钉)', { webhook_url: '', secret: '', title: '' }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'secret', label: 'Secret', kind: 'password' },
    { key: 'title', label: 'Title' },
  ]),
  provider('clicksend_sms', 'ClickSend SMS', {
    username: '', api_key: '', recipient: '', sender_id: '', api_url: '', custom_message: '',
  }, [
    { key: 'username', label: 'Username', required: true }, { key: 'api_key', label: 'API key', kind: 'password', required: true },
    { key: 'recipient', label: 'Recipient', required: true }, { key: 'sender_id', label: 'Sender ID' },
    { key: 'api_url', label: 'API URL', kind: 'url' }, { key: 'custom_message', label: 'Custom message', kind: 'textarea' },
  ]),
  provider('rocket_chat', 'Rocket.Chat', { webhook_url: '', username: '', channel: '', icon_emoji: '' }, [
    { key: 'webhook_url', label: 'Webhook URL', kind: 'url', required: true }, { key: 'username', label: 'Username' },
    { key: 'channel', label: 'Channel' }, { key: 'icon_emoji', label: 'Icon emoji' },
  ]),
];

export function getNotificationProvider(type: string): ChannelProvider | undefined {
  return NOTIFICATION_PROVIDERS.find((item) => item.type === type);
}

export function createChannelPayload(
  name: string,
  type: string,
  config: Record<string, ChannelValue>,
  active: boolean,
  isDefault: boolean,
): { name: string; type: string; config: string; active: boolean; is_default: boolean } {
  const configValues = { ...config };
  delete configValues['name'];
  delete configValues['type'];
  return {
    name,
    type,
    config: JSON.stringify({ name, type, ...configValues }),
    active,
    is_default: isDefault,
  };
}

export function createTestChannelPayload(
  name: string,
  type: string,
  config: Record<string, ChannelValue>,
): { name: string; type: string; config: string; active: boolean; is_default: boolean } {
  const configValues = { ...config };
  delete configValues['name'];
  delete configValues['type'];
  return {
    name,
    type,
    config: JSON.stringify({ type, ...configValues }),
    active: true,
    is_default: false,
  };
}

export function validateChannel(
  name: string,
  config: Record<string, ChannelValue>,
  providerInfo: ChannelProvider,
): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!name.trim()) errors['name'] = 'Name is required.';
  for (const field of providerInfo.fields) {
    const value = config[field.key] ?? '';
    if (field.showWhen && config[field.showWhen.key] !== field.showWhen.value) continue;
    if (field.required && (value === '' || (typeof value === 'string' && !value.trim()))) {
      errors[field.key] = 'This field is required.';
      continue;
    }
    if (typeof value === 'string' && value !== '') {
      if (field.kind === 'url' && !isValidUrl(value.trim())) errors[field.key] = 'Enter a valid URL.';
      if (field.kind === 'email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())) errors[field.key] = 'Enter a valid email address.';
      if (field.validate === 'e164' && !/^\+[1-9]\d{1,14}$/.test(value.trim())) errors[field.key] = 'Enter a valid E.164 phone number.';
      if (field.validate === 'pagertree-url' && !value.includes('api.pagertree.com/integration/')) errors[field.key] = 'Enter a PagerTree integration endpoint URL.';
      if (field.validate === 'json') {
        try { JSON.parse(value); } catch { errors[field.key] = 'Enter valid JSON.'; }
      }
    }
    if (field.kind === 'number' && value !== '' && typeof value === 'number') {
      if (!Number.isFinite(value)) errors[field.key] = 'Enter a valid number.';
      else if (field.min !== undefined && value < field.min) errors[field.key] = `Must be at least ${field.min}.`;
      else if (field.max !== undefined && value > field.max) errors[field.key] = `Must be at most ${field.max}.`;
    }
  }
  if (providerInfo.type === 'callmebot' && config['service'] !== 'facebook' && !String(config['recipient'] ?? '').trim()) {
    errors['recipient'] = 'Recipient is required for WhatsApp and Telegram.';
  }
  return errors;
}

function isValidUrl(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}
