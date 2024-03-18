import { BaseComponent, BaseComponentProps } from '../BaseComponent';

export class DeterminedAuth extends BaseComponent {
    static defaultSelector: string = 'Form[data-test=authForm]';
    override readonly defaultSelector: string = DeterminedAuth.defaultSelector;
    readonly form: BaseComponent
    readonly docs: BaseComponent
    
    constructor({parent, selector, subelements}: BaseComponentProps) {
        super({parent: parent, selector: selector, subelements: subelements})
        this.form = new BaseComponent({parent: this, selector: 'form', subelements: [
            {name: 'username', type: BaseComponent, selector: 'input[data-testid=username]'},
            {name: 'password', type: BaseComponent, selector: 'input[data-testid=password]'},
            {name: 'submit', type: BaseComponent, selector: 'button[data-testid=submit]'},
            {name: 'error', type: BaseComponent, selector: 'p[data-testid=error]'},
        ]})
        
        this.docs = new BaseComponent({parent: this, selector: 'link[data-testid=docs]'})
    }
}