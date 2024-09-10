import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

export class GridListRadioGroup extends NamedComponent {
  override defaultSelector = '[data-test-component="grid-list-radio-group"]';
  readonly grid = new BaseComponent({
    parent: this,
    selector: 'label:first-of-type',
  });
  readonly list = new BaseComponent({
    parent: this,
    selector: 'label:last-of-type',
  });
}
