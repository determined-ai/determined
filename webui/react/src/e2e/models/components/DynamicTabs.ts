import { BaseReactFragment } from 'e2e/models/BaseComponent';
import { Pivot } from 'e2e/models/hew/Pivot';

/**
 * Represents the DynamicTabs component in src/components/DynamicTabs.tsx
 */
export class DynamicTabs extends BaseReactFragment {
  readonly pivot = new Pivot({ parent: this });
}
