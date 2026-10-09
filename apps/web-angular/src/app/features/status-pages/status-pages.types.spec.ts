import { heartbeatClass, overallStatus, statusPagePayload } from './status-pages.types';

describe('status page feature helpers', () => {
  it('converts the selected monitors into the API monitor_ids field', () => {
    expect(statusPagePayload({
      title: 'Example', slug: 'example', description: '', icon: '', footer_text: '',
      auto_refresh_interval: 300, published: true, domains: ['status.example.com'],
      monitors: [{ value: 'monitor-1', label: 'API' }, { value: 'monitor-2', label: 'Web' }],
    })).toEqual({
      title: 'Example', slug: 'example', description: '', icon: '', footer_text: '',
      auto_refresh_interval: 300, published: true, domains: ['status.example.com'],
      monitor_ids: ['monitor-1', 'monitor-2'],
    });
  });

  it('prioritizes down monitors over maintenance and defaults to operational', () => {
    expect(overallStatus([])).toBe(1);
    expect(overallStatus([{ type: 'http', heartbeats: [{ status: 3 }] }])).toBe(3);
    expect(overallStatus([
      { type: 'http', heartbeats: [{ status: 3 }] },
      { type: 'http', heartbeats: [{ status: 2 }] },
    ])).toBe(0);
  });

  it('maps heartbeat states to accessible chart classes', () => {
    expect([heartbeatClass(1), heartbeatClass(0), heartbeatClass(3), heartbeatClass(undefined)])
      .toEqual(['up', 'down', 'maintenance', 'unknown']);
  });
});
