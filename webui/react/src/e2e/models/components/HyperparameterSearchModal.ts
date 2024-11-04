import { BaseComponent, BaseComponentArgs } from 'playwright-page-model-base/BaseComponent';
import { OverlayArgs } from 'playwright-page-model-base/BaseOverlay';

import { Modal } from 'e2e/models/common/hew/Modal';
import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the HyperparameterSearchModal component in src/components/HyperparameterSearchModal.tsx
 */
export default class HyperparameterSearchModal extends Modal {
  constructor(args: OverlayArgs) {
    super(args);
  }

  readonly searcherPage = new SearcherPage({ parent: this });
  readonly hyperparameterPage = new HyperparameterPage({ parent: this });
}

class SearcherPage extends BaseComponent {
  constructor(args: Omit<BaseComponentArgs, 'selector'>) {
    super({ ...args, selector: 'div[id="searcher-page"]' });
  }
  readonly #searchMethods = new BaseComponent({
    parent: this,
    selector: '.ant-radio-group',
  });
  readonly #searcherButtons = new BaseComponent({
    parent: this.#searchMethods,
    selector: 'button',
  });
  readonly adaptiveSearcher = this.#searcherButtons.nth(0);
  readonly gridSearcher = this.#searcherButtons.nth(1);
  readonly randomSearcher = this.#searcherButtons.nth(2);

  readonly nameInput = new BaseComponent({
    parent: this,
    selector: 'input[id="name"]',
  });
  readonly poolInput = new Select({
    parent: this,
    selector: '[data-test="pool"]',
  });
  readonly slotsInput = new BaseComponent({
    parent: this,
    selector: 'input[id="slots_per_trial"]',
  });
  readonly earlyStoppingInput = new Select({
    parent: this,
    selector: '[data-test="mode"]',
  });
  readonly stopOnceInput = new BaseComponent({
    parent: this,
    selector: 'input[id="stop_once"][type=checkbox]',
  });
  readonly maxRunsInput = new BaseComponent({
    parent: this,
    selector: 'input[id="max_trials"]',
  });
  readonly maxConcurrentRunsInput = new BaseComponent({
    parent: this,
    selector: 'input[id="max_concurrent_trials"]',
  });
}

class HyperparameterPage extends BaseComponent {
  constructor(args: Omit<BaseComponentArgs, 'selector'>) {
    super({ ...args, selector: 'div[id="hyperparameter-page"]' });
  }

  readonly hpNameInput = new BaseComponent({
    parent: this,
    selector: 'input[id*="name"]',
  });
  readonly hpTypeInput = new Select({
    parent: this,
    selector: '[data-test*="type"]',
  });
  readonly hpValInput = new BaseComponent({
    parent: this,
    selector: 'input[id*="value"]',
  });
  readonly hpMinInput = new BaseComponent({
    parent: this,
    selector: 'input[id*="min"]',
  });
  readonly hpMaxInput = new BaseComponent({
    parent: this,
    selector: 'input[id*="max"]',
  });
  readonly hpCountInput = new BaseComponent({
    parent: this,
    selector: 'input[id*="count"]',
  });
  readonly hpDeleteButton = new BaseComponent({
    parent: this,
    selector: '[class*="HyperparameterSearchModal_delete"]',
  });

  readonly addHpButton = new BaseComponent({
    parent: this,
    selector: '[id="add"]',
  });
}
