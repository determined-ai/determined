import { type Locator } from '@playwright/test';

import { ModelBasics, BasePage } from './BasePage';

export type CanBeParent = ComponentBasics | BasePage

export interface ComponentBasics extends ModelBasics {
  _parent: CanBeParent
  get root(): BasePage
}

interface ComponentArgBasics {
  parent: CanBeParent;
}

interface NamedComponentWithDefaultSelector extends ComponentArgBasics {
  attachment?: never;
  sleector?: never;
}
interface NamedComponentWithAttachment extends ComponentArgBasics {
  attachment: string;
  sleector?: never;
}
export interface BaseComponentArgs extends ComponentArgBasics {
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
 * @param {CanBeParent} obj.parent - The parent used to locate this BaseComponent
 * @param {string} obj.selector - Used as a selector uesd to locate this object
 */
export class BaseComponent implements ModelBasics {
  protected _selector: string;
  readonly _parent: CanBeParent;
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
    let root: CanBeParent = this._parent;
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
 * @param {CanBeParent} obj.parent - The parent used to locate this BaseComponent
 */
export class BaseReactFragment implements ModelBasics {
  readonly _parent: CanBeParent;

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
    let root: CanBeParent = this._parent;
    while (!(root instanceof BasePage)) {
      root = root._parent;
    }
    return root;
  }
}

/**
 * Returns a representation of a named component. These components need a defaultSelector.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this NamedComponent
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
