import { NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { Switch } from 'e2e/models/common/ant/Switch';
import { Label } from 'e2e/models/common/hew/Label';

/**
 * Represents the Toggle component in hew/src/kit/Toggle.tsx
 */
export class Toggle extends NamedComponent {
  override defaultSelector = '[class^="Row_row"]';
  readonly label = new Label({
    parent: this,
  });
  readonly switch = new Switch({
    parent: this,
  });
}
