import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { AUTH_ROUTES } from './auth.routes';
import { LoginComponent } from './login.component';

@Component({ standalone: true, template: '' })
class TestTargetComponent {}

describe('LoginComponent', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([{ path: '', component: TestTargetComponent }]),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('exports separate login and registration routes', () => {
    expect(AUTH_ROUTES.map((route) => route.path)).toEqual(['login', 'register']);
  });

  it('blocks empty and invalid credentials', () => {
    const fixture = TestBed.createComponent(LoginComponent);
    const component = fixture.componentInstance;

    component.submit();

    expect(component.form.invalid).toBe(true);
    http.expectNone('/api/v1/auth/login');
  });

  it('retries with a two-factor token and persists returned auth state', async () => {
    const fixture = TestBed.createComponent(LoginComponent);
    const component = fixture.componentInstance;
    component.form.setValue({ email: 'person@example.com', password: 'secret' });
    component.submit();

    http.expectOne('/api/v1/auth/login').flush(
      { message: '2FA token required', data: null },
      { status: 401, statusText: 'Unauthorized' },
    );
    expect(component.showTwoFactor).toBe(true);

    component.verifyTwoFactor('123456');
    const retry = http.expectOne('/api/v1/auth/login');
    expect(retry.request.body).toEqual({
      email: 'person@example.com',
      password: 'secret',
      token: '123456',
    });
    retry.flush({
      data: {
        accessToken: 'access',
        refreshToken: 'refresh',
        user: { id: 'user-1', email: 'person@example.com' },
      },
    });

    expect(TestBed.inject(AuthService).accessToken()).toBe('access');
    expect(TestBed.inject(AuthService).refreshToken()).toBe('refresh');
    expect(TestBed.inject(AuthService).user()?.email).toBe('person@example.com');
    expect(component.showTwoFactor).toBe(false);
  });
});
