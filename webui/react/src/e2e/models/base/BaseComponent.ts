import { type Locator } from '@playwright/test';

import { BasePage, ModelBasics } from './BasePage';

export type CanBeParent = ComponentBasics | BasePage;

export interface ComponentBasics extends ModelBasics {
  _parent: CanBeParent;
  get root(): BasePage;
}

function implementsComponentBasics(obj: object): obj is ComponentBasics {
  return '_parent' in obj && 'root' in obj;
}

export interface ComponentArgBasics {
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
    if (!Object.prototype.hasOwnProperty.call(this, '_locator') || this._locator === undefined) {
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

  /**
   * Returns the nth component which matches the selector
   */
  nth(n: number): this {
    /**
     * @param {T1} containerComponent the component to search properties for references to oldParent
     * @param {T2} oldParent the old parent to replace
     * @param {T2} newParent the new parent to replace with
     *
     * iterate through every property in the containerComponent prototype, if a
     * property is a component and it's parent is oldParent, replace it with a
     * new component with the same properties but a new parent.
     */
    const replaceParent = <T1 extends ComponentBasics, T2 extends ComponentBasics>(
      containerComponent: T1,
      oldParent: T2,
      newParent: T2,
    ): void => {
      // this isn't a deep search, but i don't expect components to be instanciated
      // with a parent set to "this.parent". it's always set to "this" or another
      // component with "this" as a parent.
      for (const key in containerComponent) {
        // special case for _parent so we don't recurse backwards
        if (key === '_parent') continue;
        const childComponent = containerComponent[key];
        if (childComponent instanceof Object && implementsComponentBasics(childComponent)) {
          // make a copy of the component and set it's parent to resetComponent
          if (childComponent._parent === oldParent) {
            // create a new object with the same properties as the childComponent
            const newChildComponent = Object.create(childComponent);
            // update the parent of the new object
            newChildComponent._parent = newParent;
            // update the property in containerComponent
            containerComponent[key] = newChildComponent;
            // update any properties in containerComponent which have a parent of childComponent
            replaceParent(containerComponent, childComponent, newChildComponent);
            // update any properties in 'newChildComponent' which has a parent of childComponent
            replaceParent(newChildComponent, childComponent, newChildComponent);
          }
        }
      }
    };

    const nthObj = Object.create(this, {
      pwLocator: {
        configurable: true,
        enumerable: true,
        get: () => {
          return this.pwLocator.nth(n);
        },
      },
    });
    replaceParent(nthObj, this, nthObj);
    return nthObj;
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
