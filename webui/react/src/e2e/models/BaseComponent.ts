import { type Locator } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Used in the constructor for BaseComponent
 */
export interface BaseComponentProps {
  parent: BasePage | BaseComponent
  selector?: string
}

/**
 * Returns the representation of a Component.
 * 
 * @remarks
 * This constructor is a base class for any component in src/components/.
 * 
 * @param {Object} obj
 * @param {BasePage | BaseComponent} obj.parent - The parent used to locate this BaseComponent
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class BaseComponent {
  readonly defaultSelector: undefined | string;

  readonly #selector: string;
  protected readonly _parent: BasePage | BaseComponent;
  protected _locator: Locator | undefined;

  constructor({ parent, selector }: BaseComponentProps) {
    if (typeof this !== typeof BaseComponent && typeof this.defaultSelector === 'undefined') {
      throw new Error(`defaultSelector is undefined in class ${typeof this}`);
    }
    if (typeof this === typeof BaseComponent && typeof selector === 'undefined') {
      throw new Error(`BaseComponent needs a selector`);
    }

    // guardrails above ensure that either selector or defaultSelector are defined
    this.#selector = selector || this.defaultSelector!;
    this._parent = parent;
  }

  /**
   * The playwright locator method from this model's locator
   */
  get pwLocatorFunction() { return this.locateSelf.locator }

  /**
   * The playwright Locator that represents this model
   */
  get locateSelf(): Locator {
    if (typeof this._locator === 'undefined') {
      this._locator = this._parent.pwLocatorFunction(this.#selector);
      Object.freeze(this._locator)
    }
    return this._locator;
  }
}

/**
 * Returns the representation of a React Fragment.
 * 
 * @remarks
 * React Fragment Components are special in that they group elements, but not under a dir.
 * Fragments cannot have selectors
 * 
 * @param {Object} obj
 * @param {BasePage | BaseComponent} obj.parent - The parent used to locate this BaseComponent
 */
export class BaseReactFragment extends BaseComponent {
  // we never use the defaultSelector, but there are guardrails enforcing it be set
  override readonly defaultSelector: string = '';

  constructor({ parent }: { parent: BasePage | BaseComponent}) {
    super({parent: parent})
  }
  /**
   * The playwright Locator that represents this model
   * 
   * @remarks
   * Since this model is a fragment, we simply get the parent's locator
   */
  override get pwLocatorFunction() { return this._parent.pwLocatorFunction }
}
