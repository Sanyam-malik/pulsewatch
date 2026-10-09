export const MONITOR_TYPES = [
  'http', 'http-keyword', 'http-json-query', 'tcp', 'ping', 'dns', 'push',
  'docker', 'grpc-keyword', 'snmp', 'mysql', 'postgres', 'sqlserver',
  'mongodb', 'redis', 'mqtt', 'rabbitmq', 'kafka-producer', 'steam',
  'gamedig', 'playwright',
] as const;

export type MonitorType = (typeof MONITOR_TYPES)[number];
export type MonitorStatus = 'all' | 'up' | 'down' | 'maintenance';
export type MonitorFieldType = 'text' | 'number' | 'textarea' | 'select' | 'checkbox';

export interface MonitorField {
  key: string;
  label: string;
  type?: MonitorFieldType;
  defaultValue?: string | number | boolean;
  required?: boolean;
  min?: number;
  max?: number;
  placeholder?: string;
  options?: Array<{ value: string; label: string }>;
  format?: 'lines' | 'csv' | 'json';
  secret?: boolean;
  help?: string;
}

export interface MonitorModel {
  id?: string;
  name?: string;
  type?: MonitorType;
  config?: string;
  active?: boolean;
  status?: number;
  interval?: number;
  timeout?: number;
  max_retries?: number;
  retry_interval?: number;
  resend_interval?: number;
  notification_ids?: string[];
  tag_ids?: string[];
  proxy_id?: string;
  push_token?: string;
  [key: string]: unknown;
}

export interface MonitorEnvelope<T> {
  data: T;
  message?: string;
}

export interface TagModel {
  id?: string;
  name?: string;
  color?: string;
}

export interface ProxyModel {
  id?: string;
  protocol?: string;
  host?: string;
  port?: number;
}

export interface NotificationChannelModel {
  id?: string;
  name?: string;
}

export interface HeartbeatModel {
  id?: string;
  monitor_id?: string;
  status?: boolean | number;
  time?: string;
  end_time?: string;
  ping?: number;
  msg?: string;
  important?: boolean;
  notified?: boolean;
}

const select = (key: string, label: string, options: string[], defaultValue: string): MonitorField => ({
  key, label, type: 'select', defaultValue,
  options: options.map((value) => ({ value, label: value })),
});
const text = (key: string, label: string, defaultValue = '', extra: Partial<MonitorField> = {}): MonitorField =>
  ({ key, label, defaultValue, ...extra });
const number = (key: string, label: string, defaultValue: number, extra: Partial<MonitorField> = {}): MonitorField =>
  ({ key, label, type: 'number', defaultValue, ...extra });
const area = (key: string, label: string, defaultValue = '', extra: Partial<MonitorField> = {}): MonitorField =>
  ({ key, label, type: 'textarea', defaultValue, ...extra });
const check = (key: string, label: string, defaultValue = false): MonitorField =>
  ({ key, label, type: 'checkbox', defaultValue });

const httpFields: MonitorField[] = [
  text('url', 'URL', 'https://example.com', { required: true }),
  select('method', 'HTTP method', ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'], 'GET'),
  select('encoding', 'Body encoding', ['json', 'form', 'xml', 'text'], 'json'),
  area('headers', 'Request headers (JSON)', '{ "Content-Type": "application/json" }'),
  area('body', 'Request body'),
  text('accepted_statuscodes', 'Accepted status codes (comma-separated)', '2XX', { format: 'csv', required: true }),
  number('max_redirects', 'Maximum redirects', 10, { min: 0 }),
  check('ignore_tls_errors', 'Ignore TLS errors'),
  check('check_cert_expiry', 'Check certificate expiry'),
  select('authMethod', 'Authentication', ['none', 'basic', 'oauth2-cc', 'ntlm', 'mtls'], 'none'),
  text('basic_auth_user', 'Username'),
  text('basic_auth_pass', 'Password', '', { secret: true }),
  text('oauth_auth_method', 'OAuth client authentication', 'client_secret_basic', {
    type: 'select',
    options: ['client_secret_basic', 'client_secret_post'].map((value) => ({ value, label: value })),
  }),
  text('oauth_token_url', 'OAuth token URL'),
  text('oauth_client_id', 'OAuth client ID'),
  text('oauth_client_secret', 'OAuth client secret', '', { secret: true }),
  text('oauth_scopes', 'OAuth scopes'),
  text('authDomain', 'NTLM domain'),
  text('authWorkstation', 'NTLM workstation'),
  area('tlsCert', 'TLS client certificate'),
  area('tlsKey', 'TLS client key'),
  area('tlsCa', 'TLS CA certificate'),
];

export const MONITOR_FIELDS: Record<MonitorType, MonitorField[]> = {
  http: httpFields,
  'http-keyword': [...httpFields, text('keyword', 'Keyword', '', { required: true }), check('invert_keyword', 'Invert keyword match')],
  'http-json-query': [
    ...httpFields,
    select('condition_operator', 'Condition matching', ['and', 'or'], 'and'),
    area('conditions', 'JSON query conditions (JSON array)', '[]', { format: 'json' }),
  ],
  tcp: [text('host', 'Host', '', { required: true }), number('port', 'Port', 80, { required: true, min: 1, max: 65535 })],
  ping: [text('host', 'Host', '', { required: true }), number('packet_size', 'Packet size (bytes)', 32, { min: 0, max: 65507 })],
  dns: [
    text('host', 'Host', '', { required: true }), text('resolver_server', 'Resolver server', '8.8.8.8', { required: true }),
    number('port', 'Port', 53, { min: 1, max: 65535 }),
    select('resolve_type', 'Record type', ['A', 'AAAA', 'CAA', 'CNAME', 'MX', 'NS', 'PTR', 'SOA', 'SRV', 'TXT'], 'A'),
  ],
  push: [text('pushToken', 'Push token', '', { required: true, help: 'The push token is also sent in the push_token API field.' })],
  docker: [
    text('container_id', 'Container name or ID', '', { required: true }),
    select('connection_type', 'Connection type', ['socket', 'tcp'], 'socket'),
    text('docker_daemon', 'Docker daemon', 'unix:///var/run/docker.sock', { required: true }),
    check('tls_enabled', 'Enable TLS'), area('tls_cert', 'TLS certificate'), area('tls_key', 'TLS key'),
    area('tls_ca', 'TLS CA certificate'), check('tls_verify', 'Verify TLS certificate', true),
  ],
  'grpc-keyword': [
    text('grpcUrl', 'gRPC URL', '', { required: true }), area('grpcProtobuf', 'Proto content', '', { required: true }),
    text('grpcServiceName', 'Proto service name', '', { required: true }), text('grpcMethod', 'Proto method', '', { required: true }),
    check('grpcEnableTls', 'Enable TLS'), area('grpcBody', 'Request body'), text('keyword', 'Keyword'), check('invertKeyword', 'Invert keyword match'),
  ],
  snmp: [
    text('host', 'Host', '', { required: true }), number('port', 'Port', 161, { min: 1, max: 65535 }),
    text('community', 'Community', 'public', { required: true }), select('snmp_version', 'SNMP version', ['v1', 'v2c', 'v3'], 'v2c'),
    text('oid', 'OID', '', { required: true }), text('json_path', 'JSON path', '$'),
    select('json_path_operator', 'JSON comparison', ['eq', 'ne', 'lt', 'gt', 'le', 'ge'], 'eq'),
    text('expected_value', 'Expected value'),
  ],
  mysql: [text('connection_string', 'Connection string', '******localhost:3306/database', { required: true }), area('query', 'Query', 'SELECT 1')],
  postgres: [text('database_connection_string', 'Database connection string', '******localhost:5432/database', { required: true }), area('database_query', 'Query', 'SELECT 1')],
  sqlserver: [text('database_connection_string', 'Database connection string', '', { required: true }), area('database_query', 'Query', 'SELECT 1')],
  mongodb: [
    text('connectionString', 'Connection string', 'mongodb://localhost:27017/database', { required: true }),
    area('command', 'Command (JSON)', '{}'), text('jsonPath', 'JSON path'), text('expectedValue', 'Expected value'),
  ],
  redis: [
    text('databaseConnectionString', 'Database connection string', 'redis://localhost:6379', { required: true }),
    check('ignoreTls', 'Ignore TLS errors'), area('caCert', 'CA certificate'), area('clientCert', 'Client certificate'), area('clientKey', 'Client key'),
  ],
  mqtt: [
    text('hostname', 'Hostname', 'localhost', { required: true }), number('port', 'Port', 1883, { min: 1, max: 65535 }),
    text('topic', 'Topic', 'test/topic', { required: true }), text('username', 'Username'), text('password', 'Password', '', { secret: true }),
    select('check_type', 'Check type', ['keyword', 'json-query', 'none'], 'none'), text('success_keyword', 'Success keyword'),
    text('json_path', 'JSON path'), text('expected_value', 'Expected value'),
  ],
  rabbitmq: [
    area('nodes', 'Node URLs (one per line)', 'amqp://localhost:5672', { format: 'lines', required: true }),
    text('username', 'Username', 'guest', { required: true }), text('password', 'Password', 'guest', { required: true, secret: true }),
  ],
  'kafka-producer': [
    area('brokers', 'Brokers (one per line)', 'localhost:9092', { format: 'lines', required: true }),
    text('topic', 'Topic', 'test-topic', { required: true }),
    area('message', 'Message', '{"status":"up"}', { required: true }),
    check('allow_auto_topic_creation', 'Allow automatic topic creation'), check('ssl', 'Enable SSL'),
    select('sasl_options.mechanism', 'SASL mechanism', ['None', 'PLAIN', 'SCRAM-SHA-256', 'SCRAM-SHA-512'], 'None'),
    text('sasl_options.username', 'SASL username'), text('sasl_options.password', 'SASL password', '', { secret: true }),
  ],
  steam: [
    text('host', 'Host', 'example.com', { required: true }), number('port', 'Port', 27015, { min: 1, max: 65535 }),
    number('expected_app_id', 'Expected app ID', 0, { min: 0, max: 65535 }), text('expected_name', 'Expected server name'),
  ],
  gamedig: [
    text('game', 'Game', 'source', { required: true }), text('host', 'Host', 'example.com', { required: true }),
    number('port', 'Port', 27015, { min: 1, max: 65535 }), text('expected_name', 'Expected server name'),
    text('expected_map', 'Expected map'), number('min_players', 'Minimum players', 0, { min: 0 }),
    number('max_players', 'Maximum players', 0, { min: 0 }),
  ],
  playwright: [
    text('browser_ws_endpoint', 'Browser WebSocket endpoint', '', { required: true, placeholder: 'ws://localhost:3000' }),
    text('url', 'Page URL', 'https://example.com', { required: true }), text('selector', 'Element selector', 'body'), text('keyword', 'Keyword'),
  ],
};

export function parseMonitorConfig(monitor: MonitorModel): Record<string, unknown> {
  try {
    const config = monitor.config ? JSON.parse(monitor.config) as Record<string, unknown> : {};
    if (monitor.type === 'rabbitmq' && Array.isArray(config['nodes'])) {
      config['nodes'] = (config['nodes'] as unknown[]).map((node) =>
        typeof node === 'string' ? node : (node as { url?: string })?.url ?? '',
      ).join('\n');
    }
    if (monitor.type === 'kafka-producer' && config['sasl_options'] && typeof config['sasl_options'] === 'object') {
      const sasl = config['sasl_options'] as Record<string, unknown>;
      config['sasl_options.mechanism'] = sasl['mechanism'] ?? 'None';
      config['sasl_options.username'] = sasl['username'] ?? '';
      config['sasl_options.password'] = sasl['password'] ?? '';
    }
    return config;
  } catch {
    return {};
  }
}

function setNested(target: Record<string, unknown>, path: string, value: unknown): void {
  const keys = path.split('.');
  let current = target;
  keys.slice(0, -1).forEach((key) => {
    if (!current[key] || typeof current[key] !== 'object') current[key] = {};
    current = current[key] as Record<string, unknown>;
  });
  current[keys[keys.length - 1]] = value;
}

export function serializeMonitor(values: Record<string, unknown>, active: boolean): Record<string, unknown> {
  const type = values['type'] as MonitorType;
  const config: Record<string, unknown> = {};
  for (const field of MONITOR_FIELDS[type]) {
    const value = values[field.key];
    if (field.format === 'lines') {
      const lines = String(value ?? '').split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
      setNested(config, field.key, type === 'rabbitmq' ? lines : lines);
    } else if (field.format === 'csv') {
      setNested(config, field.key, String(value ?? '').split(',').map((part) => part.trim()).filter(Boolean));
    } else if (field.format === 'json') {
      try { setNested(config, field.key, JSON.parse(String(value || '[]')) as unknown); }
      catch { setNested(config, field.key, value); }
    } else {
      setNested(config, field.key, value);
    }
  }
  if (type === 'rabbitmq') config['nodes'] = (config['nodes'] as string[]).map((url) => ({ url }));
  if (type === 'kafka-producer') {
    config['sasl_options'] = {
      mechanism: values['sasl_options.mechanism'],
      username: values['sasl_options.username'] || '',
      password: values['sasl_options.password'] || '',
    };
    delete config['sasl_options.mechanism'];
    delete config['sasl_options.username'];
    delete config['sasl_options.password'];
  }
  if (type === 'http' || type === 'http-keyword' || type === 'http-json-query') {
    const method = config['authMethod'];
    const authKeys = [
      'basic_auth_user', 'basic_auth_pass', 'oauth_auth_method', 'oauth_token_url',
      'oauth_client_id', 'oauth_client_secret', 'oauth_scopes', 'authDomain',
      'authWorkstation', 'tlsCert', 'tlsKey', 'tlsCa',
    ];
    for (const key of authKeys) delete config[key];
    if (method === 'basic') {
      config['basic_auth_user'] = values['basic_auth_user'];
      config['basic_auth_pass'] = values['basic_auth_pass'];
    } else if (method === 'oauth2-cc') {
      for (const key of ['oauth_auth_method', 'oauth_token_url', 'oauth_client_id', 'oauth_client_secret', 'oauth_scopes']) {
        config[key] = values[key];
      }
    } else if (method === 'ntlm') {
      config['basic_auth_user'] = values['basic_auth_user'];
      config['basic_auth_pass'] = values['basic_auth_pass'];
      config['authDomain'] = values['authDomain'];
      config['authWorkstation'] = values['authWorkstation'];
    } else if (method === 'mtls') {
      config['tlsCert'] = values['tlsCert'];
      config['tlsKey'] = values['tlsKey'];
      config['tlsCa'] = values['tlsCa'];
    }
  }
  if (type === 'docker' && !config['tls_enabled']) {
    for (const key of ['tls_cert', 'tls_key', 'tls_ca', 'tls_verify']) delete config[key];
  }
  if (type === 'redis') {
    for (const key of ['caCert', 'clientCert', 'clientKey']) {
      if (!config[key]) delete config[key];
    }
  }
  const payload: Record<string, unknown> = {
    type,
    name: values['name'],
    interval: Number(values['interval']),
    max_retries: Number(values['max_retries']),
    retry_interval: Number(values['retry_interval']),
    resend_interval: Number(values['resend_interval']),
    timeout: Number(values['timeout']),
    notification_ids: values['notification_ids'] ?? [],
    tag_ids: values['tag_ids'] ?? [],
    config: JSON.stringify(config),
    active,
  };
  const proxyId = values['proxy_id'];
  if (proxyId && ['http', 'http-keyword', 'http-json-query', 'docker'].includes(type)) payload['proxy_id'] = proxyId;
  if (type === 'push') payload['push_token'] = values['pushToken'];
  return payload;
}
