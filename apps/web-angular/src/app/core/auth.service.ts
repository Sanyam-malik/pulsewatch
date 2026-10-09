import { HttpClient, HttpContext, HttpContextToken } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { catchError, finalize, map, Observable, shareReplay, tap, throwError } from 'rxjs';
import { API_URL } from './runtime-config';

export const SKIP_AUTH = new HttpContextToken(() => false);

export interface AuthUser {
  id?: string;
  email?: string;
  groupId?: string;
  role?: 'owner' | 'admin' | 'member' | 'viewer';
  twofa_status?: boolean;
  [key: string]: unknown;
}

interface PersistedAuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  activeGroupID: string | null;
}

interface PersistedAuthEnvelope {
  state: PersistedAuthState;
  version: number;
}

interface ApiEnvelope<T> {
  data?: T;
}

interface RefreshedTokens {
  accessToken: string;
  refreshToken: string;
}

const STORAGE_KEY = 'auth-storage';
const EMPTY_STATE: PersistedAuthState = {
  accessToken: null,
  refreshToken: null,
  user: null,
  activeGroupID: null,
};

function readPersistedState(): PersistedAuthState {
  if (typeof localStorage === 'undefined') {
    return EMPTY_STATE;
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return EMPTY_STATE;
    }
    const parsed = JSON.parse(raw) as Partial<PersistedAuthEnvelope>;
    return { ...EMPTY_STATE, ...parsed.state };
  } catch (error) {
    console.error('Failed to read saved authentication state', error);
    return EMPTY_STATE;
  }
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly apiUrl = inject(API_URL);
  private readonly state = signal<PersistedAuthState>(readPersistedState());
  private refreshRequest$: Observable<string> | null = null;

  readonly accessToken = computed(() => this.state().accessToken);
  readonly refreshToken = computed(() => this.state().refreshToken);
  readonly user = computed(() => this.state().user);
  readonly activeGroupID = computed(() => this.state().activeGroupID);
  readonly isAuthenticated = computed(() => !!this.state().accessToken);

  setTokens(accessToken: string, refreshToken: string): void {
    this.update({ accessToken, refreshToken });
  }

  setUser(user: AuthUser | null): void {
    this.update({ user });
  }

  setActiveGroupID(activeGroupID: string | null): void {
    this.update({ activeGroupID });
  }

  clear(): void {
    this.state.set(EMPTY_STATE);
    this.persist();
  }

  refreshAccessToken(): Observable<string> {
    if (this.refreshRequest$) {
      return this.refreshRequest$;
    }
    const refreshToken = this.refreshToken();
    if (!refreshToken) {
      return throwError(() => new Error('No refresh token is available'));
    }

    this.refreshRequest$ = this.http
      .post<ApiEnvelope<RefreshedTokens>>(
        `${this.apiUrl}/api/v1/auth/refresh`,
        { refreshToken },
        { context: new HttpContext().set(SKIP_AUTH, true) },
      )
      .pipe(
        map((response) => {
          const accessToken = response.data?.accessToken;
          const nextRefreshToken = response.data?.refreshToken;
          if (!accessToken || !nextRefreshToken) {
            throw new Error('The refresh response did not include both tokens');
          }
          return { accessToken, refreshToken: nextRefreshToken };
        }),
        tap((tokens) => this.setTokens(tokens.accessToken, tokens.refreshToken)),
        map((tokens) => tokens.accessToken),
        catchError((error: unknown) => {
          this.clear();
          return throwError(() => error);
        }),
        finalize(() => {
          this.refreshRequest$ = null;
        }),
        shareReplay({ bufferSize: 1, refCount: false }),
      );
    return this.refreshRequest$;
  }

  private update(changes: Partial<PersistedAuthState>): void {
    this.state.update((state) => ({ ...state, ...changes }));
    this.persist();
  }

  private persist(): void {
    if (typeof localStorage !== 'undefined') {
      const envelope: PersistedAuthEnvelope = { state: this.state(), version: 0 };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(envelope));
    }
  }
}
