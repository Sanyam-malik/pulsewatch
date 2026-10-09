import { TestBed } from '@angular/core/testing';
import { TwoFactorComponent } from './two-factor.component';

describe('TwoFactorComponent', () => {
  it('requires a code and emits a trimmed code for verification', async () => {
    await TestBed.configureTestingModule({
      imports: [TwoFactorComponent],
    }).compileComponents();
    const fixture = TestBed.createComponent(TwoFactorComponent);
    const component = fixture.componentInstance;
    const verify = vi.spyOn(component.verify, 'emit');

    component.submit();
    expect(verify).not.toHaveBeenCalled();

    component.form.controls.token.setValue(' 123456 ');
    component.submit();
    expect(verify).toHaveBeenCalledWith('123456');
  });
});
