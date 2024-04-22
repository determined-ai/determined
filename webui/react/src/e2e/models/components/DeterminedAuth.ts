import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';
import { ErrorComponent } from 'e2e/models/utils/error';

/**
 * Returns a representation of the DeterminedAuth component.
 * This constructor represents the contents in src/components/DeterminedAuth.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this DeterminedAuth
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */

export class DeterminedAuth extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="detAuth"]';
  readonly #form: BaseComponent = new BaseComponent({ parent: this, selector: 'form' });
  readonly username: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: 'input[data-testid="username"]',
  });
  readonly password: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: 'input[data-testid="password"]',
  });
  readonly submit: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: 'button[data-testid="submit"]',
  });
  readonly docs: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'link[data-testid="docs"]',
  });
  // TODO consdier a BaseComponents plural class
  readonly errors: ErrorComponent = new ErrorComponent({
    attachment: ErrorComponent.selectorTopRight,
    parent: this.root,
  });
}
