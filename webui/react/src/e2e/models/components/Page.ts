import { BaseComponent, BaseComponentProps, SubComponent } from 'e2e/models/BaseComponent';
// TODO Unit tests

export class Page extends BaseComponent {
  override defaultSelector: string = '';

  /**
   * Returns a representation of the Page component.
   *
   * @remarks
   * This constructor represents the contents in src/components/Page.tsx.
   * This constructor will also set a copy of its subComponents to it's parent.
   * ie. a parent may call this.theComponent or this.thePageComponent.theComponent
   *
   * @param {Object} obj
   * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
   * @param {string} [obj.selector] - Used instead of `defaultSelector`
   * @param {SubComponent[]} [obj.subComponents] - SubComponents to initialize at runtime
   */
  constructor({ parent, selector, subComponents }: BaseComponentProps) {
    // call the super like normal, but without subComponents. we'll set them manually
    super({ parent: parent, selector: selector });
    const parentSubComponents: SubComponent[] = [
      // TODO put spinners and other things from Page in here
    ];
    let allSubComponents = parentSubComponents
    if (typeof subComponents !== 'undefined') {
      allSubComponents = allSubComponents.concat(parentSubComponents)
    }
    // initializeSubComponents is protected, the following is an "escape hatch"
    this.parent['initializeSubComponents'](allSubComponents);
    allSubComponents.forEach((subComponent) => {
      // this is the part that copies references between the page object and it's parent
      // this allows the model to emulate the React Fragment `<>`
      const descriptor = Object.getOwnPropertyDescriptor(this.parent, subComponent.name);
      if (typeof descriptor === 'undefined') {
        // TODO uniquely identify each error. think about how languages throw errors
        // This should be some kind of "Unreachable" error:
        //     Meaning logic present in the same function should be guarding us against throwing this error
        // In this example, `this.parent.initializeSubComponents` ensures the components are present
        throw new Error(`subComponent ${subComponent.name} not present in parent object`);
      }
      Object.defineProperty(this, subComponent.name, {
        value: descriptor,
        // it's a good thing these are readonly because idk what would happen if we tried to delete one
        writable: false,
      });
    });
  }
}
