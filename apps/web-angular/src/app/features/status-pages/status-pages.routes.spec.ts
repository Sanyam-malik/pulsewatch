import { MAINTENANCE_ROUTES } from '../maintenance/maintenance.routes';
import { STATUS_PAGE_ROUTES } from './status-pages.routes';

describe('feature route exports', () => {
  it('exports status management, editor, and public slug routes independently', () => {
    expect(STATUS_PAGE_ROUTES.map((route) => route.path)).toEqual([
      'status-pages', 'status-pages/new', 'status-pages/:id/edit', 'status/:slug',
    ]);
  });

  it('exports maintenance list, create, and edit routes independently', () => {
    expect(MAINTENANCE_ROUTES.map((route) => route.path)).toEqual([
      'maintenances', 'maintenances/new', 'maintenances/:id/edit',
    ]);
  });
});
