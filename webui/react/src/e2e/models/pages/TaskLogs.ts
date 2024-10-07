import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { LogViewer } from 'e2e/models/common/hew/LogViewer';

/**
 * Represents the TaskLogs page from src/pages/TaskLogs.tsx
 */
export class TaskLogs extends DeterminedPage {
  readonly title = '';
  readonly url = /[^/]+\/[0-9a-fA-F-]+\/logs/;
  readonly logViewer = new LogViewer({
    parent: this,
  });
}
