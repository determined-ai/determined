import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/SplitPane.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this SplitPane
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class SplitPane extends NamedComponent {
  readonly defaultSelector = '[class^="SplitPane_base"]';
  readonly initial: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'div[style*="display: initial"]',
  });
  // TODO left pane and right pane
}
