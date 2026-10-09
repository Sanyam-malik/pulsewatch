import { Monitor } from '../status-pages/status-pages.types';

export type MaintenanceStrategy =
  | 'manual'
  | 'single'
  | 'cron'
  | 'recurring-interval'
  | 'recurring-weekday'
  | 'recurring-day-of-month';

export interface Maintenance {
  id?: string;
  title?: string;
  description?: string;
  active?: boolean;
  strategy?: MaintenanceStrategy | string;
  start_date_time?: string;
  end_date_time?: string;
  start_time?: string;
  end_time?: string;
  timezone?: string;
  cron?: string;
  duration?: number;
  interval_day?: number;
  weekdays?: number[];
  days_of_month?: number[];
  monitor_ids?: string[];
}

export interface MaintenanceFormValue {
  title: string;
  description: string;
  active: boolean;
  strategy: MaintenanceStrategy;
  monitors: Monitor[];
  timezone: string;
  startDateTime: string;
  endDateTime: string;
  startTime: string;
  endTime: string;
  cron: string;
  duration: number;
  intervalDay: number;
  weekdays: number[];
  daysOfMonth: Array<number | string>;
}

export interface MaintenancePayload {
  title: string;
  description: string;
  active: boolean;
  strategy: MaintenanceStrategy;
  monitor_ids: string[];
  timezone?: string;
  start_date_time?: string;
  end_date_time?: string;
  start_time?: string;
  end_time?: string;
  cron?: string;
  duration?: number;
  interval_day?: number;
  weekdays?: number[];
  days_of_month?: number[];
}

export function maintenancePayload(value: MaintenanceFormValue): MaintenancePayload {
  const payload: MaintenancePayload = {
    title: value.title,
    description: value.description,
    active: value.active,
    strategy: value.strategy,
    monitor_ids: value.monitors.map((monitor) => monitor.id).filter((id): id is string => !!id),
  };
  if (value.strategy !== 'manual') {
    payload.timezone = value.timezone;
    payload.start_date_time = value.startDateTime;
    payload.end_date_time = value.endDateTime;
  }
  switch (value.strategy) {
    case 'cron':
      payload.cron = value.cron;
      payload.duration = Number(value.duration);
      break;
    case 'recurring-interval':
      payload.interval_day = Number(value.intervalDay);
      payload.start_time = value.startTime;
      payload.end_time = value.endTime;
      break;
    case 'recurring-weekday':
      payload.weekdays = value.weekdays;
      payload.start_time = value.startTime;
      payload.end_time = value.endTime;
      break;
    case 'recurring-day-of-month':
      payload.days_of_month = value.daysOfMonth.map((day) =>
        typeof day === 'string' && day !== 'lastDay1' ? Number(day) : day === 'lastDay1' ? 0 : day,
      ).filter((day): day is number => typeof day === 'number');
      payload.start_time = value.startTime;
      payload.end_time = value.endTime;
      break;
  }
  return payload;
}

export function maintenanceValidation(value: MaintenanceFormValue): string | null {
  if (!value.title.trim()) return 'A title is required.';
  if (value.strategy === 'manual') return null;
  if (!value.startDateTime || !value.endDateTime) return 'Start and end dates are required.';
  if (new Date(value.startDateTime) >= new Date(value.endDateTime)) return 'The start date must be before the end date.';
  if (value.strategy === 'cron' && (!value.cron.trim() || value.duration < 1)) return 'Enter a cron expression and a duration of at least one minute.';
  if (value.strategy === 'recurring-interval' && (value.intervalDay < 1 || value.intervalDay > 3650)) return 'The recurring interval must be between 1 and 3650 days.';
  if (['recurring-interval', 'recurring-weekday', 'recurring-day-of-month'].includes(value.strategy)) {
    if (!value.startTime || !value.endTime) return 'Start and end times are required.';
    if (value.startTime >= value.endTime) return 'The start time must be before the end time.';
  }
  if (value.strategy === 'recurring-weekday' && !value.weekdays.length) return 'Select at least one weekday.';
  if (value.strategy === 'recurring-day-of-month' && !value.daysOfMonth.length) return 'Select at least one day of the month.';
  return null;
}
