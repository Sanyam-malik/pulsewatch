import {
  HttpContextToken,
  HttpErrorResponse,
  HttpInterceptorFn,
} from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService, SKIP_AUTH } from './auth.service';

const AUTH_RETRY = new HttpContextToken<boolean>(() => false);

export const authInterceptor: HttpInterceptorFn = (request, next) => {
  const auth = inject(AuthService);
  const token = auth.accessToken();
  const groupID = auth.activeGroupID();
  let outgoing = request;

  if (!request.context.get(SKIP_AUTH)) {
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    if (groupID) {
      headers['X-Group-ID'] = groupID;
    }
    outgoing = request.clone({ setHeaders: headers });
  }

  return next(outgoing).pipe(
    catchError((error: unknown) => {
      if (
        !(error instanceof HttpErrorResponse) ||
        error.status !== 401 ||
        request.context.get(SKIP_AUTH) ||
        request.context.get(AUTH_RETRY) ||
        !token
      ) {
        return throwError(() => error);
      }

      return auth.refreshAccessToken().pipe(
        switchMap((newToken) =>
          next(
            request.clone({
              setHeaders: {
                Authorization: `Bearer ${newToken}`,
                ...(groupID ? { 'X-Group-ID': groupID } : {}),
              },
              context: request.context.set(AUTH_RETRY, true),
            }),
          ),
        ),
        catchError((refreshError: unknown) => throwError(() => refreshError)),
      );
    }),
  );
};
