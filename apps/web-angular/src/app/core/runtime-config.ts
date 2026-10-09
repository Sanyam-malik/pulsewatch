import { InjectionToken } from '@angular/core';

declare global {
  interface Window {
    __CONFIG__?: {
      API_URL?: string;
    };
  }
}

export const API_URL = new InjectionToken<string>('API_URL', {
  providedIn: 'root',
  factory: () => {
    if (typeof window === 'undefined') {
      return '';
    }
    return window.__CONFIG__?.API_URL?.replace(/\/+$/, '') ?? '';
  },
});
