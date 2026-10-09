import { DatePipe } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ApiEnvelope, Tag } from './admin.types';

@Component({
  selector: 'app-tags-page',
  standalone: true,
  imports: [DatePipe, ReactiveFormsModule, RouterLink],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      @if (editing) {
        <a routerLink="/tags">← Tags</a>
        <h1>{{ id ? 'Edit tag' : 'New tag' }}</h1>
        @if (loading) { <p>Loading tag…</p> }
        @else {
          <section class="card">
            <form class="fields" [formGroup]="form" (ngSubmit)="save()">
              <label class="field">Name<input formControlName="name" maxlength="100" required></label>
              <label class="field">Color<div class="grid"><input type="color" [value]="form.controls.color.value" (input)="form.controls.color.setValue($any($event.target).value)"><input formControlName="color" placeholder="#3B82F6"></div></label>
              <p class="muted">Choose a color to identify this tag.</p>
              <label class="field">Description<textarea formControlName="description"></textarea></label>
              @if (error) { <p class="error" role="alert">{{ error }}</p> }
              <div class="actions"><button class="secondary" type="button" routerLink="/tags">Cancel</button><button [disabled]="working || form.invalid">{{ id ? 'Update tag' : 'Create tag' }}</button></div>
            </form>
          </section>
        }
      } @else {
        <div class="row"><h1>Tags</h1><a routerLink="/tags/new"><button type="button">Create tag</button></a></div>
        <label class="field">Search tags<input type="search" (input)="search = $any($event.target).value; loadList()" placeholder="Search tags"></label>
        @if (loading) { <p>Loading tags…</p> }
        @if (error) { <p class="error" role="alert">{{ error }}</p> }
        @for (tag of tags; track tag.id) {
          <section class="card row">
            <div class="row-content"><span class="tag-badge" [style.background]="tag.color">{{ tag.name }}</span>
              @if (tag.description) { <span>{{ tag.description }}</span> }
              @if (tag.created_at) { <span class="muted">Created {{ tag.created_at | date }}</span> }
            </div>
            <div class="actions"><a [routerLink]="['/tags', tag.id, 'edit']"><button class="secondary" type="button">Edit</button></a><button class="danger" type="button" (click)="remove(tag)">Delete</button></div>
          </section>
        }
        @if (!loading && !tags.length) { <section class="card">No tags found. Create one to get started.</section> }
      }
    </main>`,
  styles: [`.tag-badge { display: inline-block; padding: .35rem .7rem; color: #fff; border-radius: .35rem; font-weight: 600; } .row-content { display:flex; align-items:center; gap:1rem; flex-wrap:wrap; }`],
})
export class TagsPage {
  private readonly api = inject(ApiService);
  private readonly fb = inject(FormBuilder);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  readonly id = this.route.snapshot.paramMap.get('id');
  readonly editing = this.route.snapshot.routeConfig?.path === 'tags/new' || !!this.id;
  readonly form = this.fb.nonNullable.group({
    name: ['', [Validators.required, Validators.maxLength(100)]],
    color: ['#3B82F6', [Validators.required, Validators.pattern(/^#[0-9a-fA-F]{6}$/)]],
    description: [''],
  });
  tags: Tag[] = [];
  search = '';
  loading = false;
  working = false;
  error = '';
  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() { if (this.id) this.loadTag(); else if (!this.editing) this.loadList(); }
  loadList(): void {
    if (this.searchTimer) clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => {
      this.loading = true;
      const params = new URLSearchParams({ page: '0', limit: '100' });
      if (this.search.trim()) params.set('q', this.search.trim());
      this.api.get<ApiEnvelope<Tag[]>>(`tags?${params.toString()}`).pipe(finalize(() => this.loading = false)).subscribe({
        next: ({ data }) => this.tags = data ?? [], error: () => this.error = 'Could not load tags.',
      });
    }, 250);
  }
  loadTag(): void {
    this.loading = true;
    this.api.get<ApiEnvelope<Tag>>(`tags/${this.id}`).pipe(finalize(() => this.loading = false)).subscribe({
      next: ({ data }) => this.form.patchValue({ name: data.name, color: data.color || '#3B82F6', description: data.description ?? '' }),
      error: () => this.error = 'Could not load tag.',
    });
  }
  save(): void {
    if (this.form.invalid || this.working) return;
    this.working = true;
    const request = this.id ? this.api.put(`tags/${this.id}`, this.form.getRawValue()) : this.api.post('tags', this.form.getRawValue());
    request.pipe(finalize(() => this.working = false)).subscribe({
      next: () => void this.router.navigateByUrl('/tags'), error: () => this.error = `Could not ${this.id ? 'update' : 'create'} tag.`,
    });
  }
  remove(tag: Tag): void {
    if (!confirm(`Delete tag “${tag.name}”?`)) return;
    this.api.delete(`tags/${tag.id}`).subscribe({
      next: () => this.tags = this.tags.filter((item) => item.id !== tag.id), error: () => this.error = 'Could not delete tag.',
    });
  }
}
