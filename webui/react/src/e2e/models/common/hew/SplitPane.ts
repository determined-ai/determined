import { BaseComponent, NamedComponent } from 'e2e/models/common/base/BaseComponent';

/**
 * Represents the SplitPane component from hew/src/kit/SplitPane.tsx
 */
export class SplitPane extends NamedComponent {
  readonly defaultSelector = '[class^="SplitPane_base"]';
  readonly initial = new BaseComponent({
    parent: this,
    selector: 'div[style*="display: initial"]',
  });
  // TODO left pane and right pane
}
