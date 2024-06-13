import { type Locator } from '@playwright/test';

import { BasePage, ModelBasics } from './BasePage';

export type CanBeParent = ComponentBasics | BasePage;

export interface ComponentBasics extends ModelBasics {
  _parent: CanBeParent;
  get root(): BasePage;
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
 * Base model for any Component in src/components/
 */
export class BaseComponent implements ComponentBasics {
  protected _selector: string;
  readonly _parent: CanBeParent;
  protected _locator: Locator | undefined;

  /**
   * Constructs a BaseComponent
   * @param {object} obj
   * @param {CanBeParent} obj.parent - parent component
   * @param {string} obj.selector - identifier
   */
  constructor({ parent, selector }: BaseComponentArgs) {
    this._selector = selector;
    this._parent = parent;
  }

  /**
   * The identifier used to locate this model
   */
  get selector(): string {
    return this._selector;
  }

  /**
   * The playwright Locator that represents this model
   */
  get pwLocator(): Locator {
    // Treat the locator as a readonly, but only after we've created it
    if (this._locator === undefined) {
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
 * BaseReactFragment will preserve the parent locator heirachy while also
 * providing a way to group elements, just like the React Fragments they model.
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
    let root: CanBeParent = this._parent;
    while (!(root instanceof BasePage)) {
      root = root._parent;
    }
    return root;
  }
}

/**
 * Named Components are components that have a default selector
 */
export abstract class NamedComponent extends BaseComponent {
  abstract readonly defaultSelector: string;
  readonly #attachment: string;

  /**
   * The identifier used to locate this model
   */
  override get selector(): string {
    return this._selector || this.defaultSelector + this.#attachment;
  }

  /**
   * Internal method used to compute the named component's selector
   */
  private static getSelector(args: NamedComponentArgs): { selector: string; attachment: string } {
    if (NamedComponent.isBaseComponentArgs(args))
      return { attachment: '', selector: args.selector };
    if (NamedComponent.isNamedComponentWithAttachment(args))
      return { attachment: args.attachment, selector: '' };
    else return { attachment: '', selector: '' };
  }

  /**
   * Internal method to check the type of args passed into the constructor
   */
  private static isBaseComponentArgs(args: NamedComponentArgs): args is BaseComponentArgs {
    return 'selector' in args;
  }

  /**
   * Internal method to check the type of args passed into the constructor
   */
  private static isNamedComponentWithAttachment(
    args: NamedComponentArgs,
  ): args is NamedComponentWithAttachment {
    return 'attachment' in args;
  }

  /**
   * Constructs a NamedComponent
   * @param {object} args
   * @param {CanBeParent} args.parent - parent component
   * @param {string} [args.selector] - identifier to be used in place of defaultSelector
   * @param {string} [args.attachment] - identifier to be appended to defaultSelector
   */
  constructor(args: NamedComponentArgs) {
    const { selector, attachment } = NamedComponent.getSelector(args);
    super({ parent: args.parent, selector });
    this.#attachment = attachment;
  }
}
