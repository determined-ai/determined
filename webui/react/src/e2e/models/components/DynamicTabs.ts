import { NamedComponent } from 'e2e/models/BaseComponent';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of the DynamicTabs component.
 * This constructor represents the contents in src/components/DynamicTabs.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this DynamicTabs
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class DynamicTabs extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="dynamicTabs"]';
  readonly pivot = new Pivot({ parent: this })
}
