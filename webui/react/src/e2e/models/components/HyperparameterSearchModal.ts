import { BaseComponent, BaseComponentArgs } from 'playwright-page-model-base/BaseComponent';
import { OverlayArgs } from 'playwright-page-model-base/BaseOverlay';

import { Modal } from 'e2e/models/common/hew/Modal';
import { Select } from 'e2e/models/common/hew/Select';

type HyperparameterSearchModalPage = 'searcher' | 'hyperparameter';

/**
 * Represents the HyperparameterSearchModal component in src/components/HyperparameterSearchModal.tsx
 */
export default class HyperparameterSearchModal extends Modal {
  #pages = [new SearcherPage({ parent: this }), new HyperparameterPage({ parent: this })] as const;
  #_page: SearcherPage | HyperparameterPage;
  constructor(args: OverlayArgs) {
    super(args);
    this.#_page = this.#pages[0];
  }

  get page(): SearcherPage | HyperparameterPage {
    return this.#_page;
  }

  set page(page: HyperparameterSearchModalPage) {
    if (page === 'searcher') this.#_page = this.#pages[0];
    else if (page === 'hyperparameter') this.#_page = this.#pages[1];
  }
}

class SearcherPage extends BaseComponent {
  constructor(args: Omit<BaseComponentArgs, 'selector'>) {
    super({ ...args, selector: 'div[id="searcher-page"]' });
  }

  readonly title = 'searcher';
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
    selector: 'input[id="pool"]',
  });
  readonly slotsInput = new BaseComponent({
    parent: this,
    selector: 'input[id="slots_per_trial"]',
  });
  readonly earlyStoppingInput = new Select({
    parent: this,
    selector: 'input[id="mode"]',
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

  readonly title = 'hyperparameter';
}
