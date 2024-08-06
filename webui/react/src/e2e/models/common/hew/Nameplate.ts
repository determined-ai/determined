import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the SplitPane component from hew/src/kit/Nameplate.tsx
 */
export class Nameplate extends NamedComponent {
  readonly defaultSelector = '[class^="Nameplate_base"]';

  // We could replace this with a more specific selector in Hew
  #nameSelector = '.ant-typography:last-of-type';
  readonly icon = new BaseComponent({
    parent: this,
    selector: '[id="avatar"]',
  });
  readonly #text = new BaseComponent({
    parent: this,
    selector: '[class^="Nameplate_text"]',
  });
  readonly alias = new BaseComponent({
    parent: this.#text,
    selector: `.ant-typography:not(${this.#nameSelector})`,
  });
  readonly name = new BaseComponent({
    parent: this.#text,
    selector: this.#nameSelector,
  });
}
