import { type Locator } from '@playwright/test';
import { BasePage } from './BasePage';

// BasePage is the root of any tree, use `instanceof BasePage` when climbing.
type parentTypes = BasePage | BaseComponent | BaseReactFragment

/**
 * Returns the representation of a Component.
 * 
 * @remarks
 * This constructor is a base class for any component in src/components/.
 * 
 * @param {Object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class BaseComponent {
  protected _selector: string;
  readonly _parent: parentTypes;
  protected _locator: Locator | undefined;

  constructor({ parent, selector }: { parent: parentTypes, selector: string }) {
    this._selector = selector;
    this._parent = parent;
  }

  /**
   * The playwright Locator that represents this model
   */
  get pwLocator(): Locator {
    if (typeof this._locator === 'undefined') {
      // Treat the locator as a readonly, but only after we've created it
      this._locator = this._parent.pwLocator.locator(this._selector);
    }
    return this._locator;
  }

  get root(): BasePage {
    let root: parentTypes = this
    while (true) {
      if (root instanceof BasePage) return root
      else root = root._parent
    }
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
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 */
export class BaseReactFragment {
  readonly _parent: parentTypes

  constructor({ parent }: { parent: parentTypes }) {
    this._parent = parent
  }
  /**
   * The playwright Locator that represents this model
   * 
   * @remarks
   * Since this model is a fragment, we simply get the parent's locator
   */
  get pwLocator(): Locator { return this._parent.pwLocator }
}

export type NamedComponentArgs = {
  parent: parentTypes,
  selector?: string
}

/**
 * The actual implemntation of a NamedComponent class
 *
 * @remarks
 * Remarks regarding implementation are found in the NamedComponent function
 */
export abstract class NamedComponent extends BaseComponent {
  static defaultSelector: string;
  constructor({ parent, selector }: { parent: parentTypes, selector: string }) {
    super({parent, selector})
    if ((this.constructor as any).defaultSelector == undefined){ 
      throw new Error('A named component has been defined without a default selector!')
    }
  }
}
