import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { ADMIN_ROUTES } from './admin.routes';
import { GroupsPage } from './groups.page';

describe('admin routes and group workspace actions', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [GroupsPage],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter(ADMIN_ROUTES)],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('exports all management paths including create and edit paths', () => {
    expect(ADMIN_ROUTES.map((route) => route.path)).toEqual([
      'groups', 'monitor-groups', 'incidents', 'proxies/new', 'proxies/:id/edit',
      'proxies', 'tags/new', 'tags/:id/edit', 'tags', 'security', 'settings',
    ]);
  });

  it('creates a workspace and switches the active workspace to it', () => {
    const fixture = TestBed.createComponent(GroupsPage);
    const component = fixture.componentInstance;
    http.expectOne('/api/v1/groups').flush({ data: [] });

    component.createForm.controls.name.setValue('Operations');
    component.createGroup();
    const createRequest = http.expectOne('/api/v1/groups');
    expect(createRequest.request.method).toBe('POST');
    expect(createRequest.request.body).toEqual({ name: 'Operations' });
    createRequest.flush({ data: { id: 'group-1', name: 'Operations', role: 'owner' } });

    http.expectOne('/api/v1/groups').flush({ data: [{ id: 'group-1', name: 'Operations', role: 'owner' }] });
    http.expectOne('/api/v1/groups/group-1/members').flush({ data: [] });
    expect(TestBed.inject(AuthService).activeGroupID()).toBe('group-1');
    expect(TestBed.inject(AuthService).user()?.role).toBe('owner');
    fixture.destroy();
  });
});
