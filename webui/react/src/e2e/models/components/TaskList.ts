import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Modal } from 'e2e/models/common/ant/Modal';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { JupyterLabModal } from 'e2e/models/components/JupyterLabModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';
import { TaskAction } from 'types';

class TaskHeadRow extends HeadRow {}
class TaskRow extends Row {
  readonly actions = new TaskActionDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: '[data-testid="actions"]',
    }),
    root: this.root,
  });
  readonly state = new BaseComponent({
    parent: this,
    selector: '[data-testid="state"]',
  });
}

/**
 * Represents the TaskActionDropdown from src/components/TaskActionDropdown.tsx
 */
class TaskActionDropdown extends DropdownMenu {
  readonly kill = this.menuItem(TaskAction.Kill);
  readonly viewLogs = this.menuItem(TaskAction.ViewLogs);
  readonly connect = this.menuItem(TaskAction.Connect);
}

class TaskKillModal extends Modal {
  readonly killButton = new BaseComponent({
    parent: this,
    selector: '.ant-btn-dangerous',
  });
}

/**
 * Represents the TaskList in src/components/TaskList.tsx
 */
export class TaskList extends BaseReactFragment {
  readonly jupyterLabButton = new BaseComponent({
    parent: this,
    selector: '[data-testid="jupyter-lab-button"]',
  });
  readonly jupyterLabModal = new JupyterLabModal({
    root: this.root,
  });
  readonly table = new InteractiveTable({
    parent: this,
    tableArgs: {
      attachment: '[data-testid="table"]',
      headRowType: TaskHeadRow,
      rowType: TaskRow,
    },
  });
  readonly taskKillModal = new TaskKillModal({
    root: this.root,
  });
}
