import { BaseComponent, BaseComponentProps } from 'e2e/models/BaseComponent';

export class DeterminedAuth extends BaseComponent {
  static defaultSelector: string = 'Form[data-test=authForm]';
  override readonly defaultSelector: string = DeterminedAuth.defaultSelector;
  readonly form: BaseComponent;
  readonly docs: BaseComponent;

  /**
   * Returns a representation of the DeterminedAuth component.
   *
   * @remarks
   * This constructor represents the contents in src/components/DeterminedAuth.tsx.
   *
   * @param {Object} obj
   * @param {implementsGetLocator} obj.parent - The parent used to locate this DeterminedAuth
   * @param {string} [obj.selector] - Used instead of `defaultSelector`
   * @param {SubComponent[]} [obj.subComponents] - SubComponents to initialize at runtime
   */
  constructor({ parent, selector, subComponents }: BaseComponentProps) {
    super({ parent: parent, selector: selector, subComponents: subComponents });
    this.form = new BaseComponent({
      parent: this,
      selector: 'form',
      subComponents: [
        { name: 'username', selector: 'input[data-testid=username]', type: BaseComponent },
        { name: 'password', selector: 'input[data-testid=password]', type: BaseComponent },
        { name: 'submit', selector: 'button[data-testid=submit]', type: BaseComponent },
        { name: 'error', selector: 'p[data-testid=error]', type: BaseComponent },
      ],
    });

    this.docs = new BaseComponent({ parent: this, selector: 'link[data-testid=docs]' });
  }
}
