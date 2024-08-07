import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the Message component from hew/src/kit/Message.tsx
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
