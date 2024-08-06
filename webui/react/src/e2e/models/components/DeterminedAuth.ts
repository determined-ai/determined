import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { ErrorComponent } from 'e2e/models/utils/error';

/**
 * Represents the DeterminedAuth component in src/components/DeterminedAuth.tsx
 */
export class DeterminedAuth extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="detAuth"]';
  readonly #form = new BaseComponent({ parent: this, selector: 'form' });
  readonly username = new BaseComponent({
    parent: this.#form,
    selector: 'input[data-testid="username"]',
  });
  readonly password = new BaseComponent({
    parent: this.#form,
    selector: 'input[data-testid="password"]',
  });
  readonly submit = new BaseComponent({
    parent: this.#form,
    selector: 'button[data-testid="submit"]',
  });
  readonly docs = new BaseComponent({
    parent: this,
    selector: 'link[data-testid="docs"]',
  });
  readonly errors = new ErrorComponent({
    attachment: ErrorComponent.selectorTopRight,
    parent: this.root,
  });
}
