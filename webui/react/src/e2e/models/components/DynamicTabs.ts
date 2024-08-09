import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Pivot } from 'e2e/models/common/hew/Pivot';

/**
 * Represents the DynamicTabs component in src/components/DynamicTabs.tsx
 */
export class DynamicTabs extends BaseReactFragment {
  readonly pivot = new Pivot({ parent: this });
}
