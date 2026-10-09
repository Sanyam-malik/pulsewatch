export interface ApiEnvelope<T> {
  data: T;
  message?: string;
}

export interface StatusPage {
  id?: string;
  title?: string;
  slug?: string;
  description?: string;
  icon?: string;
  footer_text?: string;
  auto_refresh_interval?: number;
  published?: boolean;
  domains?: string[];
  monitor_ids?: string[];
}

export interface Monitor {
  id?: string;
  name?: string;
  type?: string;
}

export interface Heartbeat {
  id?: string;
  status?: number;
  time?: string;
  end_time?: string;
  ping?: number;
}

export interface PublicMonitor extends Monitor {
  heartbeats?: Heartbeat[];
  uptime_24h?: number;
}

export interface PublicIncident {
  id: string;
  title: string;
  status: string;
  created_at: string;
  updated_at: string;
  updates: Array<{ id: string; status: string; message: string; created_at: string }>;
}

export interface StatusPageFormValue {
  title: string;
  slug: string;
  description: string;
  icon: string;
  footer_text: string;
  auto_refresh_interval: number;
  published: boolean;
  domains: string[];
  monitors: Array<{ value: string; label: string }>;
}

export function statusPagePayload(value: StatusPageFormValue): Omit<StatusPageFormValue, 'monitors'> & {
  monitor_ids: string[];
} {
  const { monitors, ...rest } = value;
  return { ...rest, monitor_ids: monitors.map((monitor) => monitor.value) };
}

export function overallStatus(monitors: PublicMonitor[]): number {
  if (monitors.some((monitor) => [0, 2].includes(monitor.heartbeats?.at(-1)?.status ?? -1))) return 0;
  if (monitors.some((monitor) => monitor.heartbeats?.at(-1)?.status === 3)) return 3;
  return 1;
}

export function heartbeatClass(status?: number): string {
  switch (status) {
    case 1: return 'up';
    case 0:
    case 2: return 'down';
    case 3: return 'maintenance';
    default: return 'unknown';
  }
}
