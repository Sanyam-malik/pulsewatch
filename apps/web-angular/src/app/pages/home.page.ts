import { Component } from '@angular/core';

@Component({
  selector: 'app-home-page',
  standalone: true,
  template: `
    <main class="migration-notice">
      <span class="brand">Pulsewatch</span>
      <h1>Angular web client</h1>
      <p>The Angular client is being migrated alongside the current web app.</p>
    </main>
  `,
  styles: `
    .migration-notice {
      display: grid;
      min-height: 100vh;
      align-content: center;
      justify-items: center;
      padding: 2rem;
      text-align: center;
    }
    .brand { font-weight: 700; color: #2563eb; }
    h1 { margin: 1rem 0 0.5rem; }
    p { color: #64748b; }
  `,
})
export class HomePage {}
