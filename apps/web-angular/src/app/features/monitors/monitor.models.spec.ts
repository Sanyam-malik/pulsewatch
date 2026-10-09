import { describe, expect, it } from 'vitest';
import { MONITOR_FIELDS, MONITOR_TYPES, serializeMonitor } from './monitor.models';
import { MONITOR_ROUTES } from './monitor.routes';

describe('monitor feature', () => {
  it('registers a schema for every React monitor type and keeps monitor URLs', () => {
    expect(Object.keys(MONITOR_FIELDS).sort()).toEqual([...MONITOR_TYPES].sort());
    expect(MONITOR_ROUTES[0].path).toBe('monitors');
    expect(MONITOR_ROUTES[0].children?.map((route) => route.path)).toEqual(['', 'new', ':id/edit', ':id']);
  });

  it('serializes common fields and HTTP executor configuration to the API shape', () => {
    const payload = serializeMonitor({
      type: 'http',
      name: 'Frontend',
      interval: 30,
      timeout: 16,
      max_retries: 2,
      retry_interval: 20,
      resend_interval: 0,
      notification_ids: ['channel-a'],
      tag_ids: ['tag-a'],
      proxy_id: 'proxy-a',
      'url': 'https://example.test',
      'accepted_statuscodes': '2XX, 3XX',
      'method': 'GET',
      'encoding': 'json',
      'headers': '{}',
      'body': '',
      'max_redirects': 5,
      'ignore_tls_errors': false,
      'check_cert_expiry': true,
      'authMethod': 'none',
    }, true);
    expect(payload).toMatchObject({
      type: 'http', name: 'Frontend', interval: 30, active: true,
      notification_ids: ['channel-a'], tag_ids: ['tag-a'], proxy_id: 'proxy-a',
    });
    expect(JSON.parse(String(payload['config']))).toMatchObject({
      url: 'https://example.test', accepted_statuscodes: ['2XX', '3XX'], check_cert_expiry: true,
    });
  });

  it('converts collection and nested executor fields for RabbitMQ and Kafka', () => {
    const rabbit = serializeMonitor({
      type: 'rabbitmq', name: 'Rabbit', interval: 60, timeout: 16, max_retries: 3,
      retry_interval: 60, resend_interval: 10, notification_ids: [], tag_ids: [],
      nodes: 'amqp://one:5672\namqp://two:5672', username: 'guest', password: 'guest',
    }, true);
    expect(JSON.parse(String(rabbit['config'])).nodes).toEqual([
      { url: 'amqp://one:5672' }, { url: 'amqp://two:5672' },
    ]);

    const kafka = serializeMonitor({
      type: 'kafka-producer', name: 'Kafka', interval: 60, timeout: 16, max_retries: 3,
      retry_interval: 60, resend_interval: 10, notification_ids: [], tag_ids: [],
      brokers: 'kafka:9092', topic: 'health', message: '{}', allow_auto_topic_creation: false,
      ssl: true, 'sasl_options.mechanism': 'PLAIN', 'sasl_options.username': 'user',
      'sasl_options.password': 'secret',
    }, true);
    expect(JSON.parse(String(kafka['config']))).toMatchObject({
      brokers: ['kafka:9092'],
      sasl_options: { mechanism: 'PLAIN', username: 'user', password: 'secret' },
    });
  });
});
