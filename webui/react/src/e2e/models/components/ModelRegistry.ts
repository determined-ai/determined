import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Modal } from 'e2e/models/common/ant/Modal';
import { Notification } from 'e2e/models/common/ant/Notification';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { ModelCreateModal } from 'e2e/models/components/ModelCreateModal';
import { HeadRow, InteractiveTable, Row } from 'e2e/models/components/Table/InteractiveTable';

class ModelHeadRow extends HeadRow {}
class ModelRow extends Row {
  readonly actions = new ModelActionDropdown({
    clickThisComponentToOpen: new BaseComponent({
      parent: this,
      selector: '[data-testid="actions"]',
    }),
    root: this.root,
  });
}

/**
 * Represents the ModelActionDropdown from src/components/ModelActionDropdown.tsx
 */
class ModelActionDropdown extends DropdownMenu {
  readonly delete = this.menuItem('delete-model');
}

class ModelDeleteModal extends Modal {
  readonly deleteButton = new BaseComponent({
    parent: this,
    selector: '.ant-btn-dangerous',
  });
}

/* Represents the ModelRegistry component in src/components/ModelRegistry.tsx
 */
export class ModelRegistry extends BaseReactFragment {
  readonly newModelButton = new BaseComponent({
    parent: this,
    selector: '[data-testid="new-model-button"]',
  });
  readonly modelCreateModal = new ModelCreateModal({
    root: this.root,
  });
  readonly table = new InteractiveTable({
    parent: this,
    tableArgs: {
      attachment: '[data-testid="table"]',
      headRowType: ModelHeadRow,
      rowType: ModelRow,
    },
  });
  readonly notification = new Notification({
    parent: this.root,
  });
  readonly modelDeleteModal = new ModelDeleteModal({
    root: this.root,
  });
}
