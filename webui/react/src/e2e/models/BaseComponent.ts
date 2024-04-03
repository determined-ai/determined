import { type Locator } from '@playwright/test';

import { BasePage } from './BasePage';

// BasePage is the root of any tree, use `instanceof BasePage` when climbing.
export type parentTypes = BasePage | BaseComponent | BaseReactFragment;

interface ComponentBasics {
  parent: parentTypes;
}

interface NamedComponentWithDefaultSelector extends ComponentBasics {
  attachment?: never;
  sleector?: never;
}
interface NamedComponentWithAttachment extends ComponentBasics {
  attachment: string;
  sleector?: never;
}
export interface BaseComponentArgs extends ComponentBasics {
  attachment?: never;
  selector: string;
}

export type NamedComponentArgs =
  | BaseComponentArgs
  | NamedComponentWithDefaultSelector
  | NamedComponentWithAttachment;

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

  constructor({ parent, selector }: BaseComponentArgs) {
    this._selector = selector;
    this._parent = parent;
  }

  get selector(): string {
    return this._selector;
  }

  /**
   * The playwright Locator that represents this model
   */
  get pwLocator(): Locator {
    if (this._locator === undefined) {
      // Treat the locator as a readonly, but only after we've created it
      this._locator = this._parent.pwLocator.locator(this.selector);
    }
    return this._locator;
  }

  /**
   * Returns the root of the component tree
   */
  get root(): BasePage {
    let root: parentTypes = this._parent;
    while (!(root instanceof BasePage)) {
      root = root._parent;
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

  constructor({ parent }: ComponentBasics) {
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
    let root: parentTypes = this._parent;
    while (!(root instanceof BasePage)) {
      root = root._parent;
    }
    return root;
  }
}

/**
 * Returns a representation of a named component. These components need a defaultSelector.
 * @param {object} obj
 * @param {parentTypes} obj.parent - The parent used to locate this NamedComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export abstract class NamedComponent extends BaseComponent {
  abstract readonly defaultSelector: string;
  readonly #attachment: string;

  override get selector(): string {
    return this._selector || this.defaultSelector + this.#attachment;
  }

  static getSelector(args: NamedComponentArgs): { selector: string; attachment: string } {
    if (NamedComponent.isBaseComponentArgs(args))
      return { attachment: '', selector: args.selector };
    if (NamedComponent.isNamedComponentWithAttachment(args))
      return { attachment: args.attachment, selector: '' };
    else return { attachment: '', selector: '' };
  }

  static isBaseComponentArgs(args: NamedComponentArgs): args is BaseComponentArgs {
    return 'selector' in args;
  }

  static isNamedComponentWithAttachment(
    args: NamedComponentArgs,
  ): args is NamedComponentWithAttachment {
    return 'attachment' in args;
  }
  constructor(args: NamedComponentArgs) {
    const { selector, attachment } = NamedComponent.getSelector(args);
    super({ parent: args.parent, selector });
    this.#attachment = attachment;
  }
}
