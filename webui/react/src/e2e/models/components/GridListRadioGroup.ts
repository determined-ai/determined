import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

export class GridListRadioGroup extends BaseComponent {
  readonly grid = new BaseComponent({
    parent: this,
    selector: 'label:first-of-type',
  });
  readonly list = new BaseComponent({
    parent: this,
    selector: 'label:last-of-type',
  });
}
