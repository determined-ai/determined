import { type Locator } from '@playwright/test';

// type that has a function "locator" which takes a string and gives a Locator
type implementsLocator = {locator: (arg0: string) => Locator}
// type that has a function "locate" which takes a string and gives a Locator
type implementsLocate = {locate: () => implementsLocator}

export class hasSubelements implements implementsLocate {
    // This class exists so we can DRY `_initialize_subelements`
    _initialize_subelements(subelements: Subelement[]) {
        subelements.forEach(subelement => {
            Object.defineProperty(this, subelement.name, new BaseComponent({
                parent: this,
                selector: subelement.selector,
                subelements: subelement.subelements
            }))
        });
    }
    
    // idenity functions that should be reimplemented by BaseComponent and BasePage
    static #notImplemented() : never {
        throw new Error("not Implemented")
    }
    locate(): implementsLocator {return hasSubelements.#notImplemented()}

    // shorthand functions

    // `loc = this.locate` right here will bind it to the one in this class. we want to use the reimplementation
    loc(): implementsLocator {return this.locate()}
}

export interface BaseComponentProps {
    parent: hasSubelements
    selector?: string
    subelements?: Subelement[]
}

export interface Subelement {
    name: string
    type: typeof BaseComponent
    selector: string
    subelements?: Subelement[]
}

export class BaseComponent extends hasSubelements {
    /*
    isDisplayed needs to check all parents, might need to find a way to play nice with expect.to.be.displayed somehow
    how to waitToBeDisplayed()?

    consider proxy
    https://stackoverflow.com/questions/1529496/is-there-a-javascript-equivalent-of-pythons-getattr-method
    */

    readonly defaultSelector: undefined | string;
    
    readonly _selector: string;
    _parent: hasSubelements;
    _locator: Locator | undefined;

    constructor({parent, selector, subelements}: BaseComponentProps) {
        super()
        if (typeof this.defaultSelector === "undefined") {
            throw new Error('defaultSelector is undefined')
        }
        this._selector = selector || this.defaultSelector;
        this._parent = parent;
        
        if (typeof subelements !== "undefined") {
            this._initialize_subelements(subelements)
        }
    }

    override locate(): Locator {
        if (typeof this._selector === "undefined") {
            throw new Error('selector is undefined')
        }
        if (!this._locator) {
            this._locator = this._parent.locate().locator(this._selector);
        }
        return this._locator
    }
}