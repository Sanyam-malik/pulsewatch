import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-domains-manager',
  standalone: true,
  imports: [FormsModule],
  template: `
    <section class="domains">
      <label for="domain-entry">Custom domains</label>
      @for (domain of value; track $index) {
        <div class="domain-row">
          <span>{{ domain }}</span>
          @if (domain === currentHost) { <strong>Warning: this is the current host.</strong> }
          @if (domain === highlightedDomain && error) { <strong class="error">{{ error }}</strong> }
          <button type="button" aria-label="Remove domain" (click)="remove($index)">Remove</button>
        </div>
      }
      <div class="domain-entry">
        <input id="domain-entry" [(ngModel)]="draft" (keydown.enter)="add($event)" placeholder="status.example.com" />
        <button type="button" (click)="add()">Add domain</button>
      </div>
    </section>
  `,
  styles: [`
    .domains { display: grid; gap: .5rem; }
    .domain-row, .domain-entry { display: flex; gap: .75rem; align-items: center; flex-wrap: wrap; }
    .domain-row { padding: .65rem; background: #f4f5f7; border-radius: .4rem; }
    .domain-entry input { flex: 1; min-width: 12rem; }
    .error { color: #b42318; }
  `],
})
export class DomainsManagerComponent {
  @Input() value: string[] = [];
  @Input() error?: string;
  @Input() highlightedDomain?: string;
  draft = '';
  readonly currentHost = typeof location === 'undefined' ? '' : location.hostname;

  add(event?: Event): void {
    event?.preventDefault();
    const domain = this.draft.trim();
    if (domain && !this.value.includes(domain)) this.valueChange([...this.value, domain]);
    this.draft = '';
  }

  remove(index: number): void {
    this.valueChange(this.value.filter((_, i) => i !== index));
  }

  private valueChange(value: string[]): void {
    this.value = value;
    this.changed.emit(value);
  }

  // The form uses this event to keep domain edits in the API payload.
  @Output() changed = new EventEmitter<string[]>();
}
