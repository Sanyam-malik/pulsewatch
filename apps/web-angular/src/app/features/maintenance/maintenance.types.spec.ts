import { maintenancePayload, maintenanceValidation, MaintenanceFormValue } from './maintenance.types';

function formValue(strategy: MaintenanceFormValue['strategy']): MaintenanceFormValue {
  return {
    title: 'Database upgrade', description: 'Scheduled work', active: true, strategy,
    monitors: [{ id: 'monitor-1', name: 'Database', type: 'http' }],
    timezone: 'America/New_York', startDateTime: '2026-10-10T01:00', endDateTime: '2026-10-10T02:00',
    startTime: '01:00', endTime: '02:00', cron: '0 1 * * *', duration: 60,
    intervalDay: 7, weekdays: [1, 3], daysOfMonth: [1, 15],
  };
}

describe('maintenance feature helpers', () => {
  it('preserves the one-time API data shape and timezone', () => {
    expect(maintenancePayload(formValue('single'))).toEqual({
      title: 'Database upgrade', description: 'Scheduled work', active: true, strategy: 'single',
      monitor_ids: ['monitor-1'], timezone: 'America/New_York',
      start_date_time: '2026-10-10T01:00', end_date_time: '2026-10-10T02:00',
    });
  });

  it('serializes recurring schedule fields by strategy', () => {
    expect(maintenancePayload(formValue('recurring-weekday'))).toMatchObject({
      strategy: 'recurring-weekday', weekdays: [1, 3], start_time: '01:00', end_time: '02:00',
    });
    expect(maintenancePayload(formValue('recurring-interval'))).toMatchObject({ interval_day: 7 });
    expect(maintenancePayload(formValue('recurring-day-of-month'))).toMatchObject({ days_of_month: [1, 15] });
    expect(maintenancePayload(formValue('cron'))).toMatchObject({ cron: '0 1 * * *', duration: 60 });
  });

  it('validates schedule bounds and requires recurring selectors', () => {
    const value = formValue('single');
    expect(maintenanceValidation(value)).toBeNull();
    expect(maintenanceValidation({ ...value, endDateTime: value.startDateTime })).toContain('start date');
    expect(maintenanceValidation({ ...value, strategy: 'recurring-weekday', weekdays: [] })).toContain('weekday');
    expect(maintenanceValidation({ ...value, strategy: 'recurring-interval', startTime: '02:00' })).toContain('start time');
  });

  it('keeps manual maintenance free of schedule properties', () => {
    expect(maintenancePayload(formValue('manual'))).toEqual({
      title: 'Database upgrade', description: 'Scheduled work', active: true, strategy: 'manual',
      monitor_ids: ['monitor-1'],
    });
  });
});
