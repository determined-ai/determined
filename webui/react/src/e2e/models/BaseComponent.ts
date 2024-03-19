import { type Locator } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Alias for type that has a function "locator" which takes a string and gives a Locator
 */
type HasLocator = { locator: (arg0: string) => Locator }
/**
 * Alias for type that has a member "locator" which is of type HasLocator
 * 
 * @remarks This enables us to call `this.loc` and expect to be able to call `.locator()`
 * It's like saying (BasePage | BaseComponent) without importing type BasePage
 */
type GetsLocator = { locator: HasLocator }

export abstract class canBeParent {

  // all parents can have subComponents
  readonly subComponents: Map<String,BaseComponent> = new Map()
  readonly sc: Map<String,BaseComponent> = this.subComponents

  /**
   * Sets subComponents as properties of this object
   * 
   * @remarks
   * This class exists so we can DRY `initializeSubComponents`
   * 
   * @param {SubComponent[]} subComponents - List of subComponents to define as properties on this
   */
  protected initializeSubComponents(subComponents: SubComponent[]): void {
    subComponents.forEach((subComponent) => {
      const newComponent = new subComponent.type({
        parent: this,
        selector: subComponent.selector,
        subComponents: subComponent.subComponents,
      })
      this.subComponents.set(subComponent.name, newComponent)
    });
  }

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

export class BaseComponent extends canBeParent implements GetsLocator {
  readonly defaultSelector: undefined | string;

  readonly #selector: string;
  protected parent: canBeParent;
  #locator: Locator | undefined;

  /**
   * Returns the representation of a Component.
   * 
   * @remarks
   * This constructor is a base class for any component in src/components/.
   * 
   * @param {Object} obj
   * @param {GetsLocator} obj.parent - The parent used to locate this BaseComponent
   * @param {string} [obj.selector] - Used instead of `defaultSelector`
   * @param {SubComponent[]} [obj.subComponents] - SubComponents to initialize at runtime
   */
  constructor({ parent, selector, subComponents }: BaseComponentProps) {
    super();
    if (typeof this.defaultSelector === 'undefined') {
      throw new Error('defaultSelector is undefined');
    }
    this.#selector = selector || this.defaultSelector;
    this.parent = parent;

    if (typeof subComponents !== 'undefined') {
      this.initializeSubComponents(subComponents);
    }
  }

  /**
   * Returns this object's Locator.
   *
   * @remarks
   * We use this method to call this.loc.locate().
   */
  get locator(): Locator {
    if (typeof this.#selector === 'undefined') {
      throw new Error('selector is undefined');
    }
    if (!this.#locator) {
      if (this.parent instanceof BasePage) {
        this.#locator = this.parent._page.locator(this.#selector);
      } else if (this.parent instanceof BaseComponent) {
        this.#locator = this.parent.loc.locator(this.#selector);
      } else {
        throw new Error(`parent is bad type ${typeof this.parent}`);
      }
    }
    return this.#locator;
  }

  loc = this.locator
}
