import { type Locator } from '@playwright/test';

import { BasePage, ModelBasics } from './BasePage';

export type CanBeParent = ComponentBasics | BasePage;

export interface ComponentBasics extends ModelBasics {
  _parent: CanBeParent;
  get root(): BasePage;
  directChildren(): Generator<string>;
}

function implementsComponentBasics(obj: object): obj is ComponentBasics {
  return '_parent' in obj && 'root' in obj;
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
  protected _locator?: Locator;
  private indirectChildren: string[] = [];
  // DEBUGGING
  isNth = false;

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
    if (this._parent instanceof BasePage) {
      return this._parent;
    } else {
      return this._parent.root;
    }
  }

  *directChildren(): Generator<string> {
    // iterate through every property in the object prototype.
    // this isn't a deep search, but components i don't expect components to
    // be intanciated with a parent set to "this.parent". it's always set
    // to "this" or another component with "this" as a parent. if there are any
    // exceptions, the components should be added to the indirectChildren array.
    for (const key in Object.keys(this)) {
      if (key === '_parent') continue;
      const childComponent = Object.getPrototypeOf(this)[key];
      if (
        Object.prototype.hasOwnProperty.call(this, key) &&
        childComponent instanceof Object &&
        implementsComponentBasics(childComponent) &&
        childComponent._parent === this
      ) {
        yield key;
      }
    }
  }

  *allChildren(): Generator<string> {
    for (const key of this.directChildren()) {
      yield key;
    }
    for (const key in this.indirectChildren) {
      yield key;
    }
  }

  /**
   * Returns the nth component which matches the selector
   */
  nth(n: number): this {
    // given a component, reset the parent of all children to the new component
    const resetParent = <T extends ComponentBasics>(resetComponent: T): T => {
      const newObj = Object.create(resetComponent);
      newObj.isNth = true;
      for (const key in resetComponent.directChildren()) {
        if (key === 'columnName') {
          key;
        }
        const newChild = resetParent(newObj[key]);
        newChild._parent = newObj;
        newObj[key] = newChild;
      }
      return newObj;
    };
    const nthObj = resetParent(this);
    Object.defineProperty(nthObj, 'pwLocator', {
      configurable: true,
      enumerable: true,
      get: () => {
        return this.pwLocator.nth(n);
      },
    });
    return nthObj;
  }
}

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

  *directChildren(): Generator<string> {
    // iterate through every property in the object prototype.
    // this isn't a deep search, but components i don't expect components to
    // be intanciated with a parent set to "this.parent". it's always set
    // to "this" or another component with "this" as a parent. if there are any
    // exceptions, the components should be added to the indirectChildren array.
    for (const key in Object.keys(this)) {
      if (key === '_parent') continue;
      const childComponent = Object.getPrototypeOf(this)[key];
      if (
        Object.prototype.hasOwnProperty.call(this, key) &&
        childComponent instanceof Object &&
        implementsComponentBasics(childComponent) &&
        childComponent._parent === this
      ) {
        yield key;
      }
    }
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
