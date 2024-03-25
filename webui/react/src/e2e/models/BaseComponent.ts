import { type Locator } from '@playwright/test';

import { BasePage } from './BasePage';

// BasePage is the root of any tree, use `instanceof BasePage` when climbing.
type parentTypes = BasePage | BaseComponent | BaseReactFragment;

/**
 * Returns the representation of a Component.
 * This constructor is a base class for any component in src/components/.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class BaseComponent {
  protected _selector: string;
  readonly _parent: parentTypes;
  protected _locator: Locator | undefined;

  constructor({ parent, selector }: { parent: parentTypes; selector: string }) {
    this._selector = selector;
    this._parent = parent;
  }

  /**
   * The playwright Locator that represents this model
   */
  get pwLocator(): Locator {
    if (this._locator === undefined) {
      // Treat the locator as a readonly, but only after we've created it
      this._locator = this._parent.pwLocator.locator(this._selector);
    }
    return this._locator;
  }

  get root(): BasePage {
    let root: parentTypes = this._parent;
    for (; !(root instanceof BasePage); root = root._parent) {
      /* empty */
    }
    return root;
  }
}

/**
 * Returns the representation of a React Fragment.
 * React Fragment Components are special in that they group elements, but not under a dir.
 * Fragments cannot have selectors
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this BaseComponent
 */
export class BaseReactFragment {
  readonly _parent: parentTypes;

  constructor({ parent }: { parent: parentTypes }) {
    this._parent = parent;
  }
  /**
   * The playwright Locator that represents this model
   * Since this model is a fragment, we simply get the parent's locator
   */
  get pwLocator(): Locator {
    return this._parent.pwLocator;
  }
}

export type NamedComponentArgs = {
  parent: parentTypes;
  selector?: string;
};

/**
 * Returns a representation of a named component.
 * This class enforces that a `static defaultSelector` and `static url` be declared
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this NamedComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export abstract class NamedComponent extends BaseComponent {
  constructor({ parent, selector }: { parent: parentTypes; selector: string }) {
    super({ parent, selector });
    const requiredStaticProperties: string[] = ['defaultSelector'];
    requiredStaticProperties.forEach((requiredProp) => {
      if (!Object.hasOwn(this.constructor, requiredProp)) {
        throw new Error(`A named component must declare a static ${requiredProp}!`);
      }
    });
  }
}
