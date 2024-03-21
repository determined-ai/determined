import { type Locator } from '@playwright/test';
import { BasePage } from './BasePage';

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
      this._locator = this._parent.pwLocator.locator(this._selector);
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

export interface NamedComponentProps {
  parent: parentTypes,
  selector?: string
}

/**
 * The actual implemntation of a NamedComponent class
 *
 * @remarks
 * Remarks regarding implementation are found in the NamedComponent function
 */
abstract class _NamedComponent extends BaseComponent {
  static defaultSelector: string;
}

/**
 * Function used to extend the NamedComponent class
 * 
 * @param {Object} mandatory
 * @param {string} mandatory.defaultSelector - A selector to locate the object
 * 
 * @remarks
 * Named components should all come with a default selector so that their parents don't have to specify a selector.
 * Since the default selector is static, we can access and append to it if we want a more specific selector.
 * All named components should have a constructor that uses the defaultSelector as a selector if selector is undefined.
 */
export function NamedComponent(mandatory: { defaultSelector: string }) {
  return class extends _NamedComponent {
    static override defaultSelector = mandatory.defaultSelector
  }
}
// Classes are just a type and a function
export type NamedComponent = typeof _NamedComponent
