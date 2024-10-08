import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseReactFragment } from 'playwright-page-model-base/BaseReactFragment';

import { Modal } from 'e2e/models/common/ant/Modal';
import { Notification } from 'e2e/models/common/ant/Notification';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { Select } from 'e2e/models/common/hew/Select';
import { Toggle } from 'e2e/models/common/hew/Toggle';
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
  readonly archived = new BaseComponent({
    parent: this,
    selector: '[data-testid="archived"]',
  });
  readonly name = new BaseComponent({
    parent: this,
    selector: '[data-testid="name"]',
  });
  readonly archivedIcon = new BaseComponent({
    parent: this,
    selector: '[aria-label="Checkmark"]',
  });
}

/**
 * Represents the ModelActionDropdown from src/components/ModelActionDropdown.tsx
 */
class ModelActionDropdown extends DropdownMenu {
  readonly delete = this.menuItem('delete-model');
  readonly switchArchived = this.menuItem('switch-archived');
  readonly move = this.menuItem('move-model');
}

class ModelDeleteModal extends Modal {
  readonly deleteButton = new BaseComponent({
    parent: this,
    selector: '.ant-btn-dangerous',
  });
}

class ModelMoveModal extends Modal {
  readonly workspaceSelect = new Select({
    parent: this,
    selector: 'input[id="workspace"]',
  });
}

/* Represents the ModelRegistry component in src/components/ModelRegistry.tsx
 */
export class ModelRegistry extends BaseReactFragment {
  readonly showArchived = new Toggle({
    parent: this,
  });
  readonly newModelButton = new BaseComponent({
    parent: this,
    selector: '[test-id="new-model-button"]',
  });
  readonly modelCreateModal = new ModelCreateModal({
    root: this.root,
  });
  readonly modelMoveModal = new ModelMoveModal({
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
  readonly noModelsMessage = new BaseComponent({
    parent: this,
    selector: '[data-testid="no-models-registered"]',
  });
}
