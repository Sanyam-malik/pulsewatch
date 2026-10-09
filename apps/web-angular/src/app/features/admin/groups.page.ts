import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { finalize } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AuthService } from '../../core/auth.service';
import { ApiEnvelope, Group, GroupMember } from './admin.types';

@Component({
  selector: 'app-groups-page',
  standalone: true,
  imports: [ReactiveFormsModule],
  styleUrl: './admin.styles.scss',
  template: `
    <main class="admin-page">
      <h1>Groups and access</h1>
      <section class="card">
        <h2>Groups</h2>
        <p>Groups are workspaces. Your access level is assigned separately in each group.</p>
        @if (loadingGroups) { <p>Loading groups…</p> }
        @if (groups.length) {
          <div class="grid">
            <label class="field">Active group
              <select [value]="selected?.id ?? ''" (change)="selectGroup($any($event.target).value)">
                @for (group of groups; track group.id) {
                  <option [value]="group.id">{{ group.name }} — {{ group.role }}</option>
                }
              </select>
            </label>
          </div>
        }
        <form class="actions" [formGroup]="createForm" (ngSubmit)="createGroup()">
          <input aria-label="New group name" formControlName="name" maxlength="255" placeholder="New group name">
          <button [disabled]="creating || createForm.invalid">{{ creating ? 'Creating…' : 'Create group' }}</button>
        </form>
        @if (error) { <p class="error" role="alert">{{ error }}</p> }
      </section>
      @if (selected) {
        <section class="card">
          <h2>Members of {{ selected.name }}</h2>
          <p>Owner: full group control. Admin: manage members. Member: manage monitors. Viewer: read-only.</p>
          @if (!canManage) { <p>Only group owners and admins can view or manage the member list.</p> }
          @else {
            <form class="grid" [formGroup]="memberForm" (ngSubmit)="addMember()">
              <label class="field">Member email<input type="email" formControlName="email" required></label>
              <label class="field">Initial password (new account)<input type="password" formControlName="password"></label>
              <label class="field">Role
                <select formControlName="role">
                  <option value="member">Member</option><option value="viewer">Viewer</option>
                  @if (selected.role === 'owner') { <option value="admin">Admin</option> }
                </select>
              </label>
              <button [disabled]="working || memberForm.invalid">Add member</button>
            </form>
            <p class="muted">Existing accounts can be added without a password. For a new account, set a strong initial password and share it securely.</p>
            @if (loadingMembers) { <p>Loading members…</p> }
            @for (member of members; track member.user.id) {
              <div class="row">
                <div><strong>{{ member.user.email }}</strong><div class="muted">{{ member.user.active ? 'Active account' : 'Disabled account' }}</div></div>
                @if (member.role === 'owner') { <strong>Owner</strong> }
                @else {
                  <div class="actions">
                    <select [attr.aria-label]="'Role for ' + member.user.email" [value]="member.role" (change)="changeRole(member.user.id, $any($event.target).value)">
                      @if (selected.role === 'owner') { <option value="admin">Admin</option> }
                      <option value="member">Member</option><option value="viewer">Viewer</option>
                    </select>
                    <button class="danger" (click)="removeMember(member.user.id)">Remove</button>
                  </div>
                }
              </div>
            }
          }
        </section>
      }
    </main>`,
})
export class GroupsPage {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder);
  readonly createForm = this.fb.nonNullable.group({ name: ['', [Validators.required, Validators.maxLength(255)]] });
  readonly memberForm = this.fb.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
    password: [''],
    role: ['member' as 'admin' | 'member' | 'viewer'],
  });
  groups: Group[] = [];
  members: GroupMember[] = [];
  selected: Group | undefined;
  loadingGroups = true;
  loadingMembers = false;
  creating = false;
  working = false;
  error = '';
  get canManage(): boolean { return this.selected?.role === 'owner' || this.selected?.role === 'admin'; }

  constructor() { this.loadGroups(); }

  loadGroups(): void {
    this.loadingGroups = true;
    this.api.get<ApiEnvelope<Group[]>>('groups').pipe(finalize(() => this.loadingGroups = false)).subscribe({
      next: ({ data }) => {
        this.groups = data ?? [];
        const id = this.auth.activeGroupID() ?? this.auth.user()?.groupId;
        this.selected = this.groups.find((group) => group.id === id) ?? this.groups[0];
        if (this.selected) this.selectGroup(this.selected.id);
      },
      error: () => this.error = 'Could not load groups.',
    });
  }

  selectGroup(id: string): void {
    this.selected = this.groups.find((group) => group.id === id);
    if (!this.selected) return;
    this.auth.setActiveGroupID(this.selected.id);
    this.auth.setUser({ ...this.auth.user(), groupId: this.selected.id, role: this.selected.role });
    if (!this.canManage) { this.members = []; return; }
    this.loadingMembers = true;
    this.api.get<ApiEnvelope<GroupMember[]>>(`groups/${this.selected.id}/members`)
      .pipe(finalize(() => this.loadingMembers = false))
      .subscribe({ next: ({ data }) => this.members = data ?? [], error: () => this.error = 'Could not load group members.' });
  }

  createGroup(): void {
    if (this.createForm.invalid || this.creating) return;
    this.creating = true;
    this.error = '';
    this.api.post<ApiEnvelope<Group>>('groups', this.createForm.getRawValue()).pipe(finalize(() => this.creating = false)).subscribe({
      next: ({ data }) => {
        this.createForm.reset();
        this.auth.setActiveGroupID(data.id);
        this.auth.setUser({ ...this.auth.user(), groupId: data.id, role: data.role });
        this.loadGroups();
      },
      error: () => this.error = 'Could not create group.',
    });
  }

  addMember(): void {
    if (!this.selected || this.memberForm.invalid || this.working) return;
    this.working = true;
    const value = this.memberForm.getRawValue();
    const body = { email: value.email, password: value.password || undefined, role: value.role };
    this.api.post<ApiEnvelope<GroupMember>>(`groups/${this.selected.id}/members`, body).pipe(finalize(() => this.working = false)).subscribe({
      next: () => { this.memberForm.reset({ email: '', password: '', role: 'member' }); this.selectGroup(this.selected!.id); },
      error: () => this.error = 'Could not add member. Check the email, role, and password.',
    });
  }

  changeRole(id: string, role: 'admin' | 'member' | 'viewer'): void {
    if (!this.selected) return;
    this.api.patch(`groups/${this.selected.id}/members/${id}`, { role }).subscribe({
      next: () => this.selectGroup(this.selected!.id), error: () => this.error = 'Could not update access.',
    });
  }

  removeMember(id: string): void {
    if (!this.selected) return;
    this.api.delete(`groups/${this.selected.id}/members/${id}`).subscribe({
      next: () => this.selectGroup(this.selected!.id),
      error: () => this.error = 'Could not remove member. A group must retain an owner.',
    });
  }
}
