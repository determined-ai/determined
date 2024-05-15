import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Message component from Hew.
 * This constructor represents the contents in hew/src/kit/Message.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Message
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Message extends NamedComponent {
  readonly defaultSelector = '[class^="Message_base"]';

  readonly icon = new BaseComponent({
    parent: this,
    selector: '[class^="Icon_base"]',
  });
  readonly title = new BaseComponent({
    parent: this,
    selector: 'h1',
  });
  readonly description = new BaseComponent({
    parent: this,
    selector: '[class^="Message_description"]',
  });
  readonly link = new BaseComponent({
    parent: this,
    selector: 'a',
  });
}
