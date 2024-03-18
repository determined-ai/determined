import { type Locator } from '@playwright/test';

/**
 * Alias for type that has a function "locator" which takes a string and gives a Locator
 */
type implementsLocator = { locator: (arg0: string) => Locator }
/**
 * Alias for type that has a member "locator" which is of type implementsLocator
 * 
 * @remarks This enables us to call `this.loc` and expect to be able to call `.locator()`
 * It's like saying (BasePage | BaseComponent) without importing type BasePage
 */
type implementsGetLocator = { locator: implementsLocator }

export class canBeParent implements implementsGetLocator {

  /**
   * Sets subComponents as properties of this object
   * 
   * @remarks
   * This class exists so we can DRY `_initializeSubComponents`
   * 
   * @param {SubComponent[]} subComponents - List of subComponents to define as properties on this
   */
  _initializeSubComponents(subComponents: SubComponent[]): void {
    subComponents.forEach((subComponent) => {
      Object.defineProperty(this, subComponent.name, new BaseComponent({
        parent: this,
        selector: subComponent.selector,
        subComponents: subComponent.subComponents,
      }));
    });
  }

  // Idenity functions that should be reimplemented by BaseComponent and BasePage

  /**
   * Never Returns
   *
   * @remarks
   * Used by default methods that should be reimplemented
   */
  static #notImplemented(): never {
    throw new Error('not Implemented');
  }

  /**
   * Never Returns
   *
   * @remarks
   * Used by default methods that should be reimplemented
   */
  get locator(): implementsLocator { return canBeParent.#notImplemented(); }

  // Shorthand functions

  /**
   * Returns this.locator.
   */
  get loc(): implementsLocator { return this.locator; }
}

export interface BaseComponentProps {
  parent: canBeParent
  selector?: string
  subComponents?: SubComponent[]
}

export interface SubComponent {
  name: string
  type: typeof BaseComponent
  selector: string
  subComponents?: SubComponent[]
}

export class BaseComponent extends canBeParent {
  readonly defaultSelector: undefined | string;

  readonly _selector: string;
  _parent: canBeParent;
  _locator: Locator | undefined;

  /**
   * Returns the representation of a Component.
   * 
   * @remarks
   * This constructor is a base class for any component in src/components/.
   * 
   * @param {Object} obj
   * @param {implementsGetLocator} obj.parent - The parent used to locate this BaseComponent
   * @param {string} [obj.selector] - Used instead of `defaultSelector`
   * @param {SubComponent[]} [obj.subComponents] - SubComponents to initialize at runtime
   */
  constructor({ parent, selector, subComponents }: BaseComponentProps) {
    super();
    if (typeof this.defaultSelector === 'undefined') {
      throw new Error('defaultSelector is undefined');
    }
    this._selector = selector || this.defaultSelector;
    this._parent = parent;

    if (typeof subComponents !== 'undefined') {
      this._initializeSubComponents(subComponents);
    }
  }

  /**
   * Returns this object's Locator.
   *
   * @remarks
   * We use this method to call this.loc.locate().
   */
  override get locator(): Locator {
    if (typeof this._selector === 'undefined') {
      throw new Error('selector is undefined');
    }
    if (!this._locator) {
      this._locator = this._parent.loc.locator(this._selector);
    }
    return this._locator;
  }
}
