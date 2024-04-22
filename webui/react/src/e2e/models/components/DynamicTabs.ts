import { BaseReactFragment } from 'e2e/models/BaseComponent';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Returns a representation of the DynamicTabs component.
 * This constructor represents the contents in src/components/DynamicTabs.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this DynamicTabs
 */
export class DynamicTabs extends BaseReactFragment {
  readonly pivot = new Pivot({ parent: this });
}
