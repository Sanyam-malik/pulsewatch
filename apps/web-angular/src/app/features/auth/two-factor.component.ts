import { Component, EventEmitter, Input, Output, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';

@Component({
  selector: 'app-two-factor',
  standalone: true,
  imports: [ReactiveFormsModule],
  templateUrl: './two-factor.component.html',
  styleUrl: './two-factor.component.scss',
})
export class TwoFactorComponent {
  private readonly formBuilder = inject(FormBuilder);

  @Input({ required: true }) email = '';
  @Input() loading = false;
  @Input() error: string | null = null;
  @Output() verify = new EventEmitter<string>();
  @Output() back = new EventEmitter<void>();

  readonly form = this.formBuilder.nonNullable.group({
    token: ['', Validators.required],
  });

  submit(): void {
    if (this.form.invalid || this.loading) {
      this.form.markAllAsTouched();
      return;
    }
    this.verify.emit(this.form.controls.token.value.trim());
  }
}
