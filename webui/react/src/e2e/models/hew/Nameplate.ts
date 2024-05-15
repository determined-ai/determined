import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Nameplate component from Hew.
 * This constructor represents the contents in hew/src/kit/Nameplate.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Nameplate
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Nameplate extends NamedComponent {
  readonly defaultSelector = '[class^="Nameplate_base"]';

  // We could replace this with a more specific selector in Hew
  #nameSelector = '.ant-typography:last-of-type';
  readonly icon = new BaseComponent({
    parent: this,
    selector: '[id="avatar"]',
  });
  readonly #text: BaseComponent = new BaseComponent({
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
