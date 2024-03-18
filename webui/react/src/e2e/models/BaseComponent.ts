import { type Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export interface BaseComponentProps {
    parent: BaseComponent | BasePage
    selector?: string
    subelements?: Subelement[]
}

export interface Subelement {
    name: string
    type: typeof BaseComponent
    selector: string
    subelements?: Subelement[]
}

export class BaseComponent {
    /*
    isDisplayed needs to check all parents, might need to find a way to play nice with expect.to.be.displayed somehow
    how to waitToBeDisplayed()?

    consider proxy
    https://stackoverflow.com/questions/1529496/is-there-a-javascript-equivalent-of-pythons-getattr-method
    */

    readonly defaultSelector: undefined | string;
    
    readonly _selector: string;
    _parent: BaseComponent | BasePage;
    _locator: Locator | undefined;

    constructor({parent, selector, subelements}: BaseComponentProps) {
        if (typeof this.defaultSelector === "undefined") {
            throw new Error('defaultSelector is undefined')
        }
        this._selector = selector || this.defaultSelector;
        this._parent = parent;
        
        if (typeof subelements !== "undefined") {
            this._initialize_subelements(subelements)
        }
    }

    _initialize_subelements(subelements: Subelement[]) {
        if (typeof this._parent !== typeof BaseComponent) {
            subelements.forEach(subelement => {
                Object.defineProperty(this, subelement.name, new BaseComponent({
                parent: this,
                    selector: subelement.selector,
                    subelements: subelement.subelements
                }))
            });
        }
    }

    locate(): Locator {
        if (typeof this._selector === "undefined") {
            throw new Error('selector is undefined')
        }
        if (!this._locator) {
            this._locator = this._parent.locate().getByTestId(this._selector);
        }
        return this._locator
    }

    loc = this.locate
}