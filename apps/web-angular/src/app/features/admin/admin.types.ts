export interface ApiEnvelope<T> {
  data: T;
}

export interface Group {
  id: string;
  name: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
}

export interface GroupMember {
  user: { id: string; email: string; active: boolean };
  role: Group['role'];
}

export interface Monitor {
  id: string;
  name: string;
  type: string;
}

export interface MonitorGroup {
  id: string;
  name: string;
  description: string;
  monitor_ids: string[];
}

export interface Tag {
  id: string;
  name: string;
  color: string;
  description?: string;
  created_at?: string;
}

export interface Proxy {
  id: string;
  protocol: string;
  host: string;
  port: number;
  auth: boolean;
  username?: string;
  password?: string;
}
