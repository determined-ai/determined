import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

/**
 * Represents the LogViewer component in hew/src/kit/LogViewer.tsx
 */
export class LogViewer extends NamedComponent {
  readonly defaultSelector = '[class^="LogViewer_base"]';
  readonly logEntry = new BaseComponent({
    parent: this,
    selector: '[class^="LogViewerEntry_base"]',
  });
}
