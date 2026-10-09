import { AuthUser } from '../../core/auth.service';

export interface AuthResponse {
  accessToken?: string;
  refreshToken?: string;
  user?: AuthUser;
}

export interface AuthApiEnvelope<T> {
  data?: T;
}

export interface LoginCredentials {
  email: string;
  password: string;
  token?: string;
}

export interface RegisterCredentials {
  email: string;
  password: string;
}
