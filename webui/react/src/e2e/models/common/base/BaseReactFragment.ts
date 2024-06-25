import { type Locator } from '@playwright/test';

import { CanBeParent, ComponentArgBasics, ComponentBasics } from './BaseComponent';
import { BasePage } from './BasePage';

/**
 * BaseReactFragment will preserve the parent locator heirachy while also
 * providing a way to group components, just like the React Fragments they model.
 */
export class BaseReactFragment implements ComponentBasics {
  readonly _parent: CanBeParent;

  /**
   * Constructs a BaseReactFragment
   * @param {object} obj
   * @param {CanBeParent} obj.parent - parent component
   */
  constructor({ parent }: ComponentArgBasics) {
    this._parent = parent;
  }

  /**
   * The playwright Locator that represents this model
   * Since this model is a fragment, we simply get the parent's locator
   */
  get pwLocator(): Locator {
    return this._parent.pwLocator;
  }

  /**
   * Returns the root of the component tree
   */
  get root(): BasePage {
    if (this._parent instanceof BasePage) {
      return this._parent;
    } else {
      return this._parent.root;
    }
  }
}
