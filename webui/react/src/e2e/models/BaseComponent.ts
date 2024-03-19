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
   * Returns this object's Locator.
   *
   * @remarks
   * We use this method to call this.loc.locate().
   */
  get pwLocator(): Locator {
    // only set this._locator once. maybe consider redefining it as readonly
    if (typeof this._locator === 'undefined') {
      // this feels contrived. maybe we can have each parent return the method instead of a locator on self
      if (this._parent instanceof BasePage) {
        this._locator = this._parent._page.locator(this.#selector);
      } else if (this._parent instanceof BaseReactFragment) {
        const ancestor = this._parent._parent
        if (ancestor instanceof BaseComponent) {
          this._locator = ancestor.pwLocator.locator(this.#selector)
        } else {
          this._locator = ancestor._page.locator(this.#selector);
        }
      } else {
        this._locator = this._parent.pwLocator.locator(this.#selector);
      }
    }
    return this._locator;
  }
}

/**
 * Returns the representation of a React Fragment.
 * 
 * @remarks
 * React Fragment Components are special in that they group elements, but not under a dir.
 * 
 * @param {Object} obj
 * @param {BasePage | BaseComponent} obj.parent - The parent used to locate this BaseComponent
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class BaseReactFragment extends BaseComponent { }
