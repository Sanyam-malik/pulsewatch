import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { RegisterComponent } from './register.component';

@Component({ standalone: true, template: '' })
class TestTargetComponent {}

describe('RegisterComponent', () => {
  let http: HttpTestingController;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [RegisterComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([{ path: '', component: TestTargetComponent }]),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('requires a valid email, an eight-character password, and matching confirmation', () => {
    const component = TestBed.createComponent(RegisterComponent).componentInstance;
    component.form.setValue({
      email: 'not-an-email',
      password: 'short',
      confirmPassword: 'different',
    });

    component.submit();

    expect(component.form.invalid).toBe(true);
    http.expectNone('/api/v1/auth/register');
  });

  it('sends only the API credentials and persists returned auth state', () => {
    const component = TestBed.createComponent(RegisterComponent).componentInstance;
    component.form.setValue({
      email: 'person@example.com',
      password: 'long-enough',
      confirmPassword: 'long-enough',
    });
    component.submit();

    const request = http.expectOne('/api/v1/auth/register');
    expect(request.request.body).toEqual({
      email: 'person@example.com',
      password: 'long-enough',
    });
    request.flush({
      data: {
        accessToken: 'access',
        refreshToken: 'refresh',
        user: { id: 'user-1', email: 'person@example.com' },
      },
    });

    expect(TestBed.inject(AuthService).accessToken()).toBe('access');
    expect(TestBed.inject(AuthService).user()?.email).toBe('person@example.com');
  });
});
