import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';

import ExperimentEditModal from './ExperimentEditModal';
import ExperimentMoveModal from './ExperimentMoveModal';
import HyperparameterSearchModal from './HyperparameterSearchModal';

/**
 * Represents the ExperimentActionDropdown component in src/components/ExperimentActionDropdown.tsx
 */

export class ExperimentActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('Edit');
  readonly pause = this.menuItem('Pause');
  readonly resume = this.menuItem('Resume');
  readonly hpSearch = this.menuItem('Hyperparameter Search');

  readonly editModal = new ExperimentEditModal({
    root: this.root,
  });
  readonly moveModal = new ExperimentMoveModal({
    root: this.root,
  });
  readonly hpSearchModal = new HyperparameterSearchModal({
    root: this.root,
  });
}
