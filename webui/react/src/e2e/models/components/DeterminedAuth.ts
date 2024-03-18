import { BaseComponent, BaseComponentProps } from 'e2e/models/BaseComponent';

export class DeterminedAuth extends BaseComponent {
    static defaultSelector: string = 'Form[data-test=authForm]';
    override readonly defaultSelector: string = DeterminedAuth.defaultSelector;
    readonly form: BaseComponent;
    readonly docs: BaseComponent;

    constructor({ parent, selector, subelements }: BaseComponentProps) {
        super({ parent: parent, selector: selector, subelements: subelements });
        this.form = new BaseComponent({
            parent: this,
selector: 'form',
subelements: [
                { name: 'username', selector: 'input[data-testid=username]', type: BaseComponent },
                { name: 'password', selector: 'input[data-testid=password]', type: BaseComponent },
                { name: 'submit', selector: 'button[data-testid=submit]', type: BaseComponent },
                { name: 'error', selector: 'p[data-testid=error]', type: BaseComponent },
            ],
        });

        this.docs = new BaseComponent({ parent: this, selector: 'link[data-testid=docs]' });
    }
}
