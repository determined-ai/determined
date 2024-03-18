import { BaseComponent, BaseComponentProps, Subelement } from '../BaseComponent';
// TODO Unit tests

export class Page extends BaseComponent {
    override defaultSelector: string = '';
    
    constructor({parent, selector, subelements}: BaseComponentProps) {
        // call the super like normal, but without subelements. we'll set them manually
        super({parent: parent, selector: selector})
        let parent_subelements: Subelement[] = [
            // TODO put spinners and other things from Page in here
        ]
        // initialize subelements under normal base page as well
        // these aren't exact copies so maby
        this._parent._initialize_subelements(parent_subelements)
        if (typeof subelements !== 'undefined') {
            this._parent._initialize_subelements(subelements)
            subelements?.forEach(subelement => {
                // this is the part that copies references between the page object and it's parent
                // this allows the model to emulate the React Fragment `<>`
                const descriptor = Object.getOwnPropertyDescriptor(this._parent, subelement.name)
                if (typeof descriptor === "undefined") {
                    // TODO uniquely identify each error. think about how languages throw errors
                    // This should be some kind of "Unreachable" error:
                    //     Meaning logic present in the same function should be guarding us against throwing this error
                    // In this example, `this._parent._initialize_subelements` ensures the elements are present
                    throw new Error(`subelement ${subelement.name} not present in parent object`)
                }
                Object.defineProperty(this, subelement.name, {
                    value: descriptor,
                    writable: false // it's a good thing these are readonly because idk what would happen if we tried to delete one
                })
            })
        }
    }
}